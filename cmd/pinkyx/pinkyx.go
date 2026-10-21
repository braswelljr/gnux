// Package pinkyx wires the pinkyx command (GNU pinky) to its engine.
package pinkyx

import (
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/pinky"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the pinkyx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "pinky",
		Flags: Register,
		Run:   run,
	}
}

// format is the -s and -l switch pair: the one given last wins, as in GNU.
type format struct {
	long bool
	sel  *bool
}

func (f *format) Set(v string) error {
	on, err := strconv.ParseBool(v)
	if err == nil && on {
		*f.sel = f.long
	}
	return err
}
func (f *format) String() string   { return strconv.FormatBool(f.sel != nil && *f.sel == f.long) }
func (f *format) Type() string     { return "bool" }
func (f *format) IsBoolFlag() bool { return true }

// Register installs the pinkyx flags on fs. GNU spells every option but
// --lookup with a letter only; the long names are gnux's own.
func Register(fs *pflag.FlagSet) {
	long := new(bool)
	for _, f := range []struct {
		name, short, usage string
		long               bool
	}{
		{"long", "l", "produce long format output for the specified USERs", true},
		{"short", "s", "do short format output, this is the default", false},
	} {
		flag := fs.VarPF(&format{long: f.long, sel: long}, f.name, f.short, f.usage)
		flag.NoOptDefVal = "true"
	}
	fs.BoolP("omit-home-and-shell", "b", false, "omit the user's home directory and shell in long format")
	fs.BoolP("omit-project", "h", false, "omit the user's project file in long format")
	fs.BoolP("omit-plan", "p", false, "omit the user's plan file in long format")
	fs.BoolP("omit-heading", "f", false, "omit the line of column headings in short format")
	fs.BoolP("omit-name", "w", false, "omit the user's full name in short format")
	fs.BoolP("omit-where", "i", false, "omit the user's full name and remote host in short format")
	fs.BoolP("omit-idle", "q", false, "omit the user's full name, remote host and idle time in short format")
	fs.Bool("lookup", false, "attempt to canonicalize hostnames via DNS")
	tool.OwnNames(fs, "long", "short", "omit-home-and-shell", "omit-project", "omit-plan",
		"omit-heading", "omit-name", "omit-where", "omit-idle")
}

// buildOptions folds the parsed flags into pinky.Options.
func buildOptions(fs *pflag.FlagSet) pinky.Options {
	b := func(name string) bool { v, _ := fs.GetBool(name); return v }
	var o pinky.Options
	o.Long = *fs.Lookup("long").Value.(*format).sel
	o.NoHomeAndShell = b("omit-home-and-shell")
	o.NoProject = b("omit-project")
	o.NoPlan = b("omit-plan")
	o.NoHeading = b("omit-heading")
	q, i := b("omit-idle"), b("omit-where")
	o.NoFullName = b("omit-name") || i || q
	o.NoWhere = i || q
	o.NoIdle = q
	o.Lookup = b("lookup")
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return pinky.New(buildOptions(fs), s).Run(args)
}

// Meta returns the documentation metadata for pinkyx.
func Meta() docmeta.Meta {
	n := pinky.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"whox", "usersx", "idx"},
		Use:     n + " [OPTION]... [USER]...",
		Short:   "Show who is logged in, or what is known about users: a lightweight finger.",
		Long: n + " prints one line per login session (only the given USERs', if any): the " +
			"login name, the full name, the terminal (marked * when it does not accept " +
			"messages), how long it has been idle, when the session started and where from. " +
			"-f drops the heading, -w the full name, -i the full name and the remote host, and " +
			"-q the idle time too. With -l it prints, for each USER, the login name, the full " +
			"name, the home directory and shell (unless -b), and the contents of ~/.project " +
			"(unless -h) and ~/.plan (unless -p). The full name is the user's GECOS name, with " +
			"& standing for the capitalized login name. Sessions come from the system's login " +
			"records, as for whox; on Windows they are the signed-in console and Remote Desktop " +
			"sessions. The shell is read from /etc/passwd, so it is blank for a user kept " +
			"elsewhere (macOS directory services, LDAP) and on Windows. If both -s and -l are " +
			"given, the last one wins.",
		Permissions: "Reads the login records, the user database and other users' ~/.project and " +
			"~/.plan files when they are readable. Changes nothing.",
		UseCases: []string{
			"Seeing who is logged in to a shared machine, with names people recognize.",
			"Looking up a colleague's full name, home directory and shell.",
			"Checking how long a terminal has been idle before contacting its user.",
		},
		Examples: []docmeta.Example{
			{Description: "See who is logged in:", Bash: n, PowerShell: n,
				Output: "Login    Name                 TTY      Idle   When         Where\nalice    Alice Liddell       *pts/0    00:12  Oct 20 09:14 10.0.0.7"},
			{Description: "Look up a user's details:", Bash: n + " -l alice", PowerShell: n + " -l alice",
				Output: "Login name: alice                       In real life:  Alice Liddell\nDirectory: /home/alice                   Shell:  /bin/bash"},
			{Description: "Only the login names, terminals and times, for a script:", Bash: n + " -fq", PowerShell: n + " -fq"},
			{Description: "Check whether one user is logged in:", Bash: n + " -f bob | headx -n 1", PowerShell: n + " -f bob | headx -n 1"},
		},
	}
}
