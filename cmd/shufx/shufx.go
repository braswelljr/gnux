// Package shufx wires the shufx command (GNU shuf) to its engine.
package shufx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shuf"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the shufx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "shuf",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for shufx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    shuf.Name,
		SeeAlso: []string{"sortx", "headx", "seqx"},
		Use:     shuf.Name + " [OPTION]... [FILE]",
		Short:   "Generate random permutations.",
		Long: "shufx writes a random permutation of its input lines to standard output. The " +
			"input is the lines of FILE, or of standard input when FILE is absent or -; with " +
			"-e, each ARG operand is an input line instead (shufx -e [OPTION]... [ARG]...); " +
			"with -i LO-HI, the numbers LO through HI are (shufx -i LO-HI [OPTION]...). -n " +
			"COUNT writes at most COUNT lines (given more than once, the smallest COUNT " +
			"wins, as in GNU shuf). -r draws with replacement, so lines may repeat, and with " +
			"no -n it writes forever until its reader goes away. -o FILE writes to FILE " +
			"instead of standard output; it is opened only after the input has been read, so " +
			"FILE may also be the input. --random-source=FILE takes the random bytes from " +
			"FILE, so the same FILE always gives the same order; without it the order comes " +
			"from a generator seeded by the operating system's random source. -z uses NUL " +
			"rather than newline as the line delimiter. Differences from GNU shuf: the " +
			"permutation drawn from a given --random-source differs from the one GNU shuf " +
			"draws from the same bytes, though it is just as reproducible.",
		Permissions: "Needs read permission on FILE and on the --random-source file, and " +
			"search permission on the directories leading to them. With -o, needs write " +
			"permission on the output file, or on its directory when it does not exist yet.",
		UseCases: []string{
			"Putting the lines of a file in random order.",
			"Picking a random sample of N lines with -n.",
			"Choosing one item at random from a list of operands with -e -n 1.",
			"Generating random numbers in a range with -i, with or without repeats.",
			"Shuffling a playlist, a deck or a test order.",
			"Getting a reproducible shuffle in tests with --random-source.",
			"Producing an endless stream of random choices with -r.",
			"Splitting a data set into random training and test portions.",
		},
		Examples: []docmeta.Example{
			{Description: "Shuffle the lines of a file:", Bash: "shufx names.txt", PowerShell: "shufx names.txt"},
			{Description: "Pick three random lines:", Bash: "shufx -n 3 names.txt", PowerShell: "shufx -n 3 names.txt"},
			{Description: "Pick one of several words:", Bash: "shufx -e -n 1 red green blue", PowerShell: "shufx -e -n 1 red green blue"},
			{Description: "Shuffle the numbers 1 to 10:", Bash: "shufx -i 1-10", PowerShell: "shufx -i 1-10"},
			{Description: "Roll a die five times (numbers may repeat):", Bash: "shufx -i 1-6 -r -n 5", PowerShell: "shufx -i 1-6 -r -n 5"},
			{Description: "Shuffle standard input:", Bash: "seqx 5 | shufx", PowerShell: "1..5 | shufx"},
			{Description: "Shuffle a file in place:", Bash: "shufx -o deck.txt deck.txt", PowerShell: "shufx -o deck.txt deck.txt"},
			{Description: "A reproducible order for tests:", Bash: "shufx --random-source=seed.bin cases.txt", PowerShell: "shufx --random-source=seed.bin cases.txt"},
			{Description: "Pick a random file from a NUL-separated list:", Bash: "find . -type f -print0 | shufx -z -n 1", PowerShell: "lsx -1 | shufx -n 1"},
			{Description: "An endless stream of coin flips, cut short by headx:", Bash: "shufx -r -e heads tails | headx -n 4", PowerShell: "shufx -r -e heads tails | headx -n 4"},
			{Description: "Sample 100 lines for a quick look at a large log:", Bash: "shufx -n 100 access.log > sample.log", PowerShell: "shufx -n 100 access.log > sample.log"},
			{Description: "Shuffle in the portable gnux shell:", Bash: "gnux shell -c 'lsx -1 | shufx -n 2'", PowerShell: "gnux shell -c 'lsx -1 | shufx -n 2'"},
		},
	}
}

// Register installs the shufx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("echo", "e", false, "treat each ARG as an input line")
	fs.StringP("input-range", "i", "", "treat each number `LO-HI` as an input line")
	fs.StringArrayP("head-count", "n", nil, "output at most `COUNT` lines")
	fs.StringP("output", "o", "", "write result to `FILE` instead of standard output")
	fs.String("random-source", "", "get random bytes from `FILE`")
	fs.BoolP("repeat", "r", false, "output lines can be repeated")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")
}

// buildOptions folds the parsed flags into shuf.Options. A bad -n COUNT is
// returned as a plain error: GNU reports it without the --help hint.
func buildOptions(fs *pflag.FlagSet) (shuf.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := shuf.Options{
		Echo:           on("echo"),
		Repeat:         on("repeat"),
		ZeroTerminated: on("zero-terminated"),
	}
	o.InputRange, _ = fs.GetString("input-range")
	o.Output, _ = fs.GetString("output")
	o.RandomSource, _ = fs.GetString("random-source")

	counts, _ := fs.GetStringArray("head-count")
	for _, c := range counts {
		n, err := shuf.ParseCount(c)
		if err != nil {
			return shuf.Options{}, err
		}
		// GNU shuf keeps the smallest of several -n values.
		if !o.HeadCountSet || n < o.HeadCount {
			o.HeadCount = n
		}
		o.HeadCountSet = true
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		core.NewDiag(shuf.Name, s).Errorf("%s", err.Error())
		return core.ErrFailed
	}
	return shuf.New(opts, s).Run(args)
}
