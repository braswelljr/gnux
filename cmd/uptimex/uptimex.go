// Package uptimex wires the uptimex command (GNU uptime) to its engine.
package uptimex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uptime"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the uptimex tool. It has no options of its own.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "uptime",
		Run:  run,
	}
}

// Meta returns the documentation metadata for uptimex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    uptime.Name,
		SeeAlso: []string{"whox", "datex", "nprocx", "unamex"},
		Use:     uptime.Name + " [OPTION]... [FILE]",
		Short:   "Tell how long the system has been running.",
		Long: "uptimex prints, on one line, the current time, how long the system has been " +
			"up, how many users are logged in, and the load averages: the average number of " +
			"processes running or waiting to run over the last 1, 5 and 15 minutes (on " +
			"Linux, processes waiting on disk count too). A load average above the number of " +
			"CPUs (nprocx) means work is queuing. With FILE, the users and the boot time are " +
			"read from that utmp file instead of the current records. The boot time comes " +
			"from /proc on Linux, kern.boottime on macOS and FreeBSD, and the tick count on " +
			"Windows, where the users are the signed-in sessions; Windows keeps no load " +
			"average, so that part is left out there, as GNU uptime leaves it out wherever " +
			"the system cannot report it. If the boot time cannot be found the time up is " +
			"shown as '????' and the exit status is 1.",
		Permissions: "Needs no permissions.",
		UseCases: []string{
			"Checking whether a server rebooted unexpectedly.",
			"Seeing at a glance whether a machine is overloaded.",
			"Recording uptime and load in monitoring logs.",
			"Comparing the load with the CPU count before starting a heavy job.",
			"Getting the same summary on Linux, macOS and Windows.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Check how long the machine has been up and how loaded it is:",
				Bash:        "uptimex",
				PowerShell:  "uptimex",
				Output:      " 14:03:22  up 3 days  3:19,  2 users,  load average: 0.42, 0.37, 0.31",
			},
			{
				Description: "A server feels slow: compare the load with the number of CPUs:",
				Bash:        "uptimex && nprocx",
				PowerShell:  "uptimex; nprocx",
				Output:      " 10:12:40  up 41 days  2:07,  1 user,  load average: 15.83, 14.90, 12.02\n8",
			},
			{
				Description: "Did the server reboot overnight? Check the time up:",
				Bash:        "uptimex",
				PowerShell:  "uptimex",
				Output:      " 07:58:01  up  2:11,  0 users,  load average: 0.08, 0.05, 0.01",
			},
			{
				Description: "Log the load every 5 minutes from cron:",
				Bash:        "*/5 * * * * uptimex >> /var/log/load.log",
				PowerShell:  "uptimex >> load.log",
			},
			{
				Description: "Extract just the 1-minute load average:",
				Bash:        "uptimex | trx -s ' ' | cutx -d, -f4 | cutx -d: -f2",
				PowerShell:  "((uptimex) -split 'load average: ')[1].Split(',')[0]",
				Output:      " 0.42",
			},
			{
				Description: "Show the boot time itself rather than the time since:",
				Bash:        "whox -b",
				PowerShell:  "whox -b",
				Output:      "         system boot  Oct  9 10:44",
			},
			{
				Description: "Read the users and boot time from a saved utmp file:",
				Bash:        "uptimex /var/run/utmp",
				PowerShell:  "uptimex",
			},
			{
				Description: "Wait until the load drops below 1 before starting a backup:",
				Bash:        "until uptimex | grep -q 'load average: 0\\.'; do sleepx 60; done; ./backup.sh",
				PowerShell:  "while (-not ((uptimex) -match 'load average: 0\\.')) { sleepx 60 }; .\\backup.ps1",
			},
			{
				Description: "Record the uptime in a status report:",
				Bash:        "echox \"$(hostnamex): $(uptimex)\" >> status.txt",
				PowerShell:  "\"$(hostnamex): $(uptimex)\" >> status.txt",
			},
			{
				Description: "Check several machines' uptime over SSH:",
				Bash:        "for h in web1 web2 db1; do ssh \"$h\" uptimex; done",
				PowerShell:  "foreach ($h in 'web1','web2','db1') { ssh $h uptimex }",
			},
			{
				Description: "Show the uptime in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'uptimex && whox -q'",
				PowerShell:  "gnux shell -c 'uptimex && whox -q'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return uptime.New(uptime.Options{}, s).Run(args)
}
