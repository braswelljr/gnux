// Package usersx wires the usersx command (GNU users) to its engine.
package usersx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/users"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the usersx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "users",
		Run:  run,
	}
}

// Meta returns the documentation metadata for usersx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    users.Name,
		SeeAlso: []string{"whox", "whoamix", "lognamex"},
		Use:     users.Name + " [OPTION]... [FILE]",
		Short:   "Print the user names of users currently logged in to the current host.",
		Long: "usersx prints the login name of each session on this machine, sorted and " +
			"separated by spaces on one line; a user logged in more than once appears once per " +
			"session. With nobody logged in it prints nothing. It reads the same records as " +
			"whox: the utmp file on Linux and the BSDs (sessions whose process has ended are " +
			"left out) and utmpx on macOS. FILE reads another file in that format instead, such " +
			"as /var/log/wtmp, without leaving any session out. On Windows the sessions are the " +
			"signed-in console and Remote Desktop sessions, and FILE is not supported; in the " +
			"browser playground there are no records.",
		Permissions: "Needs read permission on the login records (normally readable by " +
			"everyone) or on FILE. Writes nothing.",
		UseCases: []string{
			"Seeing who else is logged in before rebooting a shared machine.",
			"Counting the sessions on a server.",
			"Checking whether a particular user is logged in.",
			"Recording who was logged in when a job ran.",
			"Listing the users recorded in an old wtmp file.",
		},
		Examples: []docmeta.Example{
			{Description: "List the users logged in:", Bash: "usersx", PowerShell: "usersx", Output: "alice alice bob"},
			{Description: "Count the sessions:", Bash: "usersx | wcx -w", PowerShell: "usersx | wcx -w"},
			{Description: "List each user once:", Bash: "usersx | trx ' ' '\\n' | uniqx", PowerShell: "usersx | trx ' ' '\\n' | uniqx"},
			{Description: "Check whether a user is logged in:", Bash: "usersx | grepx -qw alice && echox 'alice is on'", PowerShell: "usersx | grepx -qw alice; if ($LASTEXITCODE -eq 0) { 'alice is on' }"},
			{Description: "Count the sessions of each user:", Bash: "usersx | trx ' ' '\\n' | uniqx -c", PowerShell: "usersx | trx ' ' '\\n' | uniqx -c"},
			{Description: "Read the users from a wtmp file:", Bash: "usersx /var/log/wtmp", PowerShell: "usersx /var/log/wtmp"},
			{Description: "See the same users with their terminals and times:", Bash: "whox", PowerShell: "whox"},
			{Description: "Warn before a reboot when anyone else is on:", Bash: "[ \"$(usersx | trx ' ' '\\n' | grepx -vx \"$(whoamix)\")\" ] && echox 'others are logged in'", PowerShell: "if (usersx | trx ' ' '\\n' | grepx -vx (whoamix)) { 'others are logged in' }"},
			{Description: "Log who was on when a job ran:", Bash: "echox \"$(datex): $(usersx)\" >> jobs.log", PowerShell: "\"$(datex): $(usersx)\" >> jobs.log"},
			{Description: "List the users in the portable gnux shell:", Bash: "gnux shell -c 'usersx'", PowerShell: "gnux shell -c 'usersx'"},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return users.New(users.Options{}, s).Run(args)
}
