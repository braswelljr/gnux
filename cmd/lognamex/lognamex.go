// Package lognamex wires the lognamex command (GNU logname) to its engine.
package lognamex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/logname"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the lognamex tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "logname",
		Run:  run,
	}
}

// Meta returns the documentation metadata for lognamex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    logname.Name,
		SeeAlso: []string{"whoamix", "usersx", "whox", "idx"},
		Use:     logname.Name + " [OPTION]...",
		Short:   "Print the user's login name.",
		Long: "lognamex prints the name the user logged in as, which stays the same after su or " +
			"sudo, where whoamix prints the user the process now runs as. It takes no operands. " +
			"When there is no login name, as in a cron job, a container or a service, it " +
			"reports 'no login name' and exits 1. GNU logname asks the C library's getlogin; " +
			"gnux has no cgo and does what glibc does instead: on Linux it reads the login user " +
			"the kernel records for the process (/proc/self/loginuid), and failing that, on " +
			"every Unix, the login record of the terminal on standard input. macOS and the " +
			"BSDs also keep the name in the kernel session, which only the C library can read, " +
			"so there a process with no terminal on standard input has no login name for " +
			"lognamex even when GNU logname would print one. On Windows it prints the user the " +
			"session was signed in as, without the domain.",
		Permissions: "Needs no permissions. Reads the process's login user and the login " +
			"records.",
		UseCases: []string{
			"Finding out who really logged in inside a sudo or su session.",
			"Addressing the human user in a script that runs as root.",
			"Detecting that a script runs without a login (cron, a service) by its exit status.",
			"Recording who started a deployment in its log.",
			"Choosing a per-user directory in an administration script.",
		},
		Examples: []docmeta.Example{
			{Description: "Print your login name:", Bash: "lognamex", PowerShell: "lognamex", Output: "alice"},
			{Description: "See the difference from whoamix under sudo:", Bash: "sudo sh -c 'lognamex; whoamix'", PowerShell: "lognamex; whoamix", Output: "alice\nroot"},
			{Description: "Record who ran a deployment:", Bash: "echox \"deployed by $(lognamex)\" >> deploy.log", PowerShell: "\"deployed by $(lognamex)\" >> deploy.log"},
			{Description: "Fall back to the effective user when there is no login:", Bash: "name=$(lognamex 2>/dev/null || whoamix)", PowerShell: "$name = lognamex 2>$null; if ($LASTEXITCODE -ne 0) { $name = whoamix }"},
			{Description: "Detect a run without a login:", Bash: "lognamex >/dev/null 2>&1 || echox 'no login session'", PowerShell: "lognamex *> $null; if ($LASTEXITCODE -ne 0) { 'no login session' }"},
			{Description: "Use the login name in a path:", Bash: "mkdirx -p \"/srv/scratch/$(lognamex)\"", PowerShell: "mkdirx -p \"scratch/$(lognamex)\""},
			{Description: "Save the login name in a variable:", Bash: "me=$(lognamex)", PowerShell: "$me = lognamex"},
			{Description: "Greet the user:", Bash: "printfx 'hello, %s\\n' \"$(lognamex)\"", PowerShell: "printfx 'hello, %s\\n' (lognamex)"},
			{Description: "List the other sessions of the same user:", Bash: "whox | grepx \"^$(lognamex) \"", PowerShell: "whox | grepx \"^$(lognamex) \""},
			{Description: "Print the login name in the portable gnux shell:", Bash: "gnux shell -c 'lognamex'", PowerShell: "gnux shell -c 'lognamex'"},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return logname.New(logname.Options{}, s).Run(args)
}
