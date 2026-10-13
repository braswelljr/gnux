package docs

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// flagPage is one option's page on the website: tools/<tool>/<name>.mdx.
type flagPage struct {
	name        string // file base name and URL segment
	title       string // every spelling: "-f, --force"
	description string
	body        string
}

// flagPages builds a page for every option a tool takes. A Topic becomes one
// page covering all the flags it registers, with its hand-written prose; every
// other flag gets a page of its own from its usage line, with the tool's
// examples that use it. --help and --version are left out: every tool has them
// and the tool page already says so.
func flagPages(t tool.Spec, flags *pflag.FlagSet) []flagPage {
	covered := map[string]bool{"help": true, "version": true}
	var pages []flagPage

	for _, topic := range t.Topics {
		fs := pflag.NewFlagSet(topic.Meta.Name, pflag.ContinueOnError)
		topic.Register(fs)
		var own []*pflag.Flag
		fs.VisitAll(func(f *pflag.Flag) {
			covered[f.Name] = true
			own = append(own, f)
		})
		m := topic.Meta
		if len(m.Examples) == 0 {
			m.Examples = examplesUsing(t, own...)
		}
		pages = append(pages, flagPage{
			name:        topic.Meta.Name,
			title:       spellings(own...),
			description: topic.Meta.Short,
			body:        flagBody(t, m, own),
		})
	}

	flags.VisitAll(func(f *pflag.Flag) {
		if covered[f.Name] || f.Hidden {
			return
		}
		_, usage := pflag.UnquoteUsage(f)
		m := docmeta.Meta{
			Name:     f.Name,
			Short:    sentence(usage),
			Long:     sentence(usage),
			Examples: examplesUsing(t, f),
		}
		pages = append(pages, flagPage{
			name:        f.Name,
			title:       spellings(f),
			description: m.Short,
			body:        flagBody(t, m, []*pflag.Flag{f}),
		})
	})
	return pages
}

// flagBody renders one option page: how to write it, what it does, its
// details, and examples. The page title comes from the frontmatter.
func flagBody(t tool.Spec, m docmeta.Meta, own []*pflag.Flag) string {
	var b strings.Builder
	name := t.Name()
	fmt.Fprintf(&b, "An option of [%s](/docs/tools/%s).\n\n", name, name)

	b.WriteString("## Usage\n\n```bash\n")
	for _, f := range own {
		for _, form := range forms(f) {
			b.WriteString(synopsisWith(t.Meta.Use, form))
			b.WriteString("\n")
		}
	}
	b.WriteString("```\n\n")

	// A flag documented only by its usage line has nothing to add to the
	// description the page already opens with.
	if m.Long != m.Short {
		fmt.Fprintf(&b, "## Description\n\n%s\n\n", m.Long)
	}
	if m.Permissions != "" {
		fmt.Fprintf(&b, "## Permissions\n\n%s\n\n", m.Permissions)
	}
	if len(m.UseCases) > 0 {
		b.WriteString("## Use cases\n\n")
		for _, u := range m.UseCases {
			fmt.Fprintf(&b, "- %s\n", u)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Details\n\n| Short | Long | Value | Default | Meaning |\n| --- | --- | --- | --- | --- |\n")
	for _, f := range own {
		varname, usage := pflag.UnquoteUsage(f)
		short, value, def := "", "none", ""
		if f.Shorthand != "" {
			short = "`-" + f.Shorthand + "`"
		}
		if varname != "" {
			value = "`" + varname + "`"
			if f.NoOptDefVal != "" {
				value += " (optional)"
			}
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "[]" {
			def = "`" + f.DefValue + "`"
		}
		fmt.Fprintf(&b, "| %s | `--%s` | %s | %s | %s |\n", short, f.Name, value, def, cell(usage))
	}
	b.WriteString("\n")

	if len(m.Examples) > 0 {
		b.WriteString("## Examples\n\n")
		writeExamples(&b, m.Examples, true, 3)
	}

	fmt.Fprintf(&b, "## Help in the terminal\n\n```bash\n%s help %s --%s\n```\n", common.AppName, name, own[0].Name)
	return b.String()
}

// spellings is every way to write the flags, short forms first: "-f, --force".
func spellings(flags ...*pflag.Flag) string {
	var short, long []string
	for _, f := range flags {
		if f.Shorthand != "" {
			short = append(short, "-"+f.Shorthand)
		}
		long = append(long, "--"+f.Name)
	}
	return strings.Join(append(short, long...), ", ")
}

// forms is how a flag is written on a command line: "-n NUM" and
// "--lines=NUM"; an optional value is bracketed, "--interactive[=WHEN]".
func forms(f *pflag.Flag) []string {
	varname, _ := pflag.UnquoteUsage(f)
	var out []string
	switch {
	case varname == "":
		if f.Shorthand != "" {
			out = append(out, "-"+f.Shorthand)
		}
		out = append(out, "--"+f.Name)
	case f.NoOptDefVal != "":
		if f.Shorthand != "" {
			out = append(out, "-"+f.Shorthand)
		}
		out = append(out, "--"+f.Name+"[="+varname+"]")
	default:
		if f.Shorthand != "" {
			out = append(out, "-"+f.Shorthand+" "+varname)
		}
		out = append(out, "--"+f.Name+"="+varname)
	}
	return out
}

// synopsisWith puts one option into a tool's synopsis, in place of its
// "[OPTION]..." placeholder, or right after the name when it has none.
func synopsisWith(use, form string) string {
	for _, placeholder := range []string{"[OPTION]...", "[SHORT-OPTION]...", "[OPTIONS]"} {
		if strings.Contains(use, placeholder) {
			return strings.Replace(use, placeholder, form, 1)
		}
	}
	name, rest, _ := strings.Cut(use, " ")
	return strings.TrimSpace(name + " " + form + " " + rest)
}

// examplesUsing picks the tool's examples whose command passes one of the
// flags to the tool itself: "-f", "--force", "--force=x", or a cluster such
// as "-rf". A flag after a pipe belongs to another command and is not counted.
func examplesUsing(t tool.Spec, flags ...*pflag.Flag) []docmeta.Example {
	var out []docmeta.Example
	for _, e := range t.Meta.Examples {
		if slices.ContainsFunc(flags, func(f *pflag.Flag) bool { return passes(t, e.Bash, f) }) {
			out = append(out, e)
		}
	}
	return out
}

// segmentSplitter splits a command line into simple commands.
var segmentSplitter = strings.NewReplacer("&&", "\n", "||", "\n", "|", "\n", ";", "\n")

// passes reports whether command runs the tool with flag f.
func passes(t tool.Spec, command string, f *pflag.Flag) bool {
	for _, line := range strings.Split(segmentSplitter.Replace(command), "\n") {
		words := strings.Fields(line)
		if len(words) > 1 && words[0] == common.AppName {
			words = words[1:]
		}
		if len(words) == 0 || (words[0] != t.Name() && words[0] != t.GNU) {
			continue
		}
		for _, w := range words[1:] {
			if w == "--" {
				break
			}
			// A tool that stops at its first operand (envx) passes the rest of
			// the line to another command, whose options are not its own.
			if t.NoInterspersed && !strings.HasPrefix(w, "-") {
				break
			}
			if w == "--"+f.Name || strings.HasPrefix(w, "--"+f.Name+"=") {
				return true
			}
			// "-" alone is an operand (standard input), never a flag.
			if f.Shorthand != "" && w == "-"+f.Shorthand {
				return true
			}
			if f.Shorthand != "" && len(w) > 1 && w[0] == '-' && w[1] != '-' &&
				isLetter(w[1]) && strings.Contains(shortCluster(w[1:]), f.Shorthand) {
				return true
			}
		}
	}
	return false
}

// shortCluster is the run of option letters at the start of a cluster: "rf"
// in "-rf", "n" in "-n5". A value glued to an option is not more options.
func shortCluster(s string) string {
	for i := 0; i < len(s); i++ {
		if !isLetter(s[i]) {
			return s[:i]
		}
	}
	return s
}

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// sentence capitalizes a usage line and ends it with a period, for prose.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = strings.ToUpper(s[:1]) + s[1:]
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}

// cell makes text safe inside a markdown table cell.
func cell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", `\|`), "\n", " ")
}
