// Package timeoutx wires the timeoutx command (GNU timeout) to its engine.
package timeoutx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/timeout"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the timeoutx tool. Options end at DURATION, so that the
// command's own options are passed to it untouched.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:           Meta(),
		GNU:            "timeout",
		Flags:          Register,
		Run:            run,
		NoInterspersed: true,
		UsageExit:      timeout.ExitCanceled,
	}
}

// Meta returns the documentation metadata for timeoutx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    timeout.Name,
		SeeAlso: []string{"killx", "sleepx", "nicex", "nohupx"},
		Use:     timeout.Name + " [OPTION] DURATION COMMAND [ARG]...",
		Short:   "Run a command with a time limit.",
		Long: "timeoutx runs COMMAND and, if it is still running after DURATION, sends it the " +
			"TERM signal, or the signal -s names. DURATION is a number, possibly decimal or " +
			"hexadecimal, with an optional unit: s (seconds, the default), m (minutes), h " +
			"(hours) or d (days); 0 means no limit. -k DURATION also sends KILL if COMMAND " +
			"is still running that long after the first signal. By default the signal goes " +
			"to COMMAND's whole process group, so the processes it started stop too; " +
			"--foreground signals COMMAND alone and leaves it in the terminal's process group, " +
			"so it can read from the terminal. A signal sent to timeoutx is passed on to " +
			"COMMAND, and -v reports each signal sent. The exit status is 124 when COMMAND " +
			"timed out (unless --preserve-status, which keeps COMMAND's own status), 125 when " +
			"timeoutx itself failed, 126 when COMMAND was found but could not be run, 127 " +
			"when it was not found, and otherwise COMMAND's status (128+N when signal N ended " +
			"it, 137 after KILL). Options must come before DURATION. On Windows, which has no " +
			"signals, the limit still works: TERM and KILL both end COMMAND, so timeoutx is a " +
			"portable way to bound a step in a script on every platform.",
		Permissions: "Needs execute permission on COMMAND; signaling it needs no more, since " +
			"timeoutx started it.",
		UseCases: []string{
			"Stopping a CI step or test suite that hangs instead of letting it run all night.",
			"Bounding a network call such as curl or ssh that may never answer.",
			"Giving a service a grace period to shut down, then killing it with -k.",
			"Polling for a condition for at most a minute in a script.",
			"Limiting a job the same way on Linux, macOS and Windows.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Give a test suite 10 minutes before stopping it:",
				Bash:        "timeoutx 10m go test ./...",
				PowerShell:  "timeoutx 10m go test ./...",
			},
			{
				Description: "Stop waiting on a slow server after 5 seconds:",
				Bash:        "timeoutx 5 curl -s https://example.com/health",
				PowerShell:  "timeoutx 5 curl.exe -s https://example.com/health",
			},
			{
				Description: "Tell a timeout apart from a failure:",
				Bash:        "timeoutx 30s ./migrate.sh; [ $? -eq 124 ] && echox 'migration timed out'",
				PowerShell:  "timeoutx 30s .\\migrate.ps1; if ($LASTEXITCODE -eq 124) { 'migration timed out' }",
			},
			{
				Description: "Send INT, like Ctrl-C, instead of TERM:",
				Bash:        "timeoutx -s INT 1h ./crawler",
				PowerShell:  "timeoutx -s INT 1h .\\crawler.exe",
			},
			{
				Description: "Kill a server that ignores TERM 10 seconds after asking it to stop:",
				Bash:        "timeoutx -k 10s 1h ./server",
				PowerShell:  "timeoutx -k 10s 1h .\\server.exe",
			},
			{
				Description: "Report the signals as they are sent:",
				Bash:        "timeoutx -v 2s sleepx 60",
				PowerShell:  "timeoutx -v 2s sleepx 60",
			},
			{
				Description: "Keep the command's own exit status after a timeout:",
				Bash:        "timeoutx --preserve-status 1m ./bench.sh",
				PowerShell:  "timeoutx --preserve-status 1m .\\bench.ps1",
			},
			{
				Description: "Let an interactive command keep reading from the terminal:",
				Bash:        "timeoutx --foreground 5m ssh build-host",
				PowerShell:  "timeoutx --foreground 5m ssh build-host",
			},
			{
				Description: "Wait at most a minute for a file to appear:",
				Bash:        "timeoutx 1m sh -c 'until [ -e ready.flag ]; do sleepx 1; done'",
				PowerShell:  "timeoutx 1m pwsh -c 'while (-not (Test-Path ready.flag)) { Start-Sleep 1 }'",
			},
			{
				Description: "Use a fractional limit:",
				Bash:        "timeoutx 0.5 ./probe",
				PowerShell:  "timeoutx 0.5 .\\probe.exe",
			},
			{
				Description: "Bound a step in the portable gnux shell (external programs run through envx):",
				Bash:        "gnux shell -c 'timeoutx 30s envx make test || echox \"tests failed or timed out\"'",
				PowerShell:  "gnux shell -c 'timeoutx 30s envx make test || echox \"tests failed or timed out\"'",
			},
		},
		ExitStatus: []docmeta.Status{
			{Code: "124", Meaning: "COMMAND timed out (and --preserve-status was not given)"},
			{Code: "125", Meaning: "timeoutx itself failed"},
			{Code: "126", Meaning: "COMMAND was found but could not be run"},
			{Code: "127", Meaning: "COMMAND was not found"},
			{Code: "137", Meaning: "COMMAND (or timeoutx) was sent the KILL signal"},
			{Code: "other", Meaning: "the exit status of COMMAND"},
		},
	}
}

// Register installs the timeoutx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.Bool("foreground", false, "when not running timeout directly from a shell prompt, allow COMMAND to read from the TTY and get TTY signals; in this mode, children of COMMAND will not be timed out")
	fs.StringP("kill-after", "k", "", "also send a KILL signal if COMMAND is still running this long after the initial signal was sent (`DURATION`)")
	fs.Bool("preserve-status", false, "exit with the same status as COMMAND, even when the command times out")
	fs.StringP("signal", "s", "", "specify the `SIGNAL` to be sent on timeout; SIGNAL may be a name like 'HUP' or a number")
	fs.BoolP("verbose", "v", false, "diagnose to stderr any signal sent upon timeout")
}

// buildOptions folds the parsed flags into timeout.Options.
func buildOptions(fs *pflag.FlagSet) (timeout.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := timeout.Options{
		PreserveStatus: on("preserve-status"),
		Foreground:     on("foreground"),
		Verbose:        on("verbose"),
	}
	o.Signal, _ = fs.GetString("signal")
	if fs.Changed("kill-after") {
		text, _ := fs.GetString("kill-after")
		d, err := timeout.ParseDuration(text)
		if err != nil {
			return timeout.Options{}, err
		}
		o.KillAfter = d
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return timeout.New(opts, s).Run(args)
}
