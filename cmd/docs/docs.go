// Package docs generates reference documentation for gnux and every tool it
// ships, in each format cobra supports (markdown, man, yaml, reST). It is used
// only by the tools/gendocs generator, so the go-md2man dependency it pulls in
// never links into the gnux binary itself.
package docs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/cmd"
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// ManDate is stamped into generated man pages. It is fixed (rather than
// time.Now) so committed documentation is reproducible; release tooling can
// override it before generating.
var ManDate = time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

// Formats lists the output formats in the order GenerateAll writes them. Each
// is also the name of its directory under the docs base.
var Formats = []string{"markdown", "man", "yaml", "rest"}

// page is one documentation topic - gnux itself, the guide, a tool, or one of a
// tool's flags - ready to render in any format.
type page struct {
	dir      string            // directory under <format>/, "" for the top level
	base     string            // file name without extension
	manTitle string            // man page title, e.g. "rmx" or "rmx-force"
	manual   string            // man page manual name
	command  *cobra.Command    // drives the man, yaml and reST output
	markdown func() string     // hand-rendered markdown; nil means cobra's
	links    map[string]string // cobra's link name for a child page -> its real path, both without extension
}

// GenerateAll writes the complete documentation set under base, grouped by
// format so each format directory is self-contained:
//
//	base/<format>/index.*               gnux reference (entry page)
//	base/<format>/guide.*               installation/usage guide
//	base/<format>/<tool>/index.*        per-tool reference
//	base/<format>/<tool>/flags/<flag>.* per-flag reference, for tools that have one
func GenerateAll(base string) error {
	for _, f := range Formats {
		if err := Generate(base, f); err != nil {
			return err
		}
	}
	return nil
}

// Generate writes the documentation set for a single format under base/<format>.
// A format directory is rebuilt from scratch, so pages for removed tools or
// flags never linger.
func Generate(base, format string) error {
	render, ext, err := renderer(format)
	if err != nil {
		return err
	}
	root := filepath.Join(base, format)
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("clear docs dir %q: %w", root, err)
	}
	for _, p := range pages() {
		if err := writeFile(filepath.Join(root, p.dir), p.base+ext, func(w io.Writer) error {
			return render(p, w)
		}); err != nil {
			return err
		}
	}
	return nil
}

// renderer returns the page renderer and file extension for format.
func renderer(format string) (func(page, io.Writer) error, string, error) {
	switch format {
	case "markdown":
		return renderMarkdown, ".md", nil
	case "man":
		return renderMan, ".1", nil
	case "yaml":
		return func(p page, w io.Writer) error { return doc.GenYaml(p.command, w) }, ".yaml", nil
	case "rest":
		return renderReST, ".rst", nil
	default:
		return nil, "", fmt.Errorf("unknown format %q (want one of: %s)", format, strings.Join(Formats, ", "))
	}
}

// pages builds every page of the reference, in a stable order.
func pages() []page {
	discard := core.Discard()

	root := cmd.NewRoot(discard)
	disableAutoGen(root)
	links := map[string]string{}
	for _, t := range cmd.Commands() {
		// cobra names a child's page "<parent>_<child>"; ours live at <child>/index.
		links[common.AppName+"_"+t.Name()] = t.Name() + "/index"
	}

	out := []page{
		{base: "index", manTitle: common.AppName, manual: common.AppName + " manual", command: root, links: links},
		{base: "guide", manTitle: common.AppName + "-guide", manual: common.AppName + " manual", command: guideCommand()},
	}

	for _, t := range cmd.Commands() {
		name := t.Name()
		flags := t.Command(discard).Flags()
		out = append(out, page{
			dir:      name,
			base:     "index",
			manTitle: name,
			manual:   common.AppName + " manual",
			command:  topicCommand(toolTopicMeta(t), func(fs *pflag.FlagSet) { fs.AddFlagSet(flags) }),
			markdown: func() string { return toolMarkdown(t, flags) },
		})
		for _, topic := range t.Topics {
			fs := pflag.NewFlagSet(topic.Meta.Name, pflag.ContinueOnError)
			topic.Register(fs)
			out = append(out, page{
				dir:      filepath.Join(name, "flags"),
				base:     topic.Meta.Name,
				manTitle: name + "-" + topic.Meta.Name,
				manual:   name + " flag reference",
				command:  topicCommand(topic.Meta, topic.Register),
				markdown: func() string { return topicMarkdown(topic.Meta, fs, 2) },
			})
		}
	}
	return out
}

// toolTopicMeta is a tool's metadata as the man, yaml and reST pages render
// it: the exit statuses, related tools and terminal help are folded into the
// description, since cobra/doc has no section for them.
func toolTopicMeta(t tool.Spec) docmeta.Meta {
	m := t.Meta
	var b strings.Builder
	b.WriteString(m.Long)
	b.WriteString("\n\nEXIT STATUS.\n")
	for _, s := range t.ExitStatus() {
		fmt.Fprintf(&b, "  %s  %s\n", s.Code, s.Meaning)
	}
	fmt.Fprintf(&b, "\nHELP. %s --help shows the options and the first examples; %s help %s prints "+
		"the full manual, and %s help %s OPTION explains one option.", t.DiagName(), common.AppName, t.Name(), common.AppName, t.Name())
	if len(m.SeeAlso) > 0 {
		b.WriteString("\n\nSEE ALSO. ")
		b.WriteString(strings.Join(m.SeeAlso, ", "))
		b.WriteString(".")
	}
	m.Long = b.String()
	return m
}

// topicCommand builds a standalone cobra command for a tool or flag topic so
// cobra/doc can render it in man, yaml and reST. Permissions and use cases are
// folded into the description, and the bash examples into the example block,
// so those formats carry the same information as the markdown page.
func topicCommand(m docmeta.Meta, register func(*pflag.FlagSet)) *cobra.Command {
	long := m.Long
	if m.Permissions != "" {
		long += "\n\nPERMISSIONS. " + m.Permissions
	}
	if len(m.UseCases) > 0 {
		long += "\n\nUSE CASES.\n"
		for _, u := range m.UseCases {
			long += "  - " + u + "\n"
		}
	}

	c := &cobra.Command{
		Use:                   m.Use,
		Short:                 m.Short,
		Long:                  strings.TrimRight(long, "\n"),
		Example:               docmeta.ExampleBlock(m.Examples),
		DisableFlagsInUseLine: true,
		// A no-op Run marks the command runnable so the synopsis is emitted.
		Run: func(*cobra.Command, []string) {},
	}
	c.DisableAutoGenTag = true
	register(c.Flags())
	return c
}

// disableAutoGen turns off cobra's "auto generated on <date>" footer for c and
// its children so the output is deterministic and safe to commit.
func disableAutoGen(c *cobra.Command) {
	c.DisableAutoGenTag = true
	for _, child := range c.Commands() {
		disableAutoGen(child)
	}
}

func renderMan(p page, w io.Writer) error {
	header := &doc.GenManHeader{
		Title:   strings.ToUpper(p.manTitle),
		Section: "1",
		Source:  common.AppName,
		Manual:  p.manual,
		Date:    &ManDate,
	}
	return doc.GenMan(p.command, header, w)
}

func renderReST(p page, w io.Writer) error {
	return doc.GenReSTCustom(p.command, w, func(name, ref string) string {
		if target, ok := p.links[ref]; ok {
			return fmt.Sprintf("`%s <%s.rst>`_", name, target)
		}
		return fmt.Sprintf(":ref:`%s <%s>`", name, ref)
	})
}

// writeFile renders gen into dir/filename, creating directories as needed.
func writeFile(dir, filename string, gen func(io.Writer) error) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create docs dir %q: %w", dir, err)
	}
	buf := new(bytes.Buffer)
	if err := gen(buf); err != nil {
		return fmt.Errorf("render %q: %w", filename, err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

// toolNames returns every registered tool name, for cross-referencing.
func toolNames() []string {
	var names []string
	for _, t := range cmd.Tools() {
		names = append(names, t.Name())
	}
	return names
}
