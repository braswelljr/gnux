// Package ptxx wires the ptxx command (GNU ptx) to its engine.
package ptxx

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/ptx"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the ptxx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "ptx",
		Flags: Register,
		Run:   run,
	}
}

// format is one way of choosing the output format: -O, -T or --format. They
// share one selection, which the last of them sets, as in GNU ptx.
type format struct {
	fixed    ptx.Format // what a switch selects; Default for --format
	sel      *ptx.Format
	text     string
	isSwitch bool
}

func (f *format) Set(v string) error {
	if f.isSwitch {
		on, err := strconv.ParseBool(v)
		if err == nil && on {
			*f.sel = f.fixed
		}
		return err
	}
	f.text = v
	var found []ptx.Format
	for _, c := range []struct {
		name string
		f    ptx.Format
	}{{"roff", ptx.Roff}, {"tex", ptx.TeX}} {
		if c.name == v {
			found = []ptx.Format{c.f}
			break
		}
		if v != "" && strings.HasPrefix(c.name, v) {
			found = append(found, c.f)
		}
	}
	if len(found) != 1 {
		what := "invalid"
		if len(found) > 1 {
			what = "ambiguous"
		}
		return &core.UsageError{Msg: fmt.Sprintf("%s argument '%s' for '--format'\nValid arguments are:\n  - 'roff'\n  - 'tex'", what, v)}
	}
	*f.sel = found[0]
	return nil
}

func (f *format) String() string {
	if f.isSwitch {
		return strconv.FormatBool(f.sel != nil && *f.sel == f.fixed)
	}
	return f.text
}

func (f *format) Type() string {
	if f.isSwitch {
		return "bool"
	}
	return "string"
}

// IsBoolFlag marks -O and -T as switches.
func (f *format) IsBoolFlag() bool { return f.isSwitch }

// Register installs the ptxx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("auto-reference", "A", false, "output automatically generated references")
	fs.BoolP("traditional", "G", false, "behave more like System V 'ptx'")
	fs.StringP("flag-truncation", "F", "", "use `STRING` for flagging line truncations. The default is '/'")
	fs.StringP("macro-name", "M", "", "macro name to use instead of 'xx'")
	sel := new(ptx.Format)
	o := fs.VarPF(&format{fixed: ptx.Roff, sel: sel, isSwitch: true}, "roff", "O", "generate output as roff directives (--format=roff)")
	o.NoOptDefVal = "true"
	t := fs.VarPF(&format{fixed: ptx.TeX, sel: sel, isSwitch: true}, "tex", "T", "generate output as TeX directives (--format=tex)")
	t.NoOptDefVal = "true"
	fs.Var(&format{sel: sel}, "format", "generate output as roff or TeX directives: `FORMAT` is roff or tex")
	fs.BoolP("right-side-refs", "R", false, "put references at right, not counted in -w")
	fs.StringP("sentence-regexp", "S", "", "for end of lines or end of sentences")
	fs.StringP("word-regexp", "W", "", "use `REGEXP` to match each keyword")
	fs.StringP("break-file", "b", "", "word break characters in this `FILE`")
	fs.BoolP("ignore-case", "f", false, "fold lower case to upper case for sorting")
	fs.StringP("gap-size", "g", "", "gap size in columns between output fields")
	fs.StringP("ignore-file", "i", "", "read ignore word list from `FILE`")
	fs.StringP("only-file", "o", "", "read only word list from this `FILE`")
	fs.BoolP("references", "r", false, "first field of each line is a reference")
	fs.BoolP("typeset-mode", "t", false, "- not implemented -")
	fs.StringP("width", "w", "", "output width in columns, reference excluded")
	tool.OwnNames(fs, "roff", "tex")
}

// fatal is a mistake GNU reports without the --help hint.
type fatal struct{ msg string }

func (e *fatal) Error() string { return e.msg }

// positive reads -g and -w: a number, base 0 as in C, above zero.
func positive(s, what string) (int, error) {
	t := strings.TrimLeft(s, " \t\n\v\f\r")
	base := 10
	switch {
	case strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "0X"):
		base, t = 16, t[2:]
	case len(t) > 1 && t[0] == '0':
		base, t = 8, t[1:]
	}
	n, err := strconv.ParseInt(t, base, 64)
	if err != nil || n <= 0 || n > math.MaxInt32 || strings.ContainsAny(t, "_") {
		return 0, &fatal{fmt.Sprintf("invalid %s: '%s'", what, s)}
	}
	return int(n), nil
}

// buildOptions folds the parsed flags into ptx.Options.
func buildOptions(fs *pflag.FlagSet) (ptx.Options, error) {
	var o ptx.Options
	b := func(name string) bool { v, _ := fs.GetBool(name); return v }
	str := func(name string) string { v, _ := fs.GetString(name); return v }
	o.AutoReference = b("auto-reference")
	o.Traditional = b("traditional")
	o.FlagTruncation, o.FlagTruncationSet = str("flag-truncation"), fs.Changed("flag-truncation")
	o.MacroName = str("macro-name")
	o.Format = *fs.Lookup("format").Value.(*format).sel
	o.RightSideRefs = b("right-side-refs")
	o.SentenceRegexp, o.SentenceRegexpSet = str("sentence-regexp"), fs.Changed("sentence-regexp")
	o.WordRegexp = str("word-regexp")
	o.BreakFile = str("break-file")
	o.IgnoreCase = b("ignore-case")
	o.IgnoreFile = str("ignore-file")
	o.OnlyFile = str("only-file")
	o.References = b("references")
	var err error
	if fs.Changed("gap-size") {
		if o.GapSize, err = positive(str("gap-size"), "gap width"); err != nil {
			return o, err
		}
	}
	if fs.Changed("width") {
		if o.Width, err = positive(str("width"), "line width"); err != nil {
			return o, err
		}
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var f *fatal
	if errors.As(err, &f) {
		d := core.NewDiag(ptx.Name, s)
		d.Errorf("%s", f.msg)
		return d.Result()
	}
	if err != nil {
		return err
	}
	return ptx.New(opts, s).Run(args)
}

// Meta returns the documentation metadata for ptxx.
func Meta() docmeta.Meta {
	n := ptx.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"sortx", "grepx", "tsortx"},
		Use:     n + " [OPTION]... [INPUT]...   (without -G)\n" + n + " -G [OPTION]... [INPUT [OUTPUT]]",
		Short:   "Produce a permuted index of the words in a text.",
		Long: n + " lists every word of the INPUTs (standard input when there are none, or for " +
			"-) in alphabetical order, each in the middle of a line with the text before it " +
			"on the left and the text after it on the right: a keyword-in-context index. A word " +
			"is a run of letters, or what -W's regular expression matches, or what is between " +
			"-b's break characters; a context ends at the end of a sentence, or where -S says. " +
			"-i leaves out the words of a file, one per line, and -o keeps only those. -A " +
			"prefixes each line with FILE:LINE, and -r takes the first word of each input line " +
			"as its reference; -R puts references on the right. -w sets the width (72), -g the " +
			"gap between fields (3), -F the mark of a cut context ('/'). -O and -T write roff or " +
			"TeX macro calls instead (-M names the macro), and -G behaves like System V ptx, with " +
			"words between blanks, contexts that are lines, roff output, and an optional OUTPUT " +
			"file. Regular expressions use the Emacs syntax GNU ptx uses, \\( \\) \\| for groups " +
			"and alternatives, run by Go's engine, which has no back references.",
		Permissions: "Needs read permission on each INPUT and on the -b, -i and -o files; with -G, " +
			"write permission to create OUTPUT.",
		UseCases: []string{
			"Building a keyword-in-context index for a book or a manual.",
			"Seeing every place a word is used in a set of notes, in context.",
			"Producing an index for typesetting with roff or TeX.",
		},
		Examples: []docmeta.Example{
			{Description: "See every word of a text in context:", Bash: n + " notes.txt", PowerShell: n + " notes.txt"},
			{Description: "Index a manual, skipping common words, with file and line references:",
				Bash: n + " -A -i stopwords.txt chapter*.txt", PowerShell: n + " -A -i stopwords.txt (Get-ChildItem chapter*.txt)"},
			{Description: "Index only the words that matter:", Bash: n + " -o terms.txt manual.txt", PowerShell: n + " -o terms.txt manual.txt"},
			{Description: "Produce TeX macro calls for typesetting:", Bash: n + " -T -M index book.txt > index.tex", PowerShell: n + " -T -M index book.txt > index.tex"},
			{Description: "Treat each line as its own context, ignoring case when sorting:", Bash: n + " -f -S '\\n' titles.txt", PowerShell: n + " -f -S '\\n' titles.txt"},
		},
	}
}
