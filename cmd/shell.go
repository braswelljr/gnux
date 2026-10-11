package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shell"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// NewShell builds a gnux shell whose commands are the registered tools, run
// in-process. It is the library entry point for running a portable script:
//
//	sh := cmd.NewShell(shell.Options{ErrExit: true}, core.Stdio())
//	err := sh.RunString("mkdirx -p dist && cpx -r assets dist/")
func NewShell(opts shell.Options, s core.Streams) *shell.Shell {
	return shell.New(opts, shellExec, s)
}

// shellExec is what a shell command runs through: a tool by either name, or
// gnux itself (gnux help rmx, gnux --list) as the whole command line.
func shellExec(name string, args []string, s core.Streams) (status int, found bool) {
	if name == common.AppName {
		return execute(NewRoot(s), args, s), true
	}
	return RunTool(name, args, s)
}

// ShellSpec describes the `gnux shell` command. It is not a tool - it mirrors
// no GNU program and has no x-name - so it lives outside Tools().
func ShellSpec() tool.Spec {
	return tool.Spec{
		Meta: docmeta.Meta{
			Name:    "shell",
			SeeAlso: []string{"envx", "truex", "falsex", "echox"},
			ExitStatus: []docmeta.Status{
				{Code: "N", Meaning: "the exit status of the last command run"},
				{Code: "1", Meaning: "a redirection could not be opened (that command is not run)"},
				{Code: "2", Meaning: "a syntax error, exit with a non-numeric argument, or a command-line mistake"},
				{Code: "127", Meaning: "an unknown command, or a SCRIPT that cannot be opened"},
			},
			Use:   "shell [OPTION]... [SCRIPT]",
			Short: "Run a portable shell whose commands are the gnux tools.",
			Long: "gnux shell is a small command interpreter whose commands are the gnux tools, " +
				"run inside the gnux process. Because nothing in it depends on the system's own " +
				"shell or utilities, the same script behaves identically on Linux, macOS and " +
				"Windows. With no SCRIPT it reads commands from standard input, prompting when " +
				"that is a terminal; with -c it runs the given string.\n\n" +
				"The language is a deliberately small subset of POSIX sh: pipelines (a | b), the " +
				"&&, || and ; connectors, redirection (<, >, >>, and 2> for standard error; " +
				"/dev/null discards on every platform), NAME=value variables with " +
				"$NAME, ${NAME}, $? and $PWD expansion, single and double quotes, globbing (*, ?, " +
				"[...]), ~ for the home directory and # comments. The builtins are cd, exit and " +
				"help. Tools answer to both names, so rm and rmx are the same command. There are " +
				"no loops, functions or conditionals beyond && and ||, and a variable's value is " +
				"never re-split, so paths with spaces need no quoting once stored. External " +
				"programs are not looked up; run one through the env tool (envx PROGRAM ARG...).\n\n" +
				"The exit status is that of the last command run, 2 for a syntax error, and 127 " +
				"for an unknown command. With -e the script stops at the first failing command.",
			Permissions: "The shell itself needs only read access to SCRIPT. Each command it " +
				"runs needs whatever its tool needs; redirection creates files with mode 0644 " +
				"(before the umask).",
			UseCases: []string{
				"Build, clean and packaging scripts that must run unchanged on Windows, macOS and Linux.",
				"A consistent interactive shell on machines without the GNU tools installed.",
				"Embedding small file-management scripts in a Go application without spawning processes.",
			},
			Examples: []docmeta.Example{
				{
					Description: "Start an interactive session:",
					Bash:        "gnux shell",
					PowerShell:  "gnux shell",
				},
				{
					Description: "Run a one-liner:",
					Bash:        "gnux shell -c 'mkdirx -p dist && cpx -r assets dist/ && lsx -l dist'",
					PowerShell:  "gnux shell -c 'mkdirx -p dist && cpx -r assets dist/ && lsx -l dist'",
				},
				{
					Description: "Run a script file, stopping at the first failure:",
					Bash:        "gnux shell -e build.gnux",
					PowerShell:  "gnux shell -e build.gnux",
				},
				{
					Description: "Pipe a script in:",
					Bash:        "echo 'catx *.log | wcx -l' | gnux shell",
					PowerShell:  "'catx *.log | wcx -l' | gnux shell",
				},
			},
		},
		Flags: func(fs *pflag.FlagSet) {
			fs.StringP("command", "c", "", "run the commands in `STRING` instead of reading a script")
			fs.BoolP("errexit", "e", false, "exit at the first command that fails")
		},
		Run:       runShell,
		UsageExit: 2,
	}
}

func runShell(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts := shell.Options{}
	opts.ErrExit, _ = fs.GetBool("errexit")

	var script io.Reader
	switch {
	case fs.Changed("command"):
		if len(args) > 0 {
			return core.Usagef("extra operand '%s'", args[0])
		}
		command, _ := fs.GetString("command")
		script = strings.NewReader(command)

	case len(args) > 1:
		return core.Usagef("extra operand '%s'", args[1])

	case len(args) == 1:
		f, err := os.Open(args[0])
		if err != nil {
			diag := core.NewDiag(shell.Name, s)
			diag.Errorf("%s: %s", args[0], core.ErrText(err))
			return &core.ExitError{Code: 127}
		}
		defer f.Close()
		script = f

	default:
		script = s.In
		opts.Interactive = core.IsTerminal(s.In)
	}

	return NewShell(opts, s).Run(script)
}
