package tool

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// DocsURL is where the online reference lives; a tool's page is
// DocsURL + "/tools/<name>".
const DocsURL = "https://gnux.braswelljr.engineer/docs"

// helpWidth is the column help text is wrapped to.
const helpWidth = 80

// briefExamples is how many examples --help shows; the full manual shows all.
const briefExamples = 6

// Style decides how help text is decorated.
type Style struct {
	Color bool // bold, colored headings and flags (a terminal that allows it)
}

// StyleFor returns the style for help written to w: color only on a terminal,
// and never when NO_COLOR is set or TERM is "dumb" (https://no-color.org).
func StyleFor(w io.Writer) Style {
	return Style{
		Color: core.IsTerminal(w) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb",
	}
}

func (st Style) wrap(code, s string) string {
	if !st.Color || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (st Style) heading(s string) string { return st.wrap("1;36", s) }
func (st Style) strong(s string) string  { return st.wrap("1", s) }
func (st Style) dim(s string) string     { return st.wrap("2", s) }
func (st Style) command(s string) string { return st.wrap("32", s) }

// synopsis returns the usage line: the tool's own, or "gnux <use>" for a gnux
// command that is not a tool.
func (s Spec) synopsis() string {
	if s.GNU == "" {
		return common.AppName + " " + s.Meta.Use
	}
	return s.Meta.Use
}

// ExitStatus returns the documented exit statuses, defaulting to the GNU
// convention for the spec's usage-error status.
func (s Spec) ExitStatus() []docmeta.Status {
	if len(s.Meta.ExitStatus) > 0 {
		return s.Meta.ExitStatus
	}
	return docmeta.DefaultExitStatus(s.usageExit())
}

// manualCommand is the command that prints the full manual for the spec.
func (s Spec) manualCommand() string {
	return common.AppName + " help " + s.Name()
}

// WriteHelp writes the --help screen: synopsis, description, options, a few
// examples, the exit status and where to read more. It stays one screen of
// essentials in GNU's order; WriteManual is the long form.
func (s Spec) WriteHelp(w io.Writer, fs *pflag.FlagSet, st Style) {
	m := s.Meta
	fmt.Fprintf(w, "%s %s\n", st.strong("Usage:"), s.synopsis())
	if m.Short != "" {
		fmt.Fprintf(w, "%s\n", m.Short)
	}
	if m.Long != "" {
		fmt.Fprintf(w, "\n%s\n", indentWrap(m.Long, 2))
	}

	writeOptions(w, fs, st)

	if n := len(m.Examples); n > 0 {
		fmt.Fprintf(w, "\n%s\n", st.heading("Examples:"))
		shown := m.Examples
		if n > briefExamples {
			shown = shown[:briefExamples]
		}
		for _, e := range shown {
			writeExample(w, e, false, st)
		}
		if n > briefExamples {
			fmt.Fprintf(w, "  %s\n", st.dim(fmt.Sprintf("... %d more in '%s'", n-briefExamples, s.manualCommand())))
		}
	}

	writeExitStatus(w, s.ExitStatus(), st)

	fmt.Fprintf(w, "\n%s\n", st.heading("More help:"))
	if s.GNU != "" {
		fmt.Fprintf(w, "  %s and %s are the same command; both names work as a link or a subcommand.\n", s.Name(), s.GNU)
	}
	rows := [][2]string{{s.manualCommand(), fmt.Sprintf("the full manual: permissions, use cases, all %d examples", len(m.Examples))}}
	if flag := firstFlag(fs); flag != "" {
		rows = append(rows, [2]string{s.manualCommand() + " " + flag, "one option in depth (any option works)"})
	}
	width := 0
	for _, r := range rows {
		width = max(width, len(r[0]))
	}
	for _, r := range rows {
		fmt.Fprintf(w, "  %s  %s\n", st.command(pad(r[0], width)), r[1])
	}
	fmt.Fprintf(w, "  Online: %s\n", s.docsURL())
}

// WriteManual writes the full manual for the spec, laid out like a man page:
// name, synopsis, description, options, permissions, use cases, every example
// in both shells, per-flag topics, exit status and related commands.
func (s Spec) WriteManual(out io.Writer, fs *pflag.FlagSet, st Style) {
	w := &tidy{}
	defer w.flush(out)
	m := s.Meta
	title := s.Name()
	if s.GNU == "" {
		title = common.AppName + " " + s.Name()
	}

	section(w, st, "NAME")
	name := title
	if s.GNU != "" {
		name += " (GNU " + s.GNU + ")"
	}
	fmt.Fprintf(w, "    %s - %s\n", st.strong(name), strings.TrimSuffix(m.Short, "."))

	section(w, st, "SYNOPSIS")
	fmt.Fprintf(w, "    %s\n", s.synopsis())
	if s.GNU != "" {
		fmt.Fprintf(w, "    %s %s\n", common.AppName, m.Use)
		fmt.Fprintf(w, "    %s%s\n", s.GNU, strings.TrimPrefix(m.Use, s.Name()))
	}

	if m.Long != "" {
		section(w, st, "DESCRIPTION")
		fmt.Fprintln(w, indentWrap(m.Long, 4))
	}

	if hasFlags(fs) {
		section(w, st, "OPTIONS")
		fs.VisitAll(func(f *pflag.Flag) {
			fmt.Fprintf(w, "    %s\n", st.strong(flagLabel(f)))
			usage := f.Usage
			if d := flagDefault(f); d != "" {
				usage += " (default " + d + ")"
			}
			fmt.Fprintln(w, indentWrap(usage, 8))
		})
	}

	if m.Permissions != "" {
		section(w, st, "PERMISSIONS")
		fmt.Fprintln(w, indentWrap(m.Permissions, 4))
	}

	if len(m.UseCases) > 0 {
		section(w, st, "USE CASES")
		for _, u := range m.UseCases {
			fmt.Fprintln(w, bulletWrap(u, 4))
		}
	}

	if len(m.Examples) > 0 {
		section(w, st, "EXAMPLES")
		for _, e := range m.Examples {
			writeExample(w, e, true, st)
		}
	}

	if len(s.Topics) > 0 {
		section(w, st, "OPTIONS IN DEPTH")
		for _, t := range s.Topics {
			fmt.Fprintf(w, "    %s %s\n", st.command(pad(s.manualCommand()+" "+t.Meta.Name, 36)), t.Meta.Short)
		}
	}

	section(w, st, "EXIT STATUS")
	writeStatusRows(w, s.ExitStatus(), 4)

	section(w, st, "SEE ALSO")
	var refs []string
	for _, r := range m.SeeAlso {
		refs = append(refs, common.AppName+" help "+strings.TrimPrefix(r, common.AppName+" "))
	}
	if len(refs) > 0 {
		fmt.Fprintf(w, "    %s\n", strings.Join(refs, ", "))
	}
	fmt.Fprintf(w, "    %s\n", s.docsURL())
}

// WriteFlagHelp writes the in-depth help for one option of the spec. flag may
// be a topic name ("force"), a long name with or without dashes ("--force",
// "force") or a shorthand ("-f", "f"). A documented topic is printed in full;
// any other flag gets its usage, default and every example that uses it.
// It reports false when the spec has no such option.
func (s Spec) WriteFlagHelp(out io.Writer, fs *pflag.FlagSet, flag string, st Style) bool {
	w := &tidy{}
	defer w.flush(out)
	key := strings.TrimLeft(flag, "-")
	f := fs.Lookup(key)
	if f == nil && len(key) == 1 {
		f = fs.ShorthandLookup(key)
	}

	for _, t := range s.Topics {
		tfs := pflag.NewFlagSet(t.Meta.Name, pflag.ContinueOnError)
		t.Register(tfs)
		match := t.Meta.Name == key
		if f != nil && tfs.Lookup(f.Name) != nil {
			match = true
		}
		if match {
			s.writeTopic(w, t, tfs, st)
			return true
		}
	}
	if f == nil {
		return false
	}

	section(w, st, "OPTION")
	fmt.Fprintf(w, "    %s  %s\n", st.strong(flagLabel(f)), st.dim("("+s.Name()+")"))
	fmt.Fprintln(w, indentWrap(f.Usage, 8))
	if d := flagDefault(f); d != "" {
		fmt.Fprintf(w, "        default: %s\n", d)
	}
	if f.NoOptDefVal != "" && f.Value.Type() != "bool" {
		fmt.Fprintf(w, "        given without a value: %s\n", f.NoOptDefVal)
	}

	section(w, st, "EXAMPLES")
	var found bool
	for _, e := range s.Meta.Examples {
		if usesFlag(e.Bash, s, f) {
			writeExample(w, e, true, st)
			found = true
		}
	}
	if !found {
		fmt.Fprintf(w, "    No worked example uses %s on its own; see '%s'.\n", flagLabel(f), s.manualCommand())
	}

	section(w, st, "SEE ALSO")
	fmt.Fprintf(w, "    %s, %s\n", s.manualCommand(), s.docsURL())
	return true
}

// writeTopic prints a per-flag topic page in full.
func (s Spec) writeTopic(w io.Writer, t Topic, fs *pflag.FlagSet, st Style) {
	m := t.Meta
	section(w, st, "NAME")
	fmt.Fprintf(w, "    %s - %s\n", st.strong(s.Name()+" "+m.Name), strings.TrimSuffix(m.Short, "."))
	if hasFlags(fs) {
		section(w, st, "OPTIONS")
		fs.VisitAll(func(f *pflag.Flag) {
			fmt.Fprintf(w, "    %s\n", st.strong(flagLabel(f)))
			fmt.Fprintln(w, indentWrap(f.Usage, 8))
		})
	}
	if m.Long != "" {
		section(w, st, "DESCRIPTION")
		fmt.Fprintln(w, indentWrap(m.Long, 4))
	}
	if m.Permissions != "" {
		section(w, st, "PERMISSIONS")
		fmt.Fprintln(w, indentWrap(m.Permissions, 4))
	}
	if len(m.UseCases) > 0 {
		section(w, st, "USE CASES")
		for _, u := range m.UseCases {
			fmt.Fprintln(w, bulletWrap(u, 4))
		}
	}
	if len(m.Examples) > 0 {
		section(w, st, "EXAMPLES")
		for _, e := range m.Examples {
			writeExample(w, e, true, st)
		}
	}
	section(w, st, "SEE ALSO")
	fmt.Fprintf(w, "    %s, %s/flags/%s\n", s.manualCommand(), s.docsURL(), m.Name)
}

// docsURL is the tool's page in the online reference.
func (s Spec) docsURL() string {
	if s.GNU == "" {
		return DocsURL + "/" + s.Name()
	}
	return DocsURL + "/tools/" + s.Name()
}

// usesFlag reports whether a bash example passes f to the spec's tool: as
// --name, --name=value, or a shorthand inside a cluster of short options
// (-rf), counting only arguments that follow the tool's own name.
func usesFlag(bash string, s Spec, f *pflag.Flag) bool {
	active := false
	for _, tok := range strings.Fields(bash) {
		tok = strings.Trim(tok, `'"`)
		switch {
		case tok == s.Name() || (s.GNU != "" && tok == s.GNU):
			active = true
			continue
		case tok == "|" || tok == "&&" || tok == "||" || tok == ";" || strings.HasSuffix(tok, ";"):
			active = false
			continue
		case !active:
			continue
		}
		if tok == "--" {
			active = false
			continue
		}
		if tok == "--"+f.Name || strings.HasPrefix(tok, "--"+f.Name+"=") {
			return true
		}
		if f.Shorthand != "" && len(tok) > 1 && tok[0] == '-' && tok[1] != '-' &&
			isLetters(tok[1:]) && strings.Contains(tok[1:], f.Shorthand) {
			return true
		}
	}
	return false
}

// isLetters reports whether s is a run of letters: a short-option cluster, as
// opposed to a negative number or an option with an attached value.
func isLetters(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return s != ""
}

// tidy collects a manual so it can be written without a leading blank line
// and with at most one blank line between blocks, however the sections and
// examples happen to be spaced.
type tidy struct{ strings.Builder }

func (t *tidy) flush(w io.Writer) {
	text := strings.TrimLeft(t.String(), "\n")
	if text == "" {
		return
	}
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	fmt.Fprint(w, strings.TrimRight(text, "\n")+"\n")
}

// pad left-aligns s in a field of n columns. It is applied before any color,
// so escape sequences never count toward the width.
func pad(s string, n int) string {
	return fmt.Sprintf("%-*s", n, s)
}

func section(w io.Writer, st Style, name string) {
	fmt.Fprintf(w, "\n%s\n", st.heading(name))
}

func writeOptions(w io.Writer, fs *pflag.FlagSet, st Style) {
	if !hasFlags(fs) {
		return
	}
	fmt.Fprintf(w, "\n%s\n", st.heading("Options:"))
	fmt.Fprint(w, fs.FlagUsagesWrapped(helpWidth))
}

// writeExample prints one example as a comment line and the command; with
// both set, the PowerShell form follows when it differs.
func writeExample(w io.Writer, e docmeta.Example, both bool, st Style) {
	if e.Description != "" {
		fmt.Fprintf(w, "  %s\n", st.dim("# "+strings.TrimRight(e.Description, ":")))
	}
	if e.Bash != "" {
		for _, line := range strings.Split(e.Bash, "\n") {
			fmt.Fprintf(w, "  %s %s\n", st.dim("$"), st.command(line))
		}
	}
	if both && e.PowerShell != "" && e.PowerShell != e.Bash {
		for _, line := range strings.Split(e.PowerShell, "\n") {
			fmt.Fprintf(w, "  %s %s\n", st.dim("PS>"), st.command(line))
		}
	}
	fmt.Fprintln(w)
}

func writeExitStatus(w io.Writer, rows []docmeta.Status, st Style) {
	fmt.Fprintf(w, "\n%s\n", st.heading("Exit status:"))
	writeStatusRows(w, rows, 2)
}

func writeStatusRows(w io.Writer, rows []docmeta.Status, indent int) {
	width := 0
	for _, r := range rows {
		width = max(width, len(r.Code))
	}
	pad := strings.Repeat(" ", indent)
	for _, r := range rows {
		label := fmt.Sprintf("%s%-*s  ", pad, width, r.Code)
		fmt.Fprintln(w, label+hangWrap(r.Meaning, len(label)))
	}
}

func hasFlags(fs *pflag.FlagSet) bool {
	if fs == nil {
		return false
	}
	found := false
	fs.VisitAll(func(*pflag.Flag) { found = true })
	return found
}

// firstFlag returns a flag worth suggesting for "gnux help TOOL FLAG": the
// first one that is not --help or --version.
func firstFlag(fs *pflag.FlagSet) string {
	var name string
	fs.VisitAll(func(f *pflag.Flag) {
		if name == "" && f.Name != "help" && f.Name != "version" {
			name = "--" + f.Name
		}
	})
	return name
}

// flagLabel is "-f, --force" or "--no-preserve-root", with "=VALUE" for a flag
// that takes one.
func flagLabel(f *pflag.Flag) string {
	label := "--" + f.Name
	if f.Shorthand != "" {
		label = "-" + f.Shorthand + ", " + label
	}
	if t := f.Value.Type(); t != "bool" {
		value := strings.ToUpper(t)
		if name, _ := pflag.UnquoteUsage(f); name != "" {
			value = strings.ToUpper(name)
		}
		if f.NoOptDefVal != "" {
			label += "[=" + value + "]"
		} else {
			label += "=" + value
		}
	}
	return label
}

// flagDefault is a flag's default when it says something: not false, zero or
// empty.
func flagDefault(f *pflag.Flag) string {
	switch f.DefValue {
	case "", "false", "0", "[]":
		return ""
	}
	return f.DefValue
}

// indentWrap wraps text to helpWidth with every line indented by n spaces.
// Paragraphs (separated by a blank line) are kept apart.
func indentWrap(text string, n int) string {
	pad := strings.Repeat(" ", n)
	var paras []string
	for _, p := range strings.Split(text, "\n\n") {
		var out []string
		for _, line := range wrapWords(p, helpWidth-n) {
			out = append(out, pad+line)
		}
		paras = append(paras, strings.Join(out, "\n"))
	}
	return strings.Join(paras, "\n\n")
}

// bulletWrap renders text as a "- " bullet indented by n, with continuation
// lines aligned under the text.
func bulletWrap(text string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := wrapWords(text, helpWidth-n-2)
	for i := range lines {
		if i == 0 {
			lines[i] = pad + "- " + lines[i]
		} else {
			lines[i] = pad + "  " + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

// hangWrap wraps text so that its first line continues after a label of
// width col and the rest are indented to col.
func hangWrap(text string, col int) string {
	lines := wrapWords(text, helpWidth-col)
	return strings.Join(lines, "\n"+strings.Repeat(" ", col))
}

// wrapWords splits text into lines of at most width columns, breaking at
// spaces. A word longer than width gets a line of its own.
func wrapWords(text string, width int) []string {
	width = max(width, 20)
	var lines []string
	var line strings.Builder
	for _, word := range strings.Fields(text) {
		if line.Len() > 0 && line.Len()+1+len(word) > width {
			lines = append(lines, line.String())
			line.Reset()
		}
		if line.Len() > 0 {
			line.WriteByte(' ')
		}
		line.WriteString(word)
	}
	if line.Len() > 0 || len(lines) == 0 {
		lines = append(lines, line.String())
	}
	return lines
}
