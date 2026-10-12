// Package nohupx wires the nohupx command (GNU nohup) to its engine.
package nohupx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nohup"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the nohupx tool. It has no options of its own, and option
// parsing stops at COMMAND so the command's options reach it untouched.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:           Meta(),
		GNU:            "nohup",
		Run:            run,
		NoInterspersed: true,
		UsageExit:      nohup.ExitCanceled,
	}
}

// Meta returns the documentation metadata for nohupx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    nohup.Name,
		SeeAlso: []string{"timeoutx", "nicex", "killx", "teex"},
		Use:     nohup.Name + " COMMAND [ARG]...",
		Short:   "Run a command immune to hangups, with output to a non-terminal.",
		Long: "nohupx runs COMMAND so that it keeps running after you log out or close the " +
			"terminal. On Unix it ignores the hangup signal, as GNU nohup does. Any of the " +
			"standard streams that is a terminal is replaced: standard input by an unusable " +
			"/dev/null, standard output by the file nohup.out, appended to and created " +
			"readable only by you, in the current directory or, if that fails, in $HOME, and " +
			"standard error by standard output; nohupx says which on standard error. Streams " +
			"that are already redirected are left alone. The exit status is COMMAND's, or 125 " +
			"when nohupx itself failed, 126 when COMMAND was found but could not be run and " +
			"127 when it was not found. GNU nohup does nothing useful on Windows; nohupx " +
			"starts COMMAND detached from the console and in a process group of its own, so " +
			"closing the console window or pressing Ctrl-C in it no longer ends COMMAND. " +
			"nohupx runs COMMAND as a child and waits for it, where GNU nohup becomes " +
			"COMMAND; to get your prompt back, start it in the background with your shell.",
		Permissions: "Needs execute permission on COMMAND, and write permission on the current " +
			"directory or $HOME when nohup.out has to be created.",
		UseCases: []string{
			"Keeping a long download, build or migration running after an SSH session drops.",
			"Starting a server from a terminal that will be closed.",
			"Capturing the output of a background job in a file automatically.",
			"Running a job on Windows that must survive the console window closing.",
			"Combining with timeoutx and nicex for a bounded, low-priority background job.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Keep a long rsync running after your SSH session ends:",
				Bash:        "nohupx rsync -a /data backup:/data &",
				PowerShell:  "Start-Job { nohupx robocopy C:\\data \\\\backup\\data /E }",
			},
			{
				Description: "Start a server in the background and log its output:",
				Bash:        "nohupx ./server > server.log 2>&1 &",
				PowerShell:  "Start-Process nohupx -ArgumentList '.\\server.exe' -RedirectStandardOutput server.log -NoNewWindow",
			},
			{
				Description: "Let output go to nohup.out and follow it:",
				Bash:        "nohupx ./train.py & tailx -f nohup.out",
				PowerShell:  "Start-Job { nohupx python train.py }; tailx -f nohup.out",
			},
			{
				Description: "Remember the process ID to stop it later:",
				Bash:        "nohupx ./worker & echox $! > worker.pid",
				PowerShell:  "(Start-Process nohupx -ArgumentList '.\\worker.exe' -PassThru).Id > worker.pid",
			},
			{
				Description: "Stop that job when it is no longer needed:",
				Bash:        "killx \"$(catx worker.pid)\"",
				PowerShell:  "killx (catx worker.pid)",
			},
			{
				Description: "Run a migration that must not be cut short by a closed laptop:",
				Bash:        "nohupx ./migrate.sh > migrate.log 2>&1 &",
				PowerShell:  "Start-Job { nohupx .\\migrate.ps1 > migrate.log }",
			},
			{
				Description: "Run a nightly build at low priority, immune to hangups:",
				Bash:        "nohupx nicex make release &",
				PowerShell:  "Start-Job { nohupx nicex make release }",
			},
			{
				Description: "Give a background job at most two hours:",
				Bash:        "nohupx timeoutx 2h ./crawl.sh &",
				PowerShell:  "Start-Job { nohupx timeoutx 2h .\\crawl.ps1 }",
			},
			{
				Description: "Keep standard error separate from the output:",
				Bash:        "nohupx ./job 2> job.err &",
				PowerShell:  "Start-Job { nohupx .\\job.exe 2> job.err }",
			},
			{
				Description: "Check how the job ended:",
				Bash:        "nohupx ./job; echox \"exit status $?\"",
				PowerShell:  "nohupx .\\job.exe; \"exit status $LASTEXITCODE\"",
			},
			{
				Description: "Run an external program immune to hangups from the portable gnux shell:",
				Bash:        "gnux shell -c 'nohupx envx ./server > server.log'",
				PowerShell:  "gnux shell -c 'nohupx envx ./server > server.log'",
			},
		},
		ExitStatus: []docmeta.Status{
			{Code: "125", Meaning: "nohupx itself failed"},
			{Code: "126", Meaning: "COMMAND was found but could not be run"},
			{Code: "127", Meaning: "COMMAND was not found"},
			{Code: "other", Meaning: "the exit status of COMMAND"},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return nohup.New(nohup.Options{}, s).Run(args)
}
