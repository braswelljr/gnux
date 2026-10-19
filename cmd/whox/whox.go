// Package whox wires the whox command (GNU who) to its engine.
package whox

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/who"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the whox tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "who",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for whox.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    who.Name,
		SeeAlso: []string{"whoamix", "idx", "uptimex", "ttyx"},
		Use:     who.Name + " [OPTION]... [ FILE | ARG1 ARG2 ]",
		Short:   "Show who is logged on.",
		Long: "whox lists the users logged in: name, terminal, login time and, for a remote " +
			"session, the host in parentheses. -u adds how long each terminal has been idle " +
			"('.' under a minute, 'old' over a day) and the session's process ID, -T (or -w) " +
			"whether the user accepts messages (+, - or ?), and -H a line of headings. -q " +
			"prints just the names and their count. -b shows the last boot, -r the run level, " +
			"-l login processes, -p processes started by init, -d dead processes and -t clock " +
			"changes; -a shows all of them. -m (or any two operands, as in 'whox am i') " +
			"shows only the terminal on standard input. --lookup turns host addresses into " +
			"names. With FILE, the records are read from that utmp file (on Linux, " +
			"/var/log/wtmp lists past logins too) instead of the current ones, and in the " +
			"current ones, logins whose process has ended are left out. Times are in the C " +
			"locale's 'Oct 12 09:30' format. The records are read without the C library, in " +
			"each system's own format: Linux utmp, macOS utmpx and FreeBSD utx.active. On " +
			"Windows, which keeps sessions rather than a utmp file, whox lists the signed-in " +
			"console and Remote Desktop sessions, so the same command answers 'who is on " +
			"this machine' everywhere; there FILE is not supported.",
		Permissions: "Needs read permission on the login records, which every user has.",
		UseCases: []string{
			"Checking who else is logged in to a shared server before rebooting it.",
			"Seeing which remote hosts users are connected from.",
			"Finding idle sessions to clean up with -u.",
			"Checking when the machine last booted with -b.",
			"Counting logged-in users in a monitoring script with -q.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Before rebooting a shared server, see who is still logged in:",
				Bash:        "whox",
				PowerShell:  "whox",
				Output:      "alice    pts/0        Oct 12 09:30 (10.0.0.5)\nbob      pts/1        Oct 12 13:02 (laptop.lan)",
			},
			{
				Description: "See how long each session has been idle, with headings:",
				Bash:        "whox -uH",
				PowerShell:  "whox -uH",
				Output: "NAME     LINE         TIME         IDLE          PID COMMENT\n" +
					"alice    pts/0        Oct 12 09:30 01:30        1234 (10.0.0.5)\n" +
					"bob      pts/1        Oct 12 13:02   .          1300 (laptop.lan)",
			},
			{
				Description: "Check when the machine last booted:",
				Bash:        "whox -b",
				PowerShell:  "whox -b",
				Output:      "         system boot  Oct  9 10:44",
			},
			{
				Description: "Count the logged-in users for a monitoring check:",
				Bash:        "whox -q",
				PowerShell:  "whox -q",
				Output:      "alice bob\n# users=2",
			},
			{
				Description: "Find out which terminal and login you are on:",
				Bash:        "whox am i",
				PowerShell:  "whox am i",
				Output:      "alice    pts/0        Oct 12 09:30 (10.0.0.5)",
			},
			{
				Description: "See who will receive a 'wall' broadcast (+ accepts messages):",
				Bash:        "whox -T",
				PowerShell:  "whox -T",
			},
			{
				Description: "Show every record, including dead processes and the run level:",
				Bash:        "whox -a",
				PowerShell:  "whox -a",
			},
			{
				Description: "Show the remote hosts by name instead of address:",
				Bash:        "whox --lookup",
				PowerShell:  "whox --lookup",
			},
			{
				Description: "Review past logins from the Linux login history:",
				Bash:        "whox /var/log/wtmp | tailx",
				PowerShell:  "whox",
			},
			{
				Description: "Warn in a script when anyone besides you is logged in:",
				Bash:        "[ \"$(whox -q | tailx -n 1 | cutx -d= -f2)\" -gt 1 ] && echox 'others are logged in'",
				PowerShell:  "if ([int]((whox -q | tailx -n 1) -split '=')[1] -gt 1) { 'others are logged in' }",
			},
			{
				Description: "List logged-in users in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'whox -q && uptimex'",
				PowerShell:  "gnux shell -c 'whox -q && uptimex'",
			},
		},
	}
}

// Register installs the whox flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("all", "a", false, "same as -b -d --login -p -r -t -T -u")
	fs.BoolP("boot", "b", false, "time of last system boot")
	fs.BoolP("dead", "d", false, "print dead processes")
	fs.BoolP("heading", "H", false, "print line of column headings")
	fs.BoolP("login", "l", false, "print system login processes")
	fs.Bool("lookup", false, "attempt to canonicalize hostnames via DNS")
	fs.BoolP("my-line", "m", false, "only hostname and user associated with stdin")
	fs.BoolP("process", "p", false, "print active processes spawned by init")
	fs.BoolP("count", "q", false, "all login names and number of users logged on")
	fs.BoolP("runlevel", "r", false, "print current runlevel")
	fs.BoolP("short", "s", false, "print only name, line, and time (default)")
	fs.BoolP("time", "t", false, "print last system clock change")
	fs.BoolP("mesg", "T", false, "add user's message status as +, - or ?")
	fs.BoolP("writable", "w", false, "same as -T")
	fs.Bool("message", false, "same as -T")
	fs.BoolP("users", "u", false, "list users logged in")

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "my-line")
}

// buildOptions folds the parsed flags into who.Options.
func buildOptions(fs *pflag.FlagSet) who.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	return who.Options{
		All:      on("all"),
		Boot:     on("boot"),
		Dead:     on("dead"),
		Heading:  on("heading"),
		Login:    on("login"),
		Lookup:   on("lookup"),
		MyLine:   on("my-line"),
		Process:  on("process"),
		Count:    on("count"),
		RunLevel: on("runlevel"),
		Short:    on("short"),
		Time:     on("time"),
		Mesg:     on("mesg") || on("writable") || on("message"),
		Users:    on("users"),
	}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return who.New(buildOptions(fs), s).Run(args)
}
