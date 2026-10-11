// Package rmx wires the rmx command (GNU rm) to its removal engine. Each flag
// lives in its own sub-package (rmx/force, rmx/interactive, …); this file
// registers them and folds their values into a single rm.Options for one
// engine run.
package rmx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/cmd/rmx/directory"
	"github.com/braswelljr/gnux/cmd/rmx/force"
	"github.com/braswelljr/gnux/cmd/rmx/interactive"
	"github.com/braswelljr/gnux/cmd/rmx/onefilesystem"
	"github.com/braswelljr/gnux/cmd/rmx/preserveroot"
	"github.com/braswelljr/gnux/cmd/rmx/recursive"
	"github.com/braswelljr/gnux/cmd/rmx/verbose"
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rm"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the rmx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "rm",
		Flags: Register,
		Run:   run,
		Topics: []tool.Topic{
			{Meta: force.Meta(), Register: force.Register},
			{Meta: interactive.Meta(), Register: interactive.Register},
			{Meta: recursive.Meta(), Register: recursive.Register},
			{Meta: directory.Meta(), Register: directory.Register},
			{Meta: verbose.Meta(), Register: verbose.Register},
			{Meta: onefilesystem.Meta(), Register: onefilesystem.Register},
			{Meta: preserveroot.Meta(), Register: preserveroot.Register},
			tool.VersionTopic(rm.Name),
		},
	}
}

// Meta returns the documentation metadata for rmx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    rm.Name,
		SeeAlso: []string{"rmdirx", "mvx", "cpx", "gnux shell"},
		Use:     rm.Name + " [OPTION]... [FILE]...",
		Short:   "Remove (unlink) files and directories.",
		Long: "rmx removes each FILE. By default it does not remove directories; use -r to " +
			"remove a directory and everything under it, or -d to remove an empty directory. " +
			"Prompting is controlled by -i (before every removal), -I (once before a bulk or " +
			"recursive removal) and --interactive; -f disables all prompts and ignores missing " +
			"files. rmx refuses to remove '.' and '..' and, by default, refuses to recurse on " +
			"the filesystem root. Independent arguments are removed concurrently whenever no " +
			"prompt or ordered output is involved.",
		Permissions: "Removing an entry needs write and search permission on the directory " +
			"that contains it, not on the entry itself. On a terminal, a write-protected file " +
			"triggers a confirmation prompt unless -f is given.",
		UseCases: []string{
			"Everyday file cleanup.",
			"Unattended deletion in CI or scripts with -f.",
			"Cautious, per-file review with -i, or one confirmation before a large delete with -I.",
			"Auditing a recursive delete with -rv.",
			"Clearing build output (build/, dist/, node_modules/) before a fresh build.",
			"Pruning directories that should already be empty with -d, without risking contents.",
			"Removing temporary files a script created, even if some were never written (-f).",
			"Deleting a tree without crossing into mounted volumes (--one-file-system).",
			"Removing files whose names start with '-' by ending options with --.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Remove a few files:",
				Bash:        "rmx notes.txt draft.md",
				PowerShell:  "rmx notes.txt draft.md",
			},
			{
				Description: "Recursively remove a directory, reporting each deletion:",
				Bash:        "rmx -rv build/",
				PowerShell:  "rmx -rv build",
			},
			{
				Description: "Unattended cleanup: no prompts, ignore missing paths:",
				Bash:        "rmx -rf /tmp/cache",
				PowerShell:  "rmx -rf $env:TEMP\\cache",
			},
			{
				Description: "Remove an empty directory only:",
				Bash:        "rmx -d generated/",
				PowerShell:  "rmx -d generated",
			},
			{
				Description: "Ask before each removal:",
				Bash:        "rmx -i *.bak",
				PowerShell:  "rmx -i (Get-ChildItem -Filter *.bak -Name)",
			},
			{
				Description: "Ask once before a recursive delete:",
				Bash:        "rmx -I -r old-project/",
				PowerShell:  "rmx -I -r old-project",
			},
			{
				Description: "Choose the prompt policy explicitly:",
				Bash:        "rmx --interactive=once *.log",
				PowerShell:  "rmx --interactive=once (Get-ChildItem -Filter *.log -Name)",
			},
			{
				Description: "Answer every prompt with yes from a pipeline:",
				Bash:        "yesx | rmx -i a.tmp b.tmp",
				PowerShell:  "yesx | rmx -i a.tmp b.tmp",
			},
			{
				Description: "Remove a file whose name starts with a dash:",
				Bash:        "rmx -- -draft.txt",
				PowerShell:  "rmx -- -draft.txt",
			},
			{
				Description: "Remove a tree but skip file systems mounted beneath it:",
				Bash:        "rmx -r --one-file-system scratch/",
				PowerShell:  "rmx -r --one-file-system scratch",
			},
			{
				Description: "Disable the root failsafe for a deliberate recursive delete:",
				Bash:        "rmx -rf --no-preserve-root /mnt/scratch",
				PowerShell:  "rmx -rf --no-preserve-root D:\\scratch",
			},
			{
				Description: "Clean up and rebuild only if the removal succeeded:",
				Bash:        "rmx -rf dist/ && mkdirx dist",
				PowerShell:  "rmx -rf dist; if ($LASTEXITCODE -eq 0) { mkdirx dist }",
			},
			{
				Description: "Detect a failed removal in a script:",
				Bash:        "rmx missing.txt || echox \"rmx exited with $?\"",
				PowerShell:  "rmx missing.txt; if ($LASTEXITCODE -ne 0) { echox \"rmx exited with $LASTEXITCODE\" }",
			},
			{
				Description: "Count how many entries a recursive delete removed:",
				Bash:        "rmx -rv cache/ | wcx -l",
				PowerShell:  "rmx -rv cache | wcx -l",
			},
			{
				Description: "Remove temporary files with the gnux shell:",
				Bash:        "gnux shell -c 'rmx -fv *.tmp && echox done'",
				PowerShell:  "gnux shell -c 'rmx -fv *.tmp && echox done'",
			},
		},
	}
}

// Register installs every rmx flag onto fs.
func Register(fs *pflag.FlagSet) {
	force.Register(fs)
	interactive.Register(fs)
	recursive.Register(fs)
	directory.Register(fs)
	verbose.Register(fs)
	onefilesystem.Register(fs)
	preserveroot.Register(fs)
}

// run builds the removal options from the parsed flags and runs the engine
// against args.
func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return rm.New(opts, s).Run(args)
}

// buildOptions folds every flag package's value into a single rm.Options. Order
// matters: force sets the baseline prompt mode, then interactive may raise it
// (see the interactive package for the precedence rules).
func buildOptions(fs *pflag.FlagSet) (rm.Options, error) {
	var o rm.Options

	force.Apply(fs, &o)
	if err := interactive.Apply(fs, &o); err != nil {
		return rm.Options{}, err
	}
	recursive.Apply(fs, &o)
	directory.Apply(fs, &o)
	verbose.Apply(fs, &o)
	onefilesystem.Apply(fs, &o)
	preserveroot.Apply(fs, &o)

	return o, nil
}
