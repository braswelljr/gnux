package docs

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// renderMarkdown writes p as markdown: the hand-rendered page when it has one,
// otherwise cobra's, with fences language-tagged and references wrapped in code.
func renderMarkdown(p page, w io.Writer) error {
	if p.markdown != nil {
		_, err := w.Write(wrapCode([]byte(p.markdown())))
		return err
	}

	buf := new(bytes.Buffer)
	err := doc.GenMarkdownCustom(p.command, buf, func(link string) string {
		if target, ok := p.links[strings.TrimSuffix(link, ".md")]; ok {
			return target + ".md"
		}
		return link
	})
	if err != nil {
		return err
	}
	_, err = w.Write(wrapCode(tagMarkdownFences(buf.Bytes())))
	return err
}

// toolMarkdown builds the reference page for one tool: the topic sections plus
// the synopsis, the GNU tool it mirrors and links to its per-flag pages.
func toolMarkdown(t tool.Spec, flags *pflag.FlagSet) string {
	var b strings.Builder
	m := t.Meta

	fmt.Fprintf(&b, "## %s\n\n%s\n\n", m.Name, m.Short)
	if t.GNU != "" {
		fmt.Fprintf(&b, "Mirrors GNU `%s`. Run it as `%s`, or as `%s %s`.\n\n", t.GNU, m.Name, common.AppName, m.Name)
	}
	use := m.Use
	if t.GNU == "" {
		// A gnux command rather than a tool: it only exists as a subcommand.
		use = common.AppName + " " + use
	}
	fmt.Fprintf(&b, "### Synopsis\n\n```bash\n%s\n```\n\n", use)
	b.WriteString(gnuReference(t, "###"))
	b.WriteString(topicBody(m, flags, 3, false))

	if len(t.Topics) > 0 {
		b.WriteString("\n### Flag reference\n\n")
		for _, topic := range t.Topics {
			fmt.Fprintf(&b, "- [%s](flags/%s.md) - %s\n", topic.Meta.Name, topic.Meta.Name, topic.Meta.Short)
		}
	}

	b.WriteString("\n")
	b.WriteString(exitStatusMarkdown(t, "###"))
	b.WriteString("\n")
	b.WriteString(terminalHelpMarkdown(t, "###"))

	b.WriteString("\n### See also\n\n")
	for _, r := range t.Meta.SeeAlso {
		if target := strings.TrimPrefix(r, common.AppName+" "); target != r {
			fmt.Fprintf(&b, "- [%s](../%s/index.md)\n", r, target)
		} else {
			fmt.Fprintf(&b, "- [%s](../%s/index.md)\n", r, r)
		}
	}
	fmt.Fprintf(&b, "- [%s](../index.md) - every tool in the bundle\n", common.AppName)
	fmt.Fprintf(&b, "- [guide](../guide.md) - installation and usage\n")
	return b.String()
}

// exitStatusMarkdown renders a tool's exit statuses as a table under a heading
// of the given level.
func exitStatusMarkdown(t tool.Spec, h string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s Exit status\n\n| Status | Meaning |\n|--------|---------|\n", h)
	for _, s := range t.ExitStatus() {
		fmt.Fprintf(&b, "| %s | %s |\n", s.Code, s.Meaning)
	}
	return b.String()
}

// terminalHelpMarkdown says how to read the same documentation in a terminal.
func terminalHelpMarkdown(t tool.Spec, h string) string {
	var b strings.Builder
	name := t.Name()
	fmt.Fprintf(&b, "%s Help in the terminal\n\n```bash\n", h)
	if t.GNU != "" {
		fmt.Fprintf(&b, "%s --help                # options, first examples, exit status\n", name)
	}
	fmt.Fprintf(&b, "%s help %s             # this page: the full manual\n", common.AppName, name)
	fmt.Fprintf(&b, "%s help %s --OPTION    # one option in depth, with its examples\n```\n", common.AppName, name)
	return b.String()
}

// topicMarkdown builds a rich, multi-section markdown page for one topic:
// description, permissions, use cases, the flag listing, and per-shell examples
// in language-tagged code blocks. level is the heading level of the title.
func topicMarkdown(m docmeta.Meta, flags *pflag.FlagSet, level int) string {
	h := strings.Repeat("#", level)
	return fmt.Sprintf("%s %s\n\n%s\n\n", h, m.Name, m.Short) + topicBody(m, flags, level+1, false)
}

// topicBody renders the sections shared by tool and flag pages, with section
// headings at the given level. mdx selects the website's rendering of the
// examples (see writeExamples).
func topicBody(m docmeta.Meta, flags *pflag.FlagSet, level int, mdx bool) string {
	var b strings.Builder
	h := strings.Repeat("#", level)

	fmt.Fprintf(&b, "%s Description\n\n%s\n\n", h, m.Long)

	if m.Permissions != "" {
		fmt.Fprintf(&b, "%s Permissions\n\n%s\n\n", h, m.Permissions)
	}

	if len(m.UseCases) > 0 {
		fmt.Fprintf(&b, "%s Use cases\n\n", h)
		for _, u := range m.UseCases {
			fmt.Fprintf(&b, "- %s\n", u)
		}
		b.WriteString("\n")
	}

	if usage := strings.TrimRight(flags.FlagUsages(), "\n"); usage != "" {
		fmt.Fprintf(&b, "%s Flags\n\n```text\n%s\n```\n\n", h, usage)
	}

	if len(m.Examples) > 0 {
		fmt.Fprintf(&b, "%s Examples\n\n", h)
		writeExamples(&b, m.Examples, mdx, level+1)
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// Patterns for wrapping flag and command references in prose. Applied to
// markdown only - the source strings stay plain so terminal help and the
// man/yaml/reST outputs are unaffected.
var (
	// flags: -d, -rf, --dir, --no-preserve-root (preceded by a non-word/dash char)
	flagRefRe = regexp.MustCompile(`(^|[^\w-])(--?[A-Za-z][\w-]*)`)
	// multi-word commands: "go install", "make build", "docker run", "gnux shell", ...
	cmdPhraseRe = regexp.MustCompile(`\b((?:go|make|docker) (?:install|build|generate|test|run|compose|mod|docs|lint|cover|tidy|get|links|ci)|gnux shell)\b`)
)

// proseWords are GNU tool names that are also ordinary English words. They are
// left alone in prose: wrapping every "true", "head" or "touch" in code would
// mangle normal sentences. The x-suffixed names are unambiguous and always wrap.
var proseWords = map[string]bool{
	"cat": true, "echo": true, "env": true, "false": true, "head": true,
	"sleep": true, "tail": true, "touch": true, "true": true, "yes": true,
	"sort": true, "cut": true, "tee": true, "find": true,
	"dir": true, "groups": true, "users": true, "sum": true, "factor": true, "arch": true,
}

// cmdWordRe matches standalone command names: gnux, every tool, and the GNU
// names that cannot be mistaken for prose. It is built once from the registry.
var cmdWordRe = sync.OnceValue(func() *regexp.Regexp {
	words := []string{common.AppName}
	for _, name := range toolNames() {
		words = append(words, name)
		if gnu := strings.TrimSuffix(name, "x"); !proseWords[gnu] {
			words = append(words, gnu)
		}
	}
	for i, w := range words {
		words[i] = regexp.QuoteMeta(w)
	}
	return regexp.MustCompile(`\b(` + strings.Join(words, "|") + `)\b`)
})

// wrapCode wraps flag and command references in prose with backticks so they
// read as code and stand out from regular words. It skips fenced code blocks,
// inline code spans and headings, so existing code is never touched.
func wrapCode(md []byte) []byte {
	lines := strings.Split(string(md), "\n")
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence || strings.HasPrefix(line, "#") {
			continue
		}
		lines[i] = wrapCodeInLine(line)
	}
	return []byte(strings.Join(lines, "\n"))
}

// linkRe matches a markdown link, whose text and target must not be rewritten.
var linkRe = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)

// wrapCodeInLine wraps references in a single prose line, leaving anything
// already inside inline-code spans (backtick pairs) or markdown links untouched.
func wrapCodeInLine(line string) string {
	parts := strings.Split(line, "`")
	// Even indexes are outside inline code; odd indexes are inside it.
	for i := 0; i < len(parts); i += 2 {
		parts[i] = outsideLinks(parts[i], func(s string) string {
			// Phrases first; what they wrap is then code, and is skipped below
			// so a word inside a phrase is not wrapped a second time.
			spans := strings.Split(cmdPhraseRe.ReplaceAllString(s, "`$1`"), "`")
			for j := 0; j < len(spans); j += 2 {
				spans[j] = cmdWordRe().ReplaceAllString(spans[j], "`$1`")
				spans[j] = flagRefRe.ReplaceAllString(spans[j], "$1`$2`")
			}
			// Neighboring references form one command line: `rmx` `-rf` reads
			// better as `rmx -rf`.
			return strings.ReplaceAll(strings.Join(spans, "`"), "` `", " ")
		})
	}
	return strings.Join(parts, "`")
}

// outsideLinks applies fn to the stretches of s that are not markdown links.
func outsideLinks(s string, fn func(string) string) string {
	var b strings.Builder
	last := 0
	for _, loc := range linkRe.FindAllStringIndex(s, -1) {
		b.WriteString(fn(s[last:loc[0]]))
		b.WriteString(s[loc[0]:loc[1]])
		last = loc[1]
	}
	b.WriteString(fn(s[last:]))
	return b.String()
}

// tagMarkdownFences rewrites each opening ``` fence to ```<lang>, choosing the
// language from the nearest preceding heading. Closing fences are left bare.
func tagMarkdownFences(md []byte) []byte {
	lines := strings.Split(string(md), "\n")
	section := ""
	inBlock := false
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			section = strings.ToLower(line)
			continue
		}
		if strings.TrimSpace(line) == "```" {
			if inBlock {
				inBlock = false
				continue
			}
			inBlock = true
			lines[i] = "```" + fenceLang(section)
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// fenceLang maps a markdown section heading to a code-block language. The
// options listing is a plain table (text); synopsis and examples are shell.
func fenceLang(section string) string {
	if strings.Contains(section, "option") {
		return "text"
	}
	return "bash"
}

// writeExamples renders worked examples: the scenario, the command for each
// shell, and the output when the example gives one.
//
// For the website (mdx) each scenario is a heading of the given level, so the
// page's table of contents lists them, and its commands are tabs of one group
// ("shell"), so a reader who picks PowerShell sees PowerShell everywhere. The
// output is a code block titled "Output". The other formats keep plain
// paragraphs and fences.
func writeExamples(b *strings.Builder, examples []docmeta.Example, mdx bool, level int) {
	if mdx {
		h := strings.Repeat("#", level)
		for _, e := range examples {
			if e.Description != "" {
				fmt.Fprintf(b, "%s %s\n\n", h, strings.TrimRight(e.Description, ":"))
			}
			if e.Bash != "" {
				fmt.Fprintf(b, "```bash tab=\"bash\" tab-group=\"shell\"\n%s\n```\n\n", e.Bash)
			}
			if e.PowerShell != "" {
				fmt.Fprintf(b, "```powershell tab=\"PowerShell\" tab-group=\"shell\"\n%s\n```\n\n", e.PowerShell)
			}
			if e.Output != "" {
				fmt.Fprintf(b, "```text title=\"Output\"\n%s\n```\n\n", e.Output)
			}
		}
		return
	}
	for _, e := range examples {
		if e.Description != "" {
			fmt.Fprintf(b, "%s\n\n", e.Description)
		}
		if e.Bash != "" {
			fmt.Fprintf(b, "```bash\n%s\n```\n\n", e.Bash)
		}
		if e.PowerShell != "" {
			fmt.Fprintf(b, "```powershell\n%s\n```\n\n", e.PowerShell)
		}
		if e.Output != "" {
			fmt.Fprintf(b, "Output:\n\n```text\n%s\n```\n\n", e.Output)
		}
	}
}

// gnuReference links a tool's page to GNU's documentation and source for the
// tool it mirrors: the coreutils manual and the source file in the
// coreutils/coreutils repository. It is empty for a gnux command with no GNU
// counterpart. The link texts carry no backticks: wrapCode splits a line on
// them before it looks for links, so code inside a link's text would break it.
func gnuReference(t tool.Spec, h string) string {
	if t.GNU == "" {
		return ""
	}
	return fmt.Sprintf("%s GNU reference\n\n"+
		"- [GNU coreutils manual: %s invocation](%s)\n"+
		"- [Source: src/%s in coreutils/coreutils](%s)\n\n",
		h, t.GNU, gnuManualURL(t.GNU), gnuSourceFile(t.GNU), gnuSourceURL(t.GNU))
}

// gnuManualURL is the coreutils manual node for a GNU tool. The SHA-2 tools
// share one node.
func gnuManualURL(gnu string) string {
	node := gnu + "-invocation"
	switch gnu {
	case "sha224sum", "sha256sum", "sha384sum", "sha512sum":
		node = "sha2-utilities"
	}
	return "https://www.gnu.org/software/coreutils/manual/html_node/" + node + ".html"
}

// gnuSourceFile is the file in coreutils' src/ that implements a GNU tool.
// The digest tools are cksum with a fixed algorithm, base32 and base64 are
// basenc, chgrp shares chown's sources and false shares true's.
func gnuSourceFile(gnu string) string {
	switch gnu {
	case "md5sum", "sha1sum", "sha224sum", "sha256sum", "sha384sum", "sha512sum", "b2sum":
		return "cksum.c"
	case "base32", "base64":
		return "basenc.c"
	case "chgrp":
		return "chown.c"
	case "false":
		return "true.c"
	}
	return gnu + ".c"
}

// gnuSourceURL is the address of a GNU tool's source on GitHub.
func gnuSourceURL(gnu string) string {
	return "https://github.com/coreutils/coreutils/blob/master/src/" + gnuSourceFile(gnu)
}
