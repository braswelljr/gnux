// Package dux wires the dux command (GNU du) to its engine.
package dux

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/du"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the dux tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "du",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for dux.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    du.Name,
		SeeAlso: []string{"dfx", "lsx", "statx", "sortx"},
		Use:     du.Name + " [OPTION]... [FILE]...",
		Short:   "Estimate file space usage.",
		Long: "dux shows how much disk space each directory under each FILE (the current " +
			"directory by default) takes, including everything below it, in 1024-byte blocks " +
			"rounded up. -a lists files too, -s only the operands, and -d N only N levels " +
			"down; -c adds a grand total. -h prints sizes like 1.5K, 23M and 4.0G (--si in " +
			"powers of 1000), -k and -m use KiB and MiB, -b bytes, and -B SIZE any unit " +
			"('-BM' prints the unit too). --apparent-size counts the bytes files hold rather " +
			"than the blocks allocated to them, which differs for sparse and compressed " +
			"files. --inodes counts files instead of space. -S leaves subdirectories out of " +
			"a directory's total, -x stays on one file system, -l counts a hard-linked file " +
			"every time it is met (by default it is counted once), -L follows symbolic links " +
			"and -H only those given as operands. -t SIZE hides entries smaller than SIZE " +
			"(or, negative, larger). --exclude=PATTERN and -X FILE skip matching paths. " +
			"--time adds the latest modification (or --time=atime, ctime) time in each " +
			"subtree, formatted by --time-style. --files0-from=F reads NUL-separated " +
			"operands from F, and -0 ends output lines with NUL. Where GNU du reads one " +
			"directory at a time and prints entries in whatever order the file system " +
			"returns them, dux reads directories in parallel, which is much faster on large " +
			"trees and network drives, and then adds everything up in name order, so the " +
			"output, and which path a hard-linked file is counted under, is the same every " +
			"run. The block size defaults to DU_BLOCK_SIZE, BLOCK_SIZE or BLOCKSIZE when " +
			"set. Names are quoted only when output goes to a terminal." +
			" On a terminal, dux shows gnux's view (--pretty=auto, the default; " +
			"GNUX_PRETTY changes it): sizes aligned and colored by magnitude, a bar and a " +
			"percentage giving each line's share of its operand, and the last part of each name " +
			"highlighted, in the theme --theme or GNUX_THEME names (NO_COLOR keeps it uncolored). " +
			"Piped or redirected output, and -0, stay exactly GNU's tab-separated lines; " +
			"--pretty=always asks for the view anyway and --pretty=never turns it off.",
		Permissions: "Needs read and search permission on every directory measured; an " +
			"unreadable directory is reported and its own size still counted.",
		UseCases: []string{
			"Seeing which subdirectory takes most of a folder's space, from the bars and percentages.",
			"Finding which directories fill a disk.",
			"Checking how big a project, cache or build output has grown.",
			"Comparing disk usage with apparent size to spot sparse files.",
			"Counting files (inodes) where a file system runs out of inodes before space.",
			"Monitoring a directory's size in a script or cron job.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Your home folder grew by 20 GB: see each folder's share of it as a bar, one level deep:",
				Bash:        "dux -h -d 1 ~",
				PowerShell:  "dux -h -d 1 $HOME",
			},
			{
				Description: "Your disk is nearly full: list the 10 biggest directories in your home, largest first:",
				Bash:        "dux -h -d 1 ~ | sortx -h -r | headx -n 11",
				PowerShell:  "dux -h -d 1 $HOME | sortx -h -r | headx -n 11",
				Output:      "48G\t/home/alice\n21G\t/home/alice/Videos\n12G\t/home/alice/.cache\n6.3G\t/home/alice/projects",
			},
			{
				Description: "Show the total size of a project, in human-readable units:",
				Bash:        "dux -sh ~/projects/website",
				PowerShell:  "dux -sh $HOME\\projects\\website",
				Output:      "1.2G\t/home/alice/projects/website",
			},
			{
				Description: "See how much node_modules folders take across all your projects:",
				Bash:        "dux -sch ~/projects/*/node_modules",
				PowerShell:  "dux -sch (Get-ChildItem $HOME\\projects\\*\\node_modules -Directory).FullName",
				Output:      "412M\t/home/alice/projects/api/node_modules\n388M\t/home/alice/projects/web/node_modules\n800M\ttotal",
			},
			{
				Description: "Measure a repository without its .git history:",
				Bash:        "dux -sh --exclude=.git ~/projects/website",
				PowerShell:  "dux -sh --exclude=.git $HOME\\projects\\website",
			},
			{
				Description: "Find files over 100 MB to clean up:",
				Bash:        "dux -a -t 100M -h ~/Downloads",
				PowerShell:  "dux -a -t 100M -h $HOME\\Downloads",
			},
			{
				Description: "Spot a sparse VM disk image: it takes far less space than its size:",
				Bash:        "dux -h vm.qcow2 && dux -h --apparent-size vm.qcow2",
				PowerShell:  "dux -h vm.vhdx; dux -h --apparent-size vm.vhdx",
				Output:      "2.1G\tvm.qcow2\n40G\tvm.qcow2",
			},
			{
				Description: "Find which directory holds millions of tiny files (out of inodes):",
				Bash:        "dux --inodes -d 1 /var | sortx -n -r | headx",
				PowerShell:  "dux --inodes -d 1 C:\\ProgramData | sortx -n -r | headx",
			},
			{
				Description: "Get a byte count for a script, without units:",
				Bash:        "bytes=$(dux -sb dist | cutx -f1)",
				PowerShell:  "$bytes = (dux -sb dist) -split \"`t\" | Select-Object -First 1",
				Output:      "5242880",
			},
			{
				Description: "Check only the root file system, not mounted drives under it:",
				Bash:        "sudo dux -xh -d 1 /",
				PowerShell:  "dux -xh -d 1 C:\\",
			},
			{
				Description: "See which log directories changed most recently:",
				Bash:        "dux -sh --time /var/log/*",
				PowerShell:  "dux -sh --time (Get-ChildItem C:\\logs -Directory).FullName",
				Output:      "1.3M\t2026-10-12 09:41\t/var/log/apt\n96M\t2026-10-12 10:02\t/var/log/journal",
			},
			{
				Description: "Report sizes in megabytes with the unit shown:",
				Bash:        "dux -BM -d 1 /srv",
				PowerShell:  "dux -BM -d 1 D:\\srv",
				Output:      "120M\t/srv/www\n2048M\t/srv/backups\n2169M\t/srv",
			},
			{
				Description: "Measure a list of paths produced by another program:",
				Bash:        "find . -name '*.log' -print0 | dux -ch --files0-from=-",
				PowerShell:  "dux -ch --files0-from=paths.lst",
			},
			{
				Description: "Show the biggest folders in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'dux -h -d 1 . | sortx -h -r | headx -n 5'",
				PowerShell:  "gnux shell -c 'dux -h -d 1 . | sortx -h -r | headx -n 5'",
			},
		},
	}
}

// unitValue is the shared value of -b, -h, --si, -k, -m and -B: whichever is
// given last decides the unit, as in GNU.
type unitValue struct {
	spec *string
	set  string // the block size the flag stands for; "" means its argument
}

func (v *unitValue) String() string { return "" }

// Type is what --help shows as the value placeholder.
func (v *unitValue) Type() string {
	if v.set == "" {
		return "SIZE"
	}
	return "bool"
}

// IsBoolFlag lets the unit switches take no argument.
func (v *unitValue) IsBoolFlag() bool { return v.set != "" }

func (v *unitValue) Set(s string) error {
	if v.set != "" {
		*v.spec = v.set
		return nil
	}
	*v.spec = s
	return nil
}

// Register installs the dux flags on fs.
func Register(fs *pflag.FlagSet) {
	unit := new(string)
	flag := func(name, short, set, usage string) {
		f := fs.VarPF(&unitValue{spec: unit, set: set}, name, short, usage)
		if set != "" {
			f.NoOptDefVal = "true"
		}
	}
	fs.BoolP("null", "0", false, "end each output line with NUL, not newline")
	fs.BoolP("all", "a", false, "write counts for all files, not just directories")
	fs.BoolP("apparent-size", "A", false, "print apparent sizes rather than device usage")
	flag("block-size", "B", "", "scale sizes by SIZE before printing them; e.g., '-BM' prints sizes in units of 1,048,576 bytes")
	flag("bytes", "b", "1", "equivalent to '--apparent-size --block-size=1'")
	fs.BoolP("total", "c", false, "produce a grand total")
	fs.BoolP("dereference-args", "D", false, "dereference only symlinks that are listed on the command line")
	fs.StringP("max-depth", "d", "", "print the total for a directory (or file, with --all) only if it is `N` or fewer levels below the command line argument")
	fs.String("files0-from", "", "summarize device usage of the NUL-terminated file names specified in file `F`; if F is -, then read names from standard input")
	fs.BoolP("dereference-args-h", "H", false, "equivalent to --dereference-args (-D)")
	fs.Lookup("dereference-args-h").Hidden = true
	flag("human-readable", "h", "human-readable", "print sizes in human readable format (e.g., 1K 234M 2G)")
	fs.Bool("inodes", false, "list inode usage information instead of block usage")
	flag("kilobytes", "k", "1K", "like --block-size=1K")
	fs.BoolP("dereference", "L", false, "dereference all symbolic links")
	fs.BoolP("count-links", "l", false, "count sizes many times if hard linked")
	flag("megabytes", "m", "1M", "like --block-size=1M")
	fs.BoolP("no-dereference", "P", false, "don't follow any symbolic links (this is the default)")
	fs.BoolP("separate-dirs", "S", false, "for directories do not include size of subdirectories")
	flag("si", "", "si", "like -h, but use powers of 1000 not 1024")
	fs.BoolP("summarize", "s", false, "display only a total for each argument")
	fs.StringP("threshold", "t", "", "exclude entries smaller than `SIZE` if positive, or entries greater than SIZE if negative")
	fs.String("time", "", "show time of the last modification of any file in the directory, or any of its subdirectories; with `WORD`: atime, access, use, ctime or status")
	fs.Lookup("time").NoOptDefVal = "mtime"
	fs.String("time-style", "", "show times using `STYLE`, which can be: full-iso, long-iso, iso, or +FORMAT")
	fs.StringArrayP("exclude-from", "X", nil, "exclude files that match any pattern in `FILE`")
	fs.StringArray("exclude", nil, "exclude files that match `PATTERN`")
	fs.BoolP("one-file-system", "x", false, "skip directories on different file systems")

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "kilobytes", "megabytes")
	tool.RegisterView(fs)
}

// buildOptions folds the parsed flags into du.Options.
func buildOptions(fs *pflag.FlagSet) (du.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	str := func(name string) string {
		s, _ := fs.GetString(name)
		return s
	}
	o := du.Options{
		All:           on("all"),
		Apparent:      on("apparent-size") || fs.Changed("bytes"),
		Total:         on("total"),
		SeparateDirs:  on("separate-dirs"),
		CountLinks:    on("count-links"),
		OneFileSystem: on("one-file-system"),
		Inodes:        on("inodes"),
		Summarize:     on("summarize"),
		Null:          on("null"),
		TimeStyle:     str("time-style"),
		Files0From:    str("files0-from"),
	}
	o.BlockSize = *fs.Lookup("block-size").Value.(*unitValue).spec
	o.Exclude, _ = fs.GetStringArray("exclude")
	o.ExcludeFrom, _ = fs.GetStringArray("exclude-from")

	// The most cautious traversal wins, whatever the order.
	switch {
	case on("no-dereference"):
		o.Deref = du.DerefNone
	case on("dereference-args") || on("dereference-args-h"):
		o.Deref = du.DerefArgs
	case on("dereference"):
		o.Deref = du.DerefAll
	}

	if fs.Changed("max-depth") {
		n, err := strconv.ParseInt(str("max-depth"), 0, 32)
		if err != nil || n < 0 {
			return o, core.Usagef("invalid maximum depth '%s'", str("max-depth"))
		}
		o.MaxDepth, o.MaxDepthSet = int(n), true
	}
	if fs.Changed("threshold") {
		n, err := threshold(str("threshold"))
		if err != nil {
			return o, err
		}
		o.Threshold = n
	}
	if fs.Changed("time") {
		t, err := timeWord(str("time"))
		if err != nil {
			return o, err
		}
		o.Time = t
	}
	return o, nil
}

// fatalError is a mistake GNU du reports without the --help hint.
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

// threshold reads -t: a signed number with an optional K, M, G, ... suffix.
func threshold(text string) (int64, error) {
	bad := &fatalError{"invalid --threshold argument '" + text + "'"}
	s := text
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(strings.TrimPrefix(s, "-"), "+")
	digits := 0
	for digits < len(s) && s[digits] >= '0' && s[digits] <= '9' {
		digits++
	}
	if digits == 0 {
		return 0, bad
	}
	n, err := strconv.ParseInt(s[:digits], 10, 64)
	if err != nil {
		return 0, &fatalError{"--threshold argument '" + text + "' too large"}
	}
	if suffix := s[digits:]; suffix != "" {
		power := strings.IndexByte("KMGTPEZYRQ", upperByte(suffix[0])) + 1
		unit := int64(1024)
		switch {
		case power == 0 || (suffix[0] >= 'a' && suffix[0] != 'k' && suffix[0] != 'm'):
			return 0, bad
		case suffix[1:] == "B":
			unit = 1000
		case suffix[1:] == "" || suffix[1:] == "iB":
		default:
			return 0, bad
		}
		for range power {
			if n > (1<<63-1)/unit {
				return 0, &fatalError{"--threshold argument '" + text + "' too large"}
			}
			n *= unit
		}
	}
	if neg {
		if n == 0 {
			return 0, &fatalError{"invalid --threshold argument '-0'"}
		}
		n = -n
	}
	return n, nil
}

func upperByte(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

// timeWord reads --time's WORD, abbreviations included.
func timeWord(w string) (string, error) {
	if w == "mtime" {
		return w, nil
	}
	words := []struct{ word, time string }{
		{"atime", "atime"}, {"access", "atime"}, {"use", "atime"}, {"ctime", "ctime"}, {"status", "ctime"},
	}
	found := ""
	for _, c := range words {
		if c.word == w {
			return c.time, nil
		}
		if strings.HasPrefix(c.word, w) && w != "" {
			if found != "" && found != c.time {
				return "", core.Usagef("ambiguous argument '%s' for '--time'", w)
			}
			found = c.time
		}
	}
	if found == "" {
		return "", core.Usagef("invalid argument '%s' for '--time'\nValid arguments are:\n  - 'atime', 'access', 'use'\n  - 'ctime', 'status'", w)
	}
	return found, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var fatal *fatalError
	if errors.As(err, &fatal) {
		d := core.NewDiag(du.Name, s)
		d.Errorf("%s", fatal.msg)
		return d.Result()
	}
	if err != nil {
		return err
	}
	view, err := tool.ViewOf(fs, core.IsTerminal(s.Out), du.Name, os.Getenv, s.Err)
	if err != nil {
		return err
	}
	opts.Pretty, opts.Color, opts.Theme = view.Pretty, view.Color, view.Theme
	return du.New(opts, s).Run(args)
}
