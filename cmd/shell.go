package cmd

import (
	"io"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/shell"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/sandbox"
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
				{Code: "1", Meaning: "a redirection could not be opened (that command is not run), or with -u a variable was not set"},
				{Code: "2", Meaning: "a syntax error, exit with a non-numeric argument, or a command-line mistake"},
				{Code: "125", Meaning: "--sandbox was given but the sandbox could not be entered; nothing was run"},
				{Code: "127", Meaning: "an unknown command, or a SCRIPT that cannot be opened"},
			},
			Use:   "shell [OPTION]... [SCRIPT [ARG]...]",
			Short: "Run a portable shell whose commands are the gnux tools.",
			Long: "gnux shell is a small command interpreter whose commands are the gnux tools, " +
				"run inside the gnux process. Because nothing in it depends on the system's own " +
				"shell or utilities, the same script behaves identically on Linux, macOS and " +
				"Windows. With no SCRIPT it reads commands from standard input, prompting when " +
				"that is a terminal; with -c it runs the given string. The ARGs after SCRIPT are " +
				"its arguments ($1, $2, ... and \"$@\"); after -c STRING, the first one is $0 and " +
				"the rest are the arguments. Options are read only before SCRIPT, so everything " +
				"after it reaches the script.\n\n" +
				"The language is a deliberately small subset of POSIX sh: pipelines (a | b), the " +
				"&&, || and ; connectors, ! to invert a status, redirection (<, >, >>, and 2> for " +
				"standard error; /dev/null discards on every platform), NAME=value variables with " +
				"$NAME, ${NAME}, ${NAME:-default}, $? and $PWD expansion, the arguments $1 to $9, " +
				"${10}, $#, $0, $@ and $*, command substitution $(command), single and double " +
				"quotes, globbing (*, ?, [...]), ~ for the home directory and # comments. Control " +
				"flow is if/then/elif/else/fi, for NAME in WORD...; do ...; done (over the " +
				"arguments when 'in' is left out), while ...; do ...; done and case WORD in " +
				"PATTERN | PATTERN) ... ;; esac, each over as many lines as it needs. The builtins " +
				"are cd, exit, help, set (-e, -u, +e, +u, -- ARG...), shift, break, continue and " +
				"[ (the test tool, written [ EXPRESSION ]). Tools answer to both names, so rm and " +
				"rmx are the same command. External programs are not looked up; run one through " +
				"the env tool (envx PROGRAM ARG...).\n\n" +
				"It is not a full sh. There are no functions, no arithmetic (use exprx), no " +
				"until, no here-documents, no backquotes, no ( ) or { } grouping and no " +
				"background jobs. A quoted string does not span lines, a backslash does not " +
				"join lines, and a line that ends in |, && or || is an error. A pipeline can " +
				"hold one if, for, while or case, which runs as a subshell there.\n\n" +
				"Two things differ from sh on purpose. A variable's value is never re-split or " +
				"re-globbed, so a path with spaces needs no quoting once stored; for the same " +
				"reason an unquoted $(command) becomes one word per line of output, never one " +
				"per space, and a quoted one a single word. And NAME=value before a command " +
				"sets the variable for the rest of the script, not just for that command. A " +
				"$(command) runs as a subshell: what it sets, and a cd it makes, end with it.\n\n" +
				"The exit status is that of the last command run, 2 for a syntax error, and 127 " +
				"for an unknown command. With -e the script stops at the first failing command, " +
				"except one whose status is tested: by &&, || or !, or as the condition of an if " +
				"or a while. With -u, expanding a variable that is not set is an error that " +
				"stops the script with status 1; ${NAME:-default} reads one that may be missing. " +
				"An environment variable that is set but empty counts as unset.\n\n" +
				"--sandbox DIR confines the shell, before it reads or runs anything, with the " +
				"operating system's own sandbox: its commands, and any program they start through " +
				"envx, may then read and write each --sandbox DIR, only read each --sandbox-read " +
				"DIR and the system's own directories, and reach nothing else, the home directory " +
				"included. The kernel enforces it, so neither a symbolic link nor .. gets around " +
				"it, and it cannot be lifted. The sandbox is Landlock on Linux (5.13 or later, in a " +
				"gnux built without cgo, as releases are), Seatbelt on macOS, an AppContainer on " +
				"Windows and Capsicum on FreeBSD, where a command can run gnux's own tools only " +
				"(capability mode starts no other program); where none can be used the shell " +
				"refuses with status 125 and runs nothing. A SCRIPT must lie in an allowed directory, as it is read after the " +
				"sandbox is entered.",
			Permissions: "The shell itself needs only read access to SCRIPT. Each command it " +
				"runs needs whatever its tool needs; redirection creates files with mode 0644 " +
				"(before the umask).",
			UseCases: []string{
				"Build, clean and packaging scripts that must run unchanged on Windows, macOS and Linux.",
				"Running a script you did not write, or one an AI agent wrote, where it can only touch one directory.",
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
					Description: "Run a script with arguments, refusing unset variables:",
					Bash:        "gnux shell -eu release.gnux v1.2.3 dist",
					PowerShell:  "gnux shell -eu release.gnux v1.2.3 dist",
				},
				{
					Description: "Loop over files and branch on each:",
					Bash:        "gnux shell -c 'for f in *.txt; do if grepx -q TODO \"$f\"; then echox \"$f\"; fi; done'",
					PowerShell:  "gnux shell -c 'for f in *.txt; do if grepx -q TODO $f; then echox $f; fi; done'",
				},
				{
					Description: "Use a command's output:",
					Bash:        "gnux shell -c 'mkdirx -p \"build/$(unamex -s)\"'",
					PowerShell:  "gnux shell -c 'mkdirx -p build/$(unamex -s)'",
				},
				{
					Description: "Run a downloaded build script so that it can change nothing outside the project:",
					Bash:        "gnux shell --sandbox . -e ./build.gnux",
					PowerShell:  "gnux shell --sandbox . -e .\\build.gnux",
				},
				{
					Description: "Let a cleanup script read the shared config but write only in dist:",
					Bash:        "gnux shell --sandbox dist --sandbox-read config -c 'rmx -rf dist/* && cpx config/app.json dist/'",
					PowerShell:  "gnux shell --sandbox dist --sandbox-read config -c 'rmx -rf dist/* && cpx config/app.json dist/'",
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
			fs.BoolP("nounset", "u", false, "treat expanding an unset variable as an error")
			fs.StringArray("sandbox", nil, "confine the shell, its commands and the programs they start to `DIR`, which they may read and write, and the system's own directories, which they may only read; may be repeated (Linux and macOS)")
			fs.StringArray("sandbox-read", nil, "with --sandbox, also let them read `DIR`; may be repeated")
		},
		Run: runShell,
		// Options stop at SCRIPT: what follows it belongs to the script.
		NoInterspersed: true,
		UsageExit:      2,
	}
}

func runShell(fs *pflag.FlagSet, args []string, s core.Streams) error {
	if err := enterSandbox(fs, s); err != nil {
		return err
	}
	opts := shell.Options{}
	opts.ErrExit, _ = fs.GetBool("errexit")
	opts.NoUnset, _ = fs.GetBool("nounset")

	// name is $0 and params are $1, $2, ...
	name, params := shell.Name, []string(nil)
	var script io.Reader
	switch {
	case fs.Changed("command"):
		command, _ := fs.GetString("command")
		script = strings.NewReader(command)
		// As in sh, the first operand after -c STRING names the script.
		if len(args) > 0 {
			name, params = args[0], args[1:]
		}

	case len(args) > 0:
		f, err := fsys.Open(args[0])
		if err != nil {
			diag := core.NewDiag(shell.Name, s)
			diag.Errorf("%s: %s", args[0], core.ErrText(err))
			return &core.ExitError{Code: 127}
		}
		defer f.Close()
		script = f
		name, params = args[0], args[1:]

	default:
		script = s.In
		opts.Interactive = core.IsTerminal(s.In)
	}

	sh := NewShell(opts, s)
	sh.SetArgs(name, params)
	return sh.Run(script)
}

// enterSandbox confines the process as --sandbox and --sandbox-read ask,
// before anything is read or run. Refused, it runs nothing: status 125.
func enterSandbox(fs *pflag.FlagSet, s core.Streams) error {
	write, _ := fs.GetStringArray("sandbox")
	read, _ := fs.GetStringArray("sandbox-read")
	if len(write) == 0 {
		if len(read) > 0 {
			return core.Usagef("--sandbox-read needs --sandbox")
		}
		return nil
	}
	if err := sandbox.Enter(sandbox.Policy{Write: write, Read: read}); err != nil {
		core.NewDiag(shell.Name, s).Errorf("%v", err)
		return &core.ExitError{Code: sandboxRefused}
	}
	return nil
}

// sandboxRefused is the status when the sandbox cannot be entered and
// nothing has run, as env uses 125 for its own failure.
const sandboxRefused = 125
