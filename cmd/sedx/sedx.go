// Package sedx wires the sedx command (GNU sed) to its engine.
package sedx

import (
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sed"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the sedx tool. GNU sed's -i takes an optional suffix glued
// to it (-i.bak), which the flag parser cannot express for a short option,
// so the arguments arrive raw and are parsed here after rewriteArgs has
// spelled such options out.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "sed",
		Flags:   Register,
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for sedx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    sed.Name,
		SeeAlso: []string{"trx", "cutx", "headx", "tailx", "csplitx", "gnux shell"},
		Use:     sed.Name + " [OPTION]... {script-only-if-no-other-script} [input-file]...",
		Short:   "Filter and transform text with a stream editor.",
		Long: "sedx reads its input one line at a time into the pattern space, runs the " +
			"script on it and, unless -n is given, prints what is left. The input is the " +
			"input-files in order as one stream, or standard input when there is none or " +
			"the file is -. The script is the first operand, or every -e SCRIPT and -f " +
			"SCRIPT-FILE in command-line order when at least one is given. It is compiled " +
			"in full before anything is read, so a mistake in it changes no file.\n\n" +
			"A script is a list of commands separated by newlines or semicolons, each " +
			"optionally preceded by an address: a line number N, $ for the last line, " +
			"/REGEX/ or \\cREGEXc (with I to ignore case and M to let ^ and $ match at " +
			"every line of the pattern space), FIRST~STEP, or a range ADDR1,ADDR2 where " +
			"ADDR2 may also be +N (N more lines) or ~N (up to the next multiple of N), and " +
			"0,/REGEX/ lets the range end on the first line. A ! after the address selects " +
			"the other lines. An empty regular expression stands for the last one used.\n\n" +
			"The commands are those of GNU sed: { } to group; s/REGEX/REPLACEMENT/FLAGS " +
			"with any delimiter, the flags g, p, a number, i or I, m or M and w FILE, and in " +
			"the replacement &, \\0 to \\9, \\n, and the case conversions \\L \\U \\l \\u \\E; " +
			"y/SOURCE/DEST/; d D p P n N g G h H x; a, i and c in both the one-line form " +
			"(a TEXT) and the POSIX form (a\\ followed by the text on the next lines); = l " +
			"q Q with an optional exit status; r FILE, R FILE, w FILE and W FILE, where " +
			"/dev/stdin, /dev/stdout and /dev/stderr are the standard streams; b, t and T " +
			"with a : LABEL; z; F; v; and # comments. A script whose first line is #n runs " +
			"as with -n.\n\n" +
			"Regular expressions are POSIX basic ones with the GNU extensions \\+ \\? \\| " +
			"or, with -E, extended ones; both take \\w \\W \\s \\S \\b \\B \\< \\> \\` \\' and " +
			"the character escapes \\n \\t \\f \\v \\a \\r \\cX \\dNNN \\oNNN \\xHH, and both " +
			"find the leftmost longest match.\n\n" +
			"-i edits each file in place: the output is written to a temporary file in the " +
			"same directory, which takes the file's permissions and then its name; with a " +
			"SUFFIX the original is kept under that name (a * in SUFFIX stands for the " +
			"file's base name, and a SUFFIX containing a slash names another directory). " +
			"The suffix must be attached: -i.bak or --in-place=.bak. A missing newline at " +
			"the end of the last line is missing from the output too.\n\n" +
			"Differences from GNU sed: the e command and the e flag of s are refused, as " +
			"sedx never runs another program. A back reference inside a regular expression " +
			"(\\(a\\)\\1) is refused, because the linear-time matcher cannot express one; " +
			"\\1 in a replacement works. \\< and \\> both match at any word boundary, like " +
			"\\b, and word characters are ASCII letters, digits and the underscore. A " +
			"repetition count may be at most 1000. Text is taken as UTF-8 whatever the " +
			"locale, a regular expression must itself be valid UTF-8, and l shows every " +
			"non-ASCII byte in octal. --debug is not supported, and long options cannot be " +
			"abbreviated. --posix only changes what n and N do at the end of the input " +
			"(nothing is printed); the GNU extensions stay available. -u flushes the " +
			"output after every line but still reads the input in blocks. -i does not " +
			"carry over a file's owner, group or extended attributes. A value of -l that " +
			"is not a number is an error rather than 0, and running with no script is a " +
			"one-line error rather than the full usage text.",
		Permissions: "Needs read permission on each input-file, script file and file read by r " +
			"or R. w, W and the w flag of s create or truncate the file they name as soon as " +
			"the script has been compiled, which needs write permission on it or on its " +
			"directory. -i needs write permission on the directory of each file, since the " +
			"file is replaced by a new one, not rewritten: the new file has the old one's " +
			"permission bits but belongs to the user running sedx, any hard link keeps the " +
			"old contents, and a symbolic link is replaced by a regular file unless " +
			"--follow-symlinks is given. --sandbox refuses any script that reads or writes " +
			"a file of its own (r, R, w, W).",
		UseCases: []string{
			"Replacing a string or a pattern in a stream or, with -i, in a set of files.",
			"Printing a range of lines of a file, by number or between two patterns.",
			"Deleting or commenting out the lines of a configuration file that match a pattern.",
			"Inserting, appending or replacing lines at a known place in a generated file.",
			"Rewriting fields with groups, such as swapping two columns or reformatting dates.",
			"Running one edit script the same way on Linux, macOS and Windows, where the system sed differs or is missing.",
		},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "success"},
			{Code: "1", Meaning: "an invalid command, syntax, regular expression or option"},
			{Code: "2", Meaning: "an input file could not be opened; the others were still processed"},
			{Code: "4", Meaning: "an I/O error while running: sedx stopped at once"},
			{Code: "N", Meaning: "the exit status given to the q or Q command"},
		},
		Examples: []docmeta.Example{
			{
				Description: "Replace every occurrence of a word in a stream:",
				Bash:        "sedx 's/staging/production/g' deploy.yaml",
				PowerShell:  "sedx 's/staging/production/g' deploy.yaml",
			},
			{
				Description: "Point a config file at a new host, keeping the original as config.ini.bak:",
				Bash:        "sedx -i.bak 's/^host = .*/host = db2.internal/' config.ini",
				PowerShell:  "sedx '-i.bak' 's/^host = .*/host = db2.internal/' config.ini",
			},
			{
				Description: "Print only lines 10 to 20 of a log:",
				Bash:        "sedx -n '10,20p' app.log",
				PowerShell:  "sedx -n '10,20p' app.log",
			},
			{
				Description: "Show the block between two markers in a file:",
				Bash:        "sedx -n '/BEGIN CERTIFICATE/,/END CERTIFICATE/p' bundle.pem",
				PowerShell:  "sedx -n '/BEGIN CERTIFICATE/,/END CERTIFICATE/p' bundle.pem",
			},
			{
				Description: "Strip comments and blank lines from a config file:",
				Bash:        "sedx -e 's/#.*//' -e '/^[[:space:]]*$/d' nginx.conf",
				PowerShell:  "sedx -e 's/#.*//' -e '/^[[:space:]]*$/d' nginx.conf",
			},
			{
				Description: "Swap the first two comma-separated fields with extended regular expressions:",
				Bash:        "sedx -E 's/^([^,]*),([^,]*)/\\2,\\1/' people.csv",
				PowerShell:  "sedx -E 's/^([^,]*),([^,]*)/\\2,\\1/' people.csv",
				Output:      "Lovelace,Ada,1815\nHopper,Grace,1906",
			},
			{
				Description: "Add a line after the one that matches, in every YAML file of a directory:",
				Bash:        "sedx -i '/^dependencies:/a\\  - gnux' *.yaml",
				PowerShell:  "sedx -i '/^dependencies:/a\\  - gnux' (Get-ChildItem *.yaml).Name",
			},
			{
				Description: "Join all the lines of a file with commas:",
				Bash:        "sedx ':a;N;$!ba;s/\\n/,/g' ids.txt",
				PowerShell:  "sedx ':a;N;$!ba;s/\\n/,/g' ids.txt",
				Output:      "17,23,42",
			},
			{
				Description: "Print the first three lines and stop reading, like headx -n 3:",
				Bash:        "sedx 3q huge.log",
				PowerShell:  "sedx 3q huge.log",
			},
			{
				Description: "Capitalize the first letter of every word:",
				Bash:        "sedx 's/\\b\\w/\\u&/g' title.txt",
				PowerShell:  "sedx 's/\\b\\w/\\u&/g' title.txt",
				Output:      "The Quick Brown Fox",
			},
			{
				Description: "Remove trailing white space from every source file, in place:",
				Bash:        "sedx -i 's/[[:space:]]*$//' main.go util.go",
				PowerShell:  "sedx -i 's/[[:space:]]*$//' main.go util.go",
			},
			{
				Description: "Run an edit script kept in a file:",
				Bash:        "sedx -f cleanup.sed report.txt > report.clean.txt",
				PowerShell:  "sedx -f cleanup.sed report.txt > report.clean.txt",
			},
			{
				Description: "Rewrite a path in the portable gnux shell:",
				Bash:        "gnux shell -c \"echox /usr/local/bin | sedx 's|/usr/local|/opt|'\"",
				PowerShell:  "gnux shell -c \"echox /usr/local/bin | sedx 's|/usr/local|/opt|'\"",
				Output:      "/opt/bin",
			},
		},
	}
}

// inPlaceValue is the value of -i. The flag is declared as a boolean with an
// optional value so that the help screen shows it as a plain switch; a
// suffix reaches it marked with a leading '=' (see rewriteArgs), which keeps
// a suffix spelled "true" apart from the bare option.
type inPlaceValue struct {
	set    bool
	suffix string
}

// noSuffix is what the parser hands -i when it is given without a suffix.
const noSuffix = "true"

func (v *inPlaceValue) String() string { return "false" }
func (v *inPlaceValue) Type() string   { return "bool" }
func (v *inPlaceValue) Set(s string) error {
	v.set = true
	v.suffix = ""
	if s != noSuffix {
		v.suffix = strings.TrimPrefix(s, "=")
	}
	return nil
}

// scriptValue is the value of -e and of -f: both append to one list, which
// keeps the order the two options were given in.
type scriptValue struct {
	scripts *[]sed.Script
	file    bool
}

func (v *scriptValue) String() string { return "" }
func (v *scriptValue) Type() string   { return "string" }
func (v *scriptValue) Set(s string) error {
	*v.scripts = append(*v.scripts, sed.Script{Text: s, File: v.file})
	return nil
}

// Register installs the sedx flags on fs.
func Register(fs *pflag.FlagSet) {
	scripts := new([]sed.Script)
	fs.BoolP("quiet", "n", false, "suppress automatic printing of pattern space")
	fs.Bool("silent", false, "same as --quiet")
	fs.VarP(&scriptValue{scripts: scripts}, "expression", "e", "add the `SCRIPT` to the commands to be executed")
	fs.VarP(&scriptValue{scripts: scripts, file: true}, "file", "f", "add the contents of `SCRIPT-FILE` to the commands to be executed")
	fs.Bool("follow-symlinks", false, "follow symlinks when processing in place")
	fs.VarPF(&inPlaceValue{}, "in-place", "i", "edit files in place; -iSUFFIX or --in-place=SUFFIX also keeps each original under that suffix").NoOptDefVal = noSuffix
	fs.StringP("line-length", "l", "", "specify the desired line-wrap length `N` for the 'l' command (0 never wraps; default 70)")
	fs.Bool("posix", false, "print nothing when n or N finds no next line, as POSIX has it")
	fs.BoolP("regexp-extended", "E", false, "use extended regular expressions in the script (-r is accepted too)")
	fs.BoolP("separate", "s", false, "consider files as separate rather than as a single continuous long stream")
	fs.Bool("sandbox", false, "operate in sandbox mode (refuse the r, R, w and W commands)")
	fs.BoolP("unbuffered", "u", false, "flush the output after every line")
	fs.BoolP("null-data", "z", false, "separate lines by NUL characters")
	fs.BoolP("binary", "b", false, "open files in binary mode (always the case here)")
}

// valueOptions are the short options that take a value, with their long
// names.
var valueOptions = map[byte]string{'e': "expression", 'f': "file", 'l': "line-length"}

// rewriteArgs spells out the forms of GNU sed's short options the flag
// parser would misread: -iSUFFIX becomes --in-place==SUFFIX (the rest of the
// argument is the suffix, even when it looks like more options, as with
// getopt; the second '=' marks it as a suffix for inPlaceValue, and the one
// of --in-place=SUFFIX is doubled the same way), a value attached to -e, -f
// or -l is kept whole (-e=x is the script "=x"), and -r becomes -E. Operands
// and the values of options are left untouched.
func rewriteArgs(args []string) []string {
	out := make([]string, 0, len(args)+2)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return append(out, args[i:]...)
		case strings.HasPrefix(arg, "--in-place="):
			out = append(out, "--in-place=="+strings.TrimPrefix(arg, "--in-place="))
		case strings.HasPrefix(arg, "--"):
			out = append(out, arg)
			// A long option given without = takes the next argument.
			for _, name := range valueOptions {
				if arg == "--"+name && i+1 < len(args) {
					i++
					out = append(out, args[i])
				}
			}
		case len(arg) > 1 && arg[0] == '-':
			flags := []byte(arg)
			var tail []string
		cluster:
			for j := 1; j < len(flags); j++ {
				switch c := flags[j]; {
				case c == 'r':
					flags[j] = 'E'
				case c == 'i' && j+1 < len(flags):
					tail = []string{"--in-place==" + arg[j+1:]}
					flags = flags[:j]
					break cluster
				case valueOptions[c] != "" && j+1 < len(flags):
					tail = []string{"--" + valueOptions[c] + "=" + arg[j+1:]}
					flags = flags[:j]
					break cluster
				case valueOptions[c] != "":
					// The value is the next argument.
					if i+1 < len(args) {
						i++
						tail = []string{args[i]}
					}
					break cluster
				}
			}
			if len(flags) > 1 {
				out = append(out, string(flags))
			}
			out = append(out, tail...)
		default:
			out = append(out, arg)
		}
	}
	return out
}

// buildOptions folds the parsed flags into sed.Options.
func buildOptions(fs *pflag.FlagSet) (sed.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := sed.Options{
		Quiet:          on("quiet") || on("silent"),
		Extended:       on("regexp-extended"),
		FollowSymlinks: on("follow-symlinks"),
		Separate:       on("separate"),
		NullData:       on("null-data"),
		Unbuffered:     on("unbuffered"),
		Posix:          on("posix"),
		Sandbox:        on("sandbox"),
	}
	if v, ok := fs.Lookup("expression").Value.(*scriptValue); ok {
		o.Scripts = *v.scripts
	}
	if v, ok := fs.Lookup("in-place").Value.(*inPlaceValue); ok {
		o.InPlace, o.Suffix = v.set, v.suffix
	}
	if fs.Changed("line-length") {
		text, _ := fs.GetString("line-length")
		n, err := strconv.Atoi(text)
		if err != nil || n < 0 {
			return sed.Options{}, core.Usagef("invalid line length '%s'", text)
		}
		// Zero asks for no wrapping, which the engine spells as negative:
		// its zero is the default length.
		o.LineLength = n
		if n == 0 {
			o.LineLength = -1
		}
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	if err := fs.Parse(rewriteArgs(args)); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			// What the parser says for -h when the tool has no such option.
			return core.Usagef("unknown shorthand flag: 'h' in -h")
		}
		return core.Usagef("%s", err)
	}
	if help, _ := fs.GetBool("help"); help {
		Spec().WriteHelp(s.Out, fs, tool.StyleFor(s.Out))
		return nil
	}
	if version, _ := fs.GetBool("version"); version {
		tool.PrintVersion(s.Out, sed.Name)
		return nil
	}
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return sed.New(opts, s).Run(fs.Args())
}
