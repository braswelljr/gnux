// Package pwdx wires the pwdx command (GNU pwd) to its engine.
package pwdx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/pwd"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the pwdx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "pwd",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for pwdx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    pwd.Name,
		SeeAlso: []string{"realpathx", "dirnamex", "gnux shell"},
		Use:     pwd.Name + " [OPTION]...",
		Short:   "Print the full name of the current working directory.",
		Long: "pwdx prints the absolute path of the current working directory. With -P (the " +
			"default) every symbolic link in the path is resolved; with -L it prints the value " +
			"of the PWD environment variable when that names the current directory, preserving " +
			"the symlinks the shell walked through. If both are given, the last one wins in " +
			"GNU pwd; pwdx prefers -L.",
		Permissions: "Needs search permission on every directory leading to the current one " +
			"to resolve it physically.",
		UseCases: []string{
			"Capturing the working directory in a script.",
			"Finding the real location behind a symlinked directory with -P.",
			"Keeping the symlinked path the user navigated through with -L, for messages or relative links.",
			"Detecting whether the current directory is reached through a symlink by comparing -L and -P.",
			"Deriving the project name or parent directory with basenamex and dirnamex.",
			"Checking where a cd or envx -C actually landed in a script.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the physical working directory:",
				Bash:        "pwdx",
				PowerShell:  "pwdx",
			},
			{
				Description: "Keep symlinks as the shell sees them:",
				Bash:        "pwdx -L",
				PowerShell:  "pwdx -L",
			},
			{
				Description: "Resolve every symlink explicitly:",
				Bash:        "cd link && pwdx -P",
				PowerShell:  "Set-Location link; pwdx -P",
			},
			{
				Description: "Save the directory in a variable:",
				Bash:        "here=$(pwdx); echox \"working in $here\"",
				PowerShell:  "$here = pwdx; echox \"working in $here\"",
			},
			{
				Description: "Print only the current directory's name:",
				Bash:        "basenamex \"$(pwdx)\"",
				PowerShell:  "basenamex (pwdx)",
			},
			{
				Description: "Print the parent directory:",
				Bash:        "dirnamex \"$(pwdx)\"",
				PowerShell:  "dirnamex (pwdx)",
			},
			{
				Description: "Tell whether the current directory is reached through a symlink:",
				Bash:        "[ \"$(pwdx -L)\" != \"$(pwdx -P)\" ] && echox 'inside a symlinked directory'",
				PowerShell:  "if ((pwdx -L) -ne (pwdx -P)) { echox 'inside a symlinked directory' }",
			},
			{
				Description: "Print the directory a command would run in:",
				Bash:        "envx -C build pwdx",
				PowerShell:  "envx -C build pwdx",
			},
			{
				Description: "Show where cd leads inside the gnux shell:",
				Bash:        "gnux shell -c 'cd docs; pwdx'",
				PowerShell:  "gnux shell -c 'cd docs; pwdx'",
			},
		},
	}
}

// Register installs the pwdx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("logical", "L", false, "use PWD from the environment, even if it contains symlinks")
	fs.BoolP("physical", "P", false, "resolve all symlinks (default)")
}

// buildOptions folds the parsed flags into pwd.Options.
func buildOptions(fs *pflag.FlagSet) pwd.Options {
	logical, _ := fs.GetBool("logical")
	return pwd.Options{Logical: logical}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return pwd.New(buildOptions(fs), s).Run(args)
}
