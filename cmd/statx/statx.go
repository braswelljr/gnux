// Package statx wires the statx command (GNU stat) to its engine.
package statx

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/stat"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the statx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "stat",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for statx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    stat.Name,
		SeeAlso: []string{"lsx", "dfx", "dux", "chmodx", "touchx"},
		Use:     stat.Name + " [OPTION]... FILE...",
		Short:   "Display file or file system status.",
		Long: "statx prints everything the system records about each FILE: its type, size, " +
			"blocks, device, inode, links, permissions, owner and group, and its access, " +
			"modification, change and birth times. A symbolic link is described itself " +
			"unless -L follows it. -f describes the file system holding FILE instead: its " +
			"type, block sizes, and total, free and available blocks and inodes. -t prints " +
			"the same fields on one line, for scripts. -c FORMAT prints only what FORMAT asks " +
			"for, followed by a newline; --printf FORMAT does the same but interprets " +
			"backslash escapes (\\n, \\t, \\x41, \\101) and adds no newline; when both are " +
			"given, the last wins. FORMAT takes GNU's directives: %n (name), %N (quoted name, " +
			"with its target for a link), %s (size), %F (file type), %a and %A (permissions " +
			"in octal and as rwx), %u %U %g %G (owner and group), %h (links), %i (inode), %b " +
			"%B %o (blocks, block unit, I/O size), %d %D %Hd %Ld (device), %r %R %t %T %Hr " +
			"%Lr (device type of a special file), %m (mount point), %w %x %y %z (birth, " +
			"access, modification and change times) and %W %X %Y %Z (the same as seconds " +
			"since the epoch, which take a precision: %.3Y), with printf flags and widths " +
			"(%-10s, %04a). With -f: %n %i %l %t %T %s %S %b %f %a %c %d. %N quotes a name " +
			"(and a link's target) in the style the QUOTING_STYLE variable names, as lsx " +
			"--quoting-style takes it, or else for the shell, always ('a.txt', 'a b', " +
			"\"it's\", 'a'$'\\n''b'); the default formats name files as they are. The device is shown as major,minor, as GNU " +
			"stat 9 shows it. Missing data is reported honestly rather than invented: the " +
			"birth time prints '-' where the system does not record it (Linux without statx), " +
			"and on Windows, which has no numeric owners, %u and %g print '?'. SELinux " +
			"contexts (%C) are not supported. --cached is accepted for compatibility and has " +
			"no effect.",
		Permissions: "Needs search permission on the directories leading to each FILE; the " +
			"FILE itself need not be readable. Owner and group names are read from the " +
			"system's account databases.",
		UseCases: []string{
			"Finding out exactly when a file was last modified, accessed or created.",
			"Getting a file's size or permissions in a script without parsing lsx -l.",
			"Checking which file system, and how much free space, a path is on.",
			"Comparing inodes and devices to tell whether two paths are the same file.",
			"Auditing ownership and permissions of sensitive files.",
		},
		Examples: []docmeta.Example{
			{
				Description: "See everything about a config file before editing it:",
				Bash:        "statx /etc/hosts",
				PowerShell:  "statx C:\\Windows\\System32\\drivers\\etc\\hosts",
				Output: "  File: /etc/hosts\n  Size: 221       \tBlocks: 8          IO Block: 4096   regular file\n" +
					"Device: 259,2\tInode: 1835014     Links: 1\n" +
					"Access: (0644/-rw-r--r--)  Uid: (    0/    root)   Gid: (    0/    root)\n" +
					"Access: 2026-10-12 08:02:11.120375871 +0000\nModify: 2026-09-30 17:44:02.006417455 +0000\n" +
					"Change: 2026-09-30 17:44:02.006417455 +0000\n Birth: 2026-03-01 10:15:40.773102148 +0000",
			},
			{
				Description: "Get a file's size in bytes in a script, without parsing lsx -l:",
				Bash:        "size=$(statx -c %s backup.tar.gz)",
				PowerShell:  "$size = statx -c %s backup.tar.gz",
				Output:      "1073741824",
			},
			{
				Description: "Check that an SSH key is private before ssh refuses to use it:",
				Bash:        "statx -c '%a %U %n' ~/.ssh/id_ed25519",
				PowerShell:  "statx -c '%a %U %n' $HOME\\.ssh\\id_ed25519",
				Output:      "600 alice /home/alice/.ssh/id_ed25519",
			},
			{
				Description: "Find out when a log file last changed, as seconds since the epoch, to alert if it went quiet:",
				Bash:        "[ $(( $(datex +%s) - $(statx -c %Y app.log) )) -gt 600 ] && echox 'app.log silent for 10 minutes'",
				PowerShell:  "if (((datex +%s) - (statx -c %Y app.log)) -gt 600) { 'app.log silent for 10 minutes' }",
			},
			{
				Description: "List several files with their permissions and owners, aligned:",
				Bash:        "statx -c '%-25n %A %U:%G' /etc/passwd /etc/shadow /etc/sudoers",
				PowerShell:  "statx -c '%-25n %A' (Get-ChildItem *.ps1 -Name)",
				Output:      "/etc/passwd               -rw-r--r-- root:root\n/etc/shadow               -rw-r----- root:shadow\n/etc/sudoers              -r--r----- root:root",
			},
			{
				Description: "Tell whether two paths are the same file (same device and inode):",
				Bash:        "[ \"$(statx -L -c %d:%i a)\" = \"$(statx -L -c %d:%i b)\" ] && echox 'same file'",
				PowerShell:  "if ((statx -L -c %d:%i a) -eq (statx -L -c %d:%i b)) { 'same file' }",
			},
			{
				Description: "See where a symbolic link points:",
				Bash:        "statx -c %N /usr/bin/python3",
				PowerShell:  "statx -c %N link",
				Output:      "/usr/bin/python3 -> python3.12",
			},
			{
				Description: "Check the file system type and free space under a directory:",
				Bash:        "statx -f /var/lib/docker",
				PowerShell:  "statx -f C:\\",
			},
			{
				Description: "Print modification times with millisecond precision, tab-separated:",
				Bash:        "statx --printf '%n\\t%.3Y\\n' *.csv",
				PowerShell:  "statx --printf '%n\\t%.3Y\\n' (Get-ChildItem *.csv -Name)",
				Output:      "jan.csv\t1767225600.123\nfeb.csv\t1769904000.456",
			},
			{
				Description: "Find a file's mount point:",
				Bash:        "statx -c %m ~/Downloads/movie.mkv",
				PowerShell:  "statx -c %m $HOME\\Downloads\\movie.mkv",
				Output:      "/home",
			},
			{
				Description: "Record a file's full status on one line, for a log or a diff:",
				Bash:        "statx -t release.zip >> inventory.txt",
				PowerShell:  "statx -t release.zip >> inventory.txt",
			},
			{
				Description: "Show file details in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'statx -c \"%n: %s bytes, modified %y\" report.pdf'",
				PowerShell:  "gnux shell -c 'statx -c \"%n: %s bytes, modified %y\" report.pdf'",
			},
		},
	}
}

// formatValue is the shared value of -c and --printf: whichever is given
// last decides the format, as in GNU.
type formatValue struct {
	f      *formats
	printf bool
}

type formats struct {
	text   string
	printf bool
	set    bool
}

func (v *formatValue) String() string { return "" }

// Type is what --help shows as the value placeholder.
func (v *formatValue) Type() string { return "FORMAT" }

func (v *formatValue) Set(s string) error {
	*v.f = formats{text: s, printf: v.printf, set: true}
	return nil
}

// Register installs the statx flags on fs.
func Register(fs *pflag.FlagSet) {
	f := &formats{}
	fs.BoolP("dereference", "L", false, "follow links")
	fs.BoolP("file-system", "f", false, "display file system status instead of file status")
	fs.String("cached", "", "specify how to use cached attributes (always, never or default); accepted, no effect")
	fs.VarP(&formatValue{f: f}, "format", "c", "use the specified FORMAT instead of the default; output a newline after each use of FORMAT")
	fs.Var(&formatValue{f: f, printf: true}, "printf", "like --format, but interpret backslash escapes, and do not output a mandatory trailing newline")
	fs.BoolP("terse", "t", false, "print the information in terse form")
}

// buildOptions folds the parsed flags into stat.Options.
func buildOptions(fs *pflag.FlagSet) (stat.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := stat.Options{
		Dereference: on("dereference"),
		FileSystem:  on("file-system"),
		Terse:       on("terse"),
	}
	if fs.Changed("cached") {
		mode, _ := fs.GetString("cached")
		if err := checkCached(mode); err != nil {
			return o, err
		}
	}
	if f := fs.Lookup("format").Value.(*formatValue).f; f.set {
		if f.printf {
			o.Printf, o.PrintfSet = f.text, true
		} else {
			o.Format, o.FormatSet = f.text, true
		}
	}
	return o, nil
}

// checkCached validates --cached as GNU's argmatch does, abbreviations
// included.
func checkCached(mode string) error {
	var found []string
	for _, w := range []string{"default", "never", "always"} {
		if w == mode {
			return nil
		}
		if mode != "" && strings.HasPrefix(w, mode) {
			found = append(found, w)
		}
	}
	if len(found) == 1 {
		return nil
	}
	problem := "invalid"
	if len(found) > 1 {
		problem = "ambiguous"
	}
	return core.Usagef("%s argument '%s' for '--cached'\nValid arguments are:\n  - 'default'\n  - 'never'\n  - 'always'", problem, mode)
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return stat.New(opts, s).Run(args)
}
