// Package groupsx wires the groupsx command (GNU groups) to its engine.
package groupsx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/groups"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the groupsx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "groups",
		Run:  run,
	}
}

// Meta returns the documentation metadata for groupsx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    groups.Name,
		SeeAlso: []string{"idx", "whoamix", "chgrpx"},
		Use:     groups.Name + " [OPTION]... [USERNAME]...",
		Short:   "Print the groups a user is in.",
		Long: "groupsx prints the names of the groups the current process is in: its real group, " +
			"its effective group when that differs, then its supplementary groups, each once, as " +
			"idx -Gn does. For each USERNAME it prints 'USERNAME : ' and the groups the account " +
			"database lists for that user instead, which can differ from a running process's " +
			"groups until the user logs in again. An unknown user is reported as 'no such user' " +
			"and the remaining names are still printed; a group with no name is reported and " +
			"printed as its number. On Windows the groups are the account's local groups, which " +
			"is what Go's account lookup reports.",
		Permissions: "Needs no permissions. Reads the process's credentials and the account " +
			"database.",
		UseCases: []string{
			"Checking that you are in the docker, wheel or sudo group after adding yourself.",
			"Seeing which groups another user belongs to.",
			"Debugging a 'Permission denied' on a group-writable file.",
			"Recording the CI user's groups in a build log.",
			"Testing group membership in a script with grepx -w.",
		},
		Examples: []docmeta.Example{
			{Description: "Print your groups:", Bash: "groupsx", PowerShell: "groupsx", Output: "staff admin docker"},
			{Description: "Print another user's groups:", Bash: "groupsx root", PowerShell: "groupsx Administrator", Output: "root : root wheel"},
			{Description: "Print several users' groups:", Bash: "groupsx alice bob", PowerShell: "groupsx alice bob"},
			{Description: "Check membership of one group:", Bash: "groupsx | grepx -qw docker && echox 'in docker'", PowerShell: "groupsx | grepx -qw docker; if ($LASTEXITCODE -eq 0) { 'in docker' }"},
			{Description: "Print one group per line:", Bash: "groupsx | trx ' ' '\\n'", PowerShell: "groupsx | trx ' ' '\\n'"},
			{Description: "Count your groups:", Bash: "groupsx | wcx -w", PowerShell: "groupsx | wcx -w"},
			{Description: "Print the numeric group IDs instead:", Bash: "idx -G", PowerShell: "idx -G"},
			{Description: "Compare the account database with this process:", Bash: "groupsx \"$(whoamix)\"; groupsx", PowerShell: "groupsx (whoamix); groupsx"},
			{Description: "Log the groups of the build user:", Bash: "echox \"groups: $(groupsx)\" >> build.log", PowerShell: "\"groups: $(groupsx)\" >> build.log"},
			{Description: "Check a group in the portable gnux shell:", Bash: "gnux shell -c 'if groupsx | grepx -qw wheel; then echox admin; fi'", PowerShell: "gnux shell -c 'if groupsx | grepx -qw wheel; then echox admin; fi'"},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return groups.New(groups.Options{}, s).Run(args)
}
