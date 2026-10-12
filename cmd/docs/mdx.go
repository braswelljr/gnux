package docs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/cmd"
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/tool"
)

// GenerateMDX writes the website's documentation content into dir (the
// Fumadocs content root, web/content/docs) from the same registry and metadata
// as the reference formats:
//
//	dir/index.mdx          overview and tool table
//	dir/guide.mdx          requirements, installation, usage, workflows
//	dir/library.mdx        using the engines from Go
//	dir/shell.mdx          the gnux shell
//	dir/tools/<tool>/index.mdx    one page per tool
//	dir/tools/<tool>/<flag>.mdx   one page per option, with its usage and examples
//	dir/meta.json, dir/tools/meta.json, dir/tools/<tool>/meta.json   navigation order
//
// In the sidebar the tools sit under a "Tools" heading rather than inside a
// collapsible folder; each tool is a folder whose children are its options.
//
// extra pages are hand-written markdown (see HandwrittenPages): they are added
// after the generated ones, each in the folder its Dir names.
//
// The directory is rebuilt from scratch. Pages carry title/description
// frontmatter and use only standard markdown, so they need no custom
// components.
func GenerateMDX(dir string, extra ...Page) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clear docs dir %q: %w", dir, err)
	}
	discard := core.Discard()

	type doc struct {
		sub, name, title, description, body string
	}
	docs := []doc{
		{"", "index", common.AppName, "The GNU core utilities in one cross-platform binary, a Go library, and a portable shell.", mdxIndex()},
		{"", "guide", "Guide", "Requirements, installation, usage and workflows.", mdxGuide()},
		{"", "library", "Go library", "Use the gnux engines, tools and shell from a Go program.", mdxLibrary},
	}

	sh := cmd.ShellSpec()
	docs = append(docs, doc{"", "shell", "gnux shell", sh.Meta.Short, mdxTool(sh, sh.Command(discard).Flags(), nil)})

	var names []string
	toolFolders := map[string][]string{} // tool -> its option pages, in order
	for _, t := range cmd.Tools() {
		names = append(names, t.Name())
		sub := filepath.Join("tools", t.Name())
		flags := t.Command(discard).Flags()
		options := flagPages(t, flags)
		docs = append(docs, doc{sub, "index", t.Name(), t.Meta.Short, mdxTool(t, flags, options)})
		toolFolders[t.Name()] = []string{}
		for _, o := range options {
			toolFolders[t.Name()] = append(toolFolders[t.Name()], o.name)
			docs = append(docs, doc{sub, o.name, o.title, o.description, o.body})
		}
	}

	// In the sidebar a "---Title---" entry is a small heading, and the pages
	// listed after it sit beneath it as an open list rather than a collapsible
	// folder. The hand-written folders (the guides) come first, each page named
	// by its path so the folder's landing page is not repeated under a heading
	// of its own name; then the tools, laid out with "...tools".
	order := []string{"index", "guide", "shell", "library"}
	var sections []string            // extra folders, in order of first appearance
	folders := map[string][]string{} // extra folder -> its pages, in order
	for _, e := range extra {
		// Hand-written prose is already marked up; it only needs MDX escaping.
		content := mdxFrontmatter(e.Title, e.Description) + mdxEscape(stripTitle(e.Markdown))
		if err := writeFile(filepath.Join(dir, e.Dir), e.Name+".mdx", func(w io.Writer) error {
			_, err := io.WriteString(w, content)
			return err
		}); err != nil {
			return err
		}
		if e.Dir == "" {
			order = append(order, e.Name)
			continue
		}
		if _, seen := folders[e.Dir]; !seen {
			sections = append(sections, e.Dir)
			folders[e.Dir] = nil
		}
		// A folder's index page is its landing page, reached by its URL; under
		// a heading of the same name it would only repeat it.
		if e.Name != "index" {
			folders[e.Dir] = append(folders[e.Dir], e.Name)
		}
	}
	for _, folder := range sections {
		order = append(order, "---"+folderTitle(folder)+"---")
		for _, page := range folders[folder] {
			order = append(order, folder+"/"+page)
		}
	}
	order = append(order, "---Tools---", "...tools")
	for folder, pages := range folders {
		if err := writeMeta(filepath.Join(dir, folder), folderTitle(folder), pages); err != nil {
			return err
		}
	}

	for _, d := range docs {
		content := mdxFrontmatter(d.title, d.description) + mdxEscape(string(wrapCode([]byte(d.body))))
		if err := writeFile(filepath.Join(dir, d.sub), d.name+".mdx", func(w io.Writer) error {
			_, err := io.WriteString(w, content)
			return err
		}); err != nil {
			return err
		}
	}

	if err := writeMeta(dir, common.AppName, order); err != nil {
		return err
	}
	for name, pages := range toolFolders {
		if err := writeMeta(filepath.Join(dir, "tools", name), name, pages); err != nil {
			return err
		}
	}
	return writeMeta(filepath.Join(dir, "tools"), "Tools", names)
}

// writeMeta writes a Fumadocs meta.json fixing a folder's title and page order.
func writeMeta(dir, title string, pages []string) error {
	return writeFile(dir, "meta.json", func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Title string   `json:"title"`
			Pages []string `json:"pages"`
		}{title, pages})
	})
}

// mdxFrontmatter renders the YAML frontmatter block. JSON strings are valid
// YAML scalars, which takes care of quoting.
func mdxFrontmatter(title, description string) string {
	t, _ := json.Marshal(title)
	d, _ := json.Marshal(description)
	return fmt.Sprintf("---\ntitle: %s\ndescription: %s\n---\n\n", t, d)
}

// mdxEscape makes markdown prose safe for MDX, which would otherwise read "{"
// as the start of a JavaScript expression and "<" as the start of a JSX tag.
// Fenced blocks and inline code are literal already and are left alone.
func mdxEscape(md string) string {
	lines := strings.Split(md, "\n")
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		parts := strings.Split(line, "`")
		for j := 0; j < len(parts); j += 2 {
			parts[j] = mdxReplacer.Replace(parts[j])
		}
		lines[i] = strings.Join(parts, "`")
	}
	return strings.Join(lines, "\n")
}

var mdxReplacer = strings.NewReplacer("{", `\{`, "}", `\}`, "<", `\<`)

// mdxIndex is the documentation landing page: what gnux is and what it ships.
func mdxIndex() string {
	var b strings.Builder
	b.WriteString(guideIntro + "\n\n")

	b.WriteString("## Three ways to use it\n\n")
	b.WriteString("- **Commands** - each tool is a command of its own, and a subcommand of the one binary. See the [guide](/docs/guide).\n")
	b.WriteString("- **Go library** - every tool's engine is an importable package. See [Go library](/docs/library).\n")
	b.WriteString("- **Portable shell** - scripts that run the same on every platform. See [gnux shell](/docs/shell).\n\n")
	b.WriteString("```bash\nrmx -rf build/                  # a command of its own\ngnux rmx -rf build/             # a subcommand\ngnux shell -c 'rmx -rf build/'  # a line of a portable script\n```\n\n")

	b.WriteString("## Tools\n\n| Tool | GNU counterpart | What it does |\n| --- | --- | --- |\n")
	for _, t := range cmd.Tools() {
		fmt.Fprintf(&b, "| [%s](/docs/tools/%s) | `%s` | %s |\n", t.Name(), t.Name(), t.GNU, strings.TrimSuffix(t.Meta.Short, "."))
	}
	b.WriteString("\nTry any of them in the [playground](/playground) - it runs the real tools in your browser.\n")
	return b.String()
}

// mdxGuide renders the guide sections with real headings, then the examples.
func mdxGuide() string {
	var b strings.Builder
	b.WriteString(guideIntro + "\n\n")
	for _, s := range guideSections {
		fmt.Fprintf(&b, "## %s\n\n", s.title)
		for _, p := range s.body {
			b.WriteString(p + "\n\n")
		}
	}
	b.WriteString("## Examples\n\n```bash\n")
	for _, line := range strings.Split(guideExample, "\n") {
		b.WriteString(strings.TrimPrefix(line, "  ") + "\n")
	}
	b.WriteString("```\n")
	return b.String()
}

// mdxTool renders one tool (or gnux command) page; the page title comes from
// the frontmatter, so sections start at level two.
//
// options are the tool's option pages, listed with links; a gnux command such
// as the shell has none and documents its flags in place.
func mdxTool(t tool.Spec, flags *pflag.FlagSet, options []flagPage) string {
	var b strings.Builder
	m := t.Meta

	use := m.Use
	if t.GNU == "" {
		use = common.AppName + " " + use
	} else {
		fmt.Fprintf(&b, "Mirrors GNU `%s`. Run it as `%s`, or as `%s %s`.\n\n", t.GNU, m.Name, common.AppName, m.Name)
	}
	fmt.Fprintf(&b, "## Synopsis\n\n```bash\n%s\n```\n\n", use)
	b.WriteString(gnuReference(t, "##"))
	b.WriteString(topicBody(m, flags, 2, true))
	if len(options) > 0 {
		b.WriteString("\n## Options\n\nEach option has a page of its own, with its usage and examples.\n\n")
		for _, o := range options {
			fmt.Fprintf(&b, "- [%s](/docs/tools/%s/%s): %s\n", o.title, t.Name(), o.name, o.description)
		}
	}
	b.WriteString("\n" + exitStatusMarkdown(t, "##"))
	b.WriteString("\n" + terminalHelpMarkdown(t, "##"))
	if len(m.SeeAlso) > 0 {
		b.WriteString("\n## See also\n\n")
		for _, r := range m.SeeAlso {
			if target := strings.TrimPrefix(r, common.AppName+" "); target != r {
				fmt.Fprintf(&b, "- [%s](/docs/%s)\n", r, target)
			} else {
				fmt.Fprintf(&b, "- [%s](/docs/tools/%s)\n", r, r)
			}
		}
	}

	return b.String()
}

// mdxLibrary is the hand-written page on using gnux from Go.
const mdxLibrary = `gnux is a Go module as well as a binary. Everything the commands do is available to a Go ` +
	`program at three levels: a single tool's engine, a tool by name, or a whole script.

## Requirements

A Go module on Go 1.25 or newer. The engines depend only on the standard library and ` + "`github.com/mattn/go-isatty`" + `; cobra is compiled only into the command-line packages under ` + "`cmd/`" + `.

` + "```bash" + `
go get github.com/braswelljr/gnux
` + "```" + `

## An engine

Each tool has an engine package under ` + "`core/`" + ` named after the GNU tool: ` + "`core/rm`, `core/cp`, `core/ls`" + ` and so on. An engine takes the options the flags would set and the streams to read and write; the zero value of its ` + "`Options`" + ` is the tool with no flags.

` + "```go" + `
package main

import (
	"errors"
	"log"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rm"
)

func main() {
	// rmx -rf build dist
	engine := rm.New(rm.Options{Recursive: true, Force: true}, core.Stdio())

	err := engine.Run([]string{"build", "dist"})
	if errors.Is(err, core.ErrFailed) {
		// Some paths could not be removed; the diagnostics already went to stderr.
		log.Fatal("cleanup incomplete")
	}
}
` + "```" + `

## Streams

` + "`core.Stdio()`" + ` wires an engine to the process's own streams and ` + "`core.Discard()`" + ` silences it. To capture output, build a ` + "`core.Streams`" + ` from any reader and writers:

` + "```go" + `
var out, errs bytes.Buffer
streams := core.Streams{In: strings.NewReader(""), Out: &out, Err: &errs}
` + "```" + `

## Errors

Every engine follows one convention.

| Returned error | Meaning |
| --- | --- |
| ` + "`nil`" + ` | success |
| ` + "`core.ErrFailed`" + ` | it ran, something failed, and the diagnostics are already written |
| ` + "`*core.UsageError`" + ` | the arguments were wrong |
| ` + "`*core.ExitError`" + ` | a specific exit status, for tools whose status carries meaning |

## A tool by name

` + "`cmd.RunTool`" + ` runs a tool from its name and arguments, exactly as the command line would, and returns the exit status instead of exiting. It accepts the gnux name or the GNU one.

` + "```go" + `
status, ok := cmd.RunTool("rmx", []string{"-rf", "build"}, core.Stdio())
` + "```" + `

## A script

` + "`cmd.NewShell`" + ` builds a [gnux shell](/docs/shell) whose commands are the tools. It runs a script in-process, the same on every platform.

` + "```go" + `
sh := cmd.NewShell(shell.Options{ErrExit: true}, core.Stdio())
sh.Set("OUT", "dist")

err := sh.RunString(` + "`" + `
mkdirx -p $OUT
cpx -r assets $OUT/
lsx -l $OUT
` + "`" + `)
` + "```" + `

The shell reaches the tools only through its ` + "`Exec`" + ` function, so a program can also construct one with ` + "`shell.New`" + ` and supply its own command set - to restrict a script to a few tools, or to add commands of its own.

## Injection points

Engines expose the things that normally need a terminal, a clock or the real environment as fields - ` + "`PromptFn`, `RemoveFn`, `Now`, `Getenv`" + ` and the like - so they can be driven, and tested, from code.
`
