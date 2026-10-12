// Package fmtx wires the fmtx command (GNU fmt) to its engine.
package fmtx

import (
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fmt"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/obsolete"
	"github.com/braswelljr/gnux/internal/tool"
)

// maxWidth is the largest width GNU fmt accepts.
const maxWidth = 2500

// Spec describes the fmtx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "fmt",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for fmtx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    fmt.Name,
		SeeAlso: []string{"foldx", "nlx"},
		Use:     fmt.Name + " [-WIDTH] [OPTION]... [FILE]...",
		Short:   "Reformat each paragraph of text.",
		Long: "fmtx fills and joins the lines of each paragraph so that they come as close as " +
			"possible to a goal width without exceeding the maximum width, and writes the " +
			"result to standard output. With no FILE, or when FILE is -, it reads standard " +
			"input. Paragraphs are separated by blank lines and by changes of indentation; " +
			"the indentation of each paragraph is kept. The line breaks are chosen as GNU fmt " +
			"chooses them, minimizing raggedness over the whole paragraph and avoiding " +
			"breaking after the first word of a sentence or before its last. -w WIDTH sets " +
			"the maximum width (75 by default, at most 2500), also written -WIDTH; -g WIDTH " +
			"sets the goal (93% of the maximum by default, and without -w the maximum becomes " +
			"the goal plus 10). -c (crown margin) keeps the indentation of the first two lines " +
			"and aligns the rest with the second; -t (tagged paragraph) is the same, except " +
			"that a first line indented like the second is a paragraph of its own. -s only " +
			"splits long lines and never joins short ones, which suits code. -u puts one space " +
			"between words and two after sentences. -p STRING reformats only the lines that " +
			"begin with STRING (after any indentation) and puts it back on each output line; " +
			"the other lines are copied unchanged. A sentence ends at '.', '?' or '!' followed " +
			"by the end of the line or two spaces, as in GNU fmt.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Rewrapping a commit message, README paragraph or email to a fixed width.",
			"Joining hard-wrapped text into evenly filled lines.",
			"Reflowing comment blocks with -p '#' or -p '//'.",
			"Splitting overlong lines without joining short ones with -s.",
			"Normalizing spacing between words and sentences with -u.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Tidy up a text file whose lines were edited into ragged lengths:",
				Bash:        "fmtx notes.txt",
				PowerShell:  "fmtx notes.txt",
			},
			{
				Description: "Rewrap a commit message to the 72 columns git log expects:",
				Bash:        "fmtx -w 72 message.txt",
				PowerShell:  "fmtx -w 72 message.txt",
			},
			{
				Description: "Run an old script that uses the obsolete width form:",
				Bash:        "fmtx -60 notes.txt",
				PowerShell:  "fmtx -60 notes.txt",
			},
			{
				Description: "Shorten overlong lines in a script or list without merging its lines:",
				Bash:        "fmtx -s -w 80 script.txt",
				PowerShell:  "fmtx -s -w 80 script.txt",
			},
			{
				Description: "Reflow a block of shell comments after editing them:",
				Bash:        "fmtx -p '#' -w 72 config.sh",
				PowerShell:  "fmtx -p '#' -w 72 config.sh",
			},
			{
				Description: "Normalize spacing in a document pasted from several sources:",
				Bash:        "fmtx -u draft.txt",
				PowerShell:  "fmtx -u draft.txt",
			},
			{
				Description: "Reflow a list whose items have a hanging indent:",
				Bash:        "fmtx -t list.txt",
				PowerShell:  "fmtx -t list.txt",
			},
			{
				Description: "Reflow a letter that indents its first line:",
				Bash:        "fmtx -c letter.txt",
				PowerShell:  "fmtx -c letter.txt",
			},
			{
				Description: "Aim for even 70-column lines, never exceeding 80:",
				Bash:        "fmtx -w 80 -g 70 notes.txt",
				PowerShell:  "fmtx -w 80 -g 70 notes.txt",
			},
			{
				Description: "Reflow text coming out of another command:",
				Bash:        "catx a.txt b.txt | fmtx -w 66",
				PowerShell:  "catx a.txt b.txt | fmtx -w 66",
			},
			{
				Description: "Save a cleanly wrapped final copy of a draft:",
				Bash:        "fmtx -w 72 draft.txt > final.txt",
				PowerShell:  "fmtx -w 72 draft.txt > final.txt",
			},
			{
				Description: "Preview a reflowed file in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'fmtx -w 60 notes.txt | headx'",
				PowerShell:  "gnux shell -c 'fmtx -w 60 notes.txt | headx'",
			},
		},
	}
}

// Register installs the fmtx flags on fs.
func Register(fs *pflag.FlagSet) {
	ord := &obsolete.Order{}
	fs.BoolP("crown-margin", "c", false, "preserve indentation of first two lines")
	fs.StringP("prefix", "p", "", "reformat only lines beginning with `STRING`, reattaching the prefix to reformatted lines")
	fs.BoolP("split-only", "s", false, "split long lines, but do not refill")
	fs.BoolP("tagged-paragraph", "t", false, "indentation of first line different from second")
	fs.BoolP("uniform-spacing", "u", false, "one space between words, two after sentences")
	fs.VarP(obsolete.NewString(ord, "WIDTH"), "width", "w", "maximum line width (default of 75 columns)")
	fs.StringP("goal", "g", "", "goal width (default of 93% of width)")
	obsolete.Register(fs, ord)
}

// buildOptions folds the parsed flags into fmt.Options.
func buildOptions(fs *pflag.FlagSet) (fmt.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := fmt.Options{
		Crown:   on("crown-margin"),
		Tagged:  on("tagged-paragraph"),
		Split:   on("split-only"),
		Uniform: on("uniform-spacing"),
	}
	o.Prefix, _ = fs.GetString("prefix")

	if text, ok := obsolete.Last(fs.Lookup("width").Value.(*obsolete.String), obsolete.Lookup(fs)); ok {
		n, err := width(text, maxWidth)
		if err != nil {
			return fmt.Options{}, err
		}
		o.Width, o.WidthSet = n, true
	}
	if fs.Changed("goal") {
		text, _ := fs.GetString("goal")
		limit := fmt.DefaultWidth
		if o.WidthSet {
			limit = o.Width
		}
		n, err := width(text, limit)
		if err != nil {
			return fmt.Options{}, err
		}
		o.Goal, o.GoalSet = n, true
	}
	return o, nil
}

// width parses a width between 0 and limit.
func width(text string, limit int) (int, error) {
	n, err := strconv.ParseUint(text, 10, 31)
	if err != nil || int(n) > limit {
		return 0, core.Usagef("invalid width: '%s'", text)
	}
	return int(n), nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return fmt.New(opts, s).Run(args)
}
