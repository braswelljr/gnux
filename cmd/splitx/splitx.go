// Package splitx wires the splitx command (GNU split) to its engine.
package splitx

import (
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/head"
	"github.com/braswelljr/gnux/core/split"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/obsolete"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the splitx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "split",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for splitx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    split.Name,
		SeeAlso: []string{"csplitx", "catx", "headx", "tailx"},
		Use:     split.Name + " [OPTION]... [FILE [PREFIX]]",
		Short:   "Split a file into pieces.",
		Long: "splitx copies FILE (standard input when FILE is - or absent) into files named " +
			"PREFIX (x by default) plus a suffix: xaa, xab, and so on, 1000 lines per file. " +
			"-l N puts N lines in each file (the obsolete -N form too), -b SIZE puts SIZE " +
			"bytes, and -C SIZE as many whole lines as fit in SIZE bytes, splitting only a " +
			"line longer than that. SIZE takes the suffixes K, M, G, ... (powers of 1024) and " +
			"KB, MB, ... (powers of 1000). -n CHUNKS makes a fixed number of files instead: N " +
			"files of nearly equal size (the first ones a byte longer when the size does not " +
			"divide), l/N without splitting lines (each line goes to the chunk it starts in), " +
			"or r/N dealing lines out in turn; K/N, l/K/N and r/K/N write only the Kth chunk " +
			"to standard output. To size the chunks of a pipe, splitx first copies it to a " +
			"temporary file, as GNU does. Suffixes are two letters, and when they run out " +
			"they grow (xyz is followed by xzaaa) so the names still sort in order; -a N " +
			"fixes their length, -d and -x make them decimal or hexadecimal, " +
			"--numeric-suffixes=FROM and --hex-suffixes=FROM start them at FROM, and " +
			"--additional-suffix appends a fixed string such as .csv. -t SEP splits lines on " +
			"SEP instead of newline ('\\0' for NUL), -e skips empty files in the -n modes, and " +
			"--verbose reports each file as it is created. splitx refuses to overwrite FILE " +
			"with one of its pieces. Only one way of splitting may be given. GNU's --filter, " +
			"which pipes each piece through a shell command, is not implemented; -u is " +
			"accepted and has no effect, since output is never held back. The digits of two " +
			"obsolete -N options in a row run together (-5 -3 reads as -53).",
		Permissions: "Needs read permission on FILE, and write permission on the directory the " +
			"pieces are created in (and on any existing piece, which is overwritten).",
		UseCases: []string{
			"Breaking a file too large for an email attachment or a FAT32 drive into pieces.",
			"Dividing a big CSV or log into batches for parallel processing.",
			"Spreading work over N workers with -n r/N or -n l/N.",
			"Sampling one slice of a large file with -n K/N without writing the rest.",
			"Reassembling the pieces later with catx, since the names sort in order.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Cut a 10 GB backup into 2 GB pieces that fit on a FAT32 USB drive:",
				Bash:        "splitx -b 2G backup.tar.gz backup.tar.gz.part-",
				PowerShell:  "splitx -b 2G backup.tar.gz backup.tar.gz.part-",
				Output:      "(creates backup.tar.gz.part-aa ... backup.tar.gz.part-ae)",
			},
			{
				Description: "Put the pieces back together on the other machine:",
				Bash:        "catx backup.tar.gz.part-* > backup.tar.gz",
				PowerShell:  "gnux shell -c 'catx backup.tar.gz.part-* > backup.tar.gz'",
			},
			{
				Description: "Break a million-row CSV into files of 100,000 rows for an import tool that chokes on big files:",
				Bash:        "tailx -n +2 users.csv | splitx -l 100000 -d --additional-suffix=.csv - users-",
				PowerShell:  "tailx -n +2 users.csv | splitx -l 100000 -d --additional-suffix=.csv - users-",
				Output:      "(creates users-00.csv ... users-09.csv)",
			},
			{
				Description: "Split a log into 4 roughly equal parts for 4 parallel workers, without cutting a line in half:",
				Bash:        "splitx -n l/4 access.log chunk-",
				PowerShell:  "splitx -n l/4 access.log chunk-",
			},
			{
				Description: "Deal a list of URLs out to 3 crawlers in turn:",
				Bash:        "splitx -n r/3 urls.txt crawler-",
				PowerShell:  "splitx -n r/3 urls.txt crawler-",
			},
			{
				Description: "Look at just the third quarter of a huge file without writing any pieces:",
				Bash:        "splitx -n l/3/4 huge.log | headx",
				PowerShell:  "splitx -n l/3/4 huge.log | headx",
			},
			{
				Description: "Send a log in email-sized pieces of at most 5 MB, keeping lines whole:",
				Bash:        "splitx -C 5M app.log app.log.",
				PowerShell:  "splitx -C 5M app.log app.log.",
			},
			{
				Description: "Number the pieces from 1 instead of 0:",
				Bash:        "splitx -l 500 --numeric-suffixes=1 -a 3 list.txt list-",
				PowerShell:  "splitx -l 500 --numeric-suffixes=1 -a 3 list.txt list-",
				Output:      "(creates list-001, list-002, ...)",
			},
			{
				Description: "See each file as it is created:",
				Bash:        "splitx --verbose -l 2 names.txt",
				PowerShell:  "splitx --verbose -l 2 names.txt",
				Output:      "creating file 'xaa'\ncreating file 'xab'",
			},
			{
				Description: "Split NUL-separated records, such as the output of find -print0:",
				Bash:        "find . -type f -print0 | splitx -t '\\0' -l 1000 - batch-",
				PowerShell:  "splitx -t '\\0' -l 1000 files.lst batch-",
			},
			{
				Description: "Split stdin into 1,000-line files the obsolete way, as old scripts do:",
				Bash:        "seqx 2500 | splitx -1000",
				PowerShell:  "seqx 2500 | splitx -1000",
			},
			{
				Description: "Split and reassemble in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'splitx -b 1M video.mp4 part- && catx part-* > copy.mp4'",
				PowerShell:  "gnux shell -c 'splitx -b 1M video.mp4 part- && catx part-* > copy.mp4'",
			},
		},
	}
}

// suffixValue is the shared value of -d and -x: whichever is given last
// decides the alphabet, as in GNU.
type suffixValue struct {
	alphabet *split.Alphabet
	start    *string
	kind     split.Alphabet
}

// noStart is the value of -d or -x given without =FROM.
const noStart = "\x00"

func (v *suffixValue) String() string { return "" }

// Type is what --help shows as the value placeholder.
func (v *suffixValue) Type() string { return "FROM" }

func (v *suffixValue) Set(s string) error {
	*v.alphabet, *v.start = v.kind, ""
	if s != noStart {
		*v.start = s
	}
	return nil
}

// suffixes records -d and -x on a flag set.
type suffixes struct {
	alphabet split.Alphabet
	start    string
}

// Register installs the splitx flags on fs.
func Register(fs *pflag.FlagSet) {
	sfx := &suffixes{}
	fs.StringP("suffix-length", "a", "", "generate suffixes of length `N` (default 2)")
	fs.String("additional-suffix", "", "append an additional `SUFFIX` to file names")
	fs.StringP("bytes", "b", "", "put `SIZE` bytes per output file")
	fs.StringP("line-bytes", "C", "", "put at most `SIZE` bytes of records per output file")
	f := fs.VarPF(&suffixValue{&sfx.alphabet, &sfx.start, split.Numeric}, "numeric-suffixes", "d",
		"use numeric suffixes starting at 0, not alphabetic; --numeric-suffixes=FROM starts at FROM")
	f.NoOptDefVal = noStart
	f = fs.VarPF(&suffixValue{&sfx.alphabet, &sfx.start, split.Hex}, "hex-suffixes", "x",
		"use hex suffixes starting at 0, not alphabetic; --hex-suffixes=FROM starts at FROM")
	f.NoOptDefVal = noStart
	fs.BoolP("elide-empty-files", "e", false, "do not generate empty output files with '-n'")
	fs.StringP("lines", "l", "", "put `NUMBER` lines/records per output file")
	fs.StringP("number", "n", "", "generate `CHUNKS` output files: N, K/N, l/N, l/K/N, r/N or r/K/N")
	fs.StringArrayP("separator", "t", nil, "use `SEP` instead of newline as the record separator; '\\0' (zero) specifies the NUL character")
	fs.BoolP("unbuffered", "u", false, "immediately copy input to output with '-n r/...' (always the case here)")
	fs.Bool("verbose", false, "print a diagnostic just before each output file is opened")
	obsolete.Register(fs, &obsolete.Order{})
}

// fatalError is a mistake GNU split reports without the --help hint.
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

func fatalf(what, arg string) error { return &fatalError{what + ": '" + arg + "'"} }

// buildOptions folds the parsed flags into split.Options.
func buildOptions(fs *pflag.FlagSet) (split.Options, error) {
	str := func(name string) string {
		s, _ := fs.GetString(name)
		return s
	}
	var o split.Options
	o.ElideEmpty, _ = fs.GetBool("elide-empty-files")
	o.Verbose, _ = fs.GetBool("verbose")
	o.AdditionalSuffix = str("additional-suffix")

	sfx := fs.Lookup("numeric-suffixes").Value.(*suffixValue)
	o.Suffixes, o.SuffixStart = *sfx.alphabet, *sfx.start

	if fs.Changed("suffix-length") {
		n, err := strconv.ParseUint(str("suffix-length"), 10, 31)
		if err != nil {
			return o, fatalf("invalid suffix length", str("suffix-length"))
		}
		o.SuffixLength = int(n)
	}

	ways := 0
	num := obsolete.Lookup(fs)
	for _, name := range []string{"bytes", "line-bytes", "lines", "number"} {
		if fs.Changed(name) {
			ways++
		}
	}
	if num.Given() {
		ways++
	}
	if ways > 1 {
		return o, core.Usagef("cannot split in more than one way")
	}
	positive := func(text, what string, sizes bool) (int64, error) {
		var n int64
		var err error
		if sizes {
			n, err = head.ParseSize(text)
		} else {
			n, err = strconv.ParseInt(text, 10, 64)
			if errors.Is(err, strconv.ErrRange) {
				n, err = 1<<63-1, nil
			}
		}
		if err != nil || n < 1 {
			return 0, fatalf(what, text)
		}
		return n, nil
	}
	var err error
	switch {
	case fs.Changed("bytes"):
		o.Mode = split.Bytes
		o.Count, err = positive(str("bytes"), "invalid number of bytes", true)
	case fs.Changed("line-bytes"):
		o.Mode = split.LineBytes
		o.Count, err = positive(str("line-bytes"), "invalid number of lines", true)
	case fs.Changed("lines"):
		o.Count, err = positive(str("lines"), "invalid number of lines", false)
	case num.Given():
		o.Count, err = positive(num.Text, "invalid number of lines", false)
	case fs.Changed("number"):
		err = parseChunks(strings.TrimLeft(str("number"), " \t\n\v\f\r"), &o)
	}
	if err != nil {
		return o, err
	}

	for _, sep := range fs.Lookup("separator").Value.(pflag.SliceValue).GetSlice() {
		var c byte
		switch {
		case sep == "":
			return o, &fatalError{"empty record separator"}
		case sep == `\0`:
			c = 0
		case len(sep) > 1:
			return o, fatalf("multi-character separator", sep)
		default:
			c = sep[0]
		}
		if o.SeparatorSet && c != o.Separator {
			return o, &fatalError{"multiple separator characters specified"}
		}
		o.Separator, o.SeparatorSet = c, true
	}
	return o, nil
}

// parseChunks reads -n: N, K/N, l/N, l/K/N, r/N or r/K/N.
func parseChunks(arg string, o *split.Options) error {
	o.Mode = split.ChunkBytes
	switch {
	case strings.HasPrefix(arg, "l/"):
		o.Mode, arg = split.ChunkLines, arg[2:]
	case strings.HasPrefix(arg, "r/"):
		o.Mode, arg = split.RoundRobin, arg[2:]
	}
	parse := func(s string) (int64, bool) {
		n, err := strconv.ParseInt(s, 10, 64)
		if errors.Is(err, strconv.ErrRange) {
			return 1<<63 - 1, true
		}
		return n, err == nil && n > 0
	}
	k, n, slash := strings.Cut(arg, "/")
	if !slash {
		count, ok := parse(arg)
		if !ok {
			return fatalf("invalid number of chunks", arg)
		}
		o.Count = count
		return nil
	}
	count, ok := parse(n)
	if !ok {
		return fatalf("invalid number of chunks", n)
	}
	chunk, err := strconv.ParseInt(k, 10, 64)
	if err != nil || chunk < 1 || chunk > count {
		return fatalf("invalid chunk number", k)
	}
	o.Count, o.Chunk = count, chunk
	return nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var fatal *fatalError
	if errors.As(err, &fatal) {
		d := core.NewDiag(split.Name, s)
		d.Errorf("%s", fatal.msg)
		return d.Result()
	}
	if err != nil {
		return err
	}
	return split.New(opts, s).Run(args)
}
