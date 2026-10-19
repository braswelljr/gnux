// Package envx wires the envx command (GNU env) to its engine.
package envx

import (
	"errors"
	"io"
	"os"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/env"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the envx tool. Option parsing stops at the first operand so
// the options of the command being run are left for that command, and every
// failure of envx itself exits with 125 to stay distinguishable from the
// command's own status.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:           Meta(),
		GNU:            "env",
		Flags:          Register,
		Run:            run,
		NoInterspersed: true,
		UsageExit:      env.ExitCanceled,
	}
}

// Meta returns the documentation metadata for envx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    env.Name,
		SeeAlso: []string{"gnux shell", "echox", "sleepx"},
		ExitStatus: []docmeta.Status{
			{Code: "0-124, 128+", Meaning: "the exit status of COMMAND (on Unix, 128+N when it is killed by signal N); with no COMMAND, 0"},
			{Code: "125", Meaning: "envx itself failed: a bad option, a variable it could not unset, a directory it could not enter (-C), a write error"},
			{Code: "126", Meaning: "COMMAND was found but could not be run"},
			{Code: "127", Meaning: "COMMAND was not found"},
		},
		Use:   env.Name + " [OPTION]... [-] [NAME=VALUE]... [COMMAND [ARG]...]",
		Short: "Run a program in a modified environment, or print the environment.",
		Long: "envx sets each NAME to VALUE in the environment and runs COMMAND with the " +
			"remaining arguments; without a COMMAND it prints the resulting environment, one " +
			"NAME=VALUE per line. -i (or a lone '-') starts from an empty environment, -u " +
			"removes a variable, -C changes the working directory for the command and -S " +
			"splits a single string into arguments, which is what makes options usable on a " +
			"'#!' line. Printed variables keep their inherited order: assigning an existing " +
			"name replaces it in place, and new names are appended. COMMAND is looked up on " +
			"the PATH of the modified environment. Where GNU env replaces itself with the " +
			"command, envx runs it as a child process connected to the same standard streams " +
			"and exits with the command's status, so the behavior is identical on platforms " +
			"that cannot replace a running process. The exit status is 125 if envx itself " +
			"fails, 126 if COMMAND is found but cannot be run, 127 if COMMAND cannot be " +
			"found, and otherwise the exit status of COMMAND (128 plus the signal number when " +
			"a signal killed it).",
		Permissions: "Printing the environment needs no permissions. Running a command needs " +
			"execute permission on the program and search permission on the directories " +
			"leading to it, plus search permission on the -C directory when one is given.",
		UseCases: []string{
			"Setting variables for a single command in a way that works in every shell.",
			"Running a command with a clean, reproducible environment using -i.",
			"Locating an interpreter through PATH on a '#!' line, with arguments via -S.",
			"Dumping the environment for debugging, or NUL-separated with -0 for safe parsing.",
			"Removing a variable that changes a program's behavior (DEBUG, GOFLAGS, LANG) for one run with -u.",
			"Running a command in another directory with -C without changing the caller's directory.",
			"Cross-compiling or configuring builds with per-command variables such as GOOS and GOARCH.",
			"Checking how a command fails: 127 when it is not found, 125 when envx itself fails.",
			"Tracing exactly which environment and arguments a command receives with -v.",
			"Running the gnux multi-call binary as a chosen tool by setting its zeroth argument with -a.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the current environment:",
				Bash:        "envx",
				PowerShell:  "envx",
			},
			{
				Description: "Show only the first few variables:",
				Bash:        "envx | headx -n 3",
				PowerShell:  "envx | headx -n 3",
			},
			{
				Description: "Run a command with extra variables:",
				Bash:        "envx GOOS=windows GOARCH=amd64 go build ./...",
				PowerShell:  "envx GOOS=windows GOARCH=amd64 go build ./...",
			},
			{
				Description: "Check that an assignment is appended to the printed environment:",
				Bash:        "envx GREETING=hello | tailx -n 1",
				PowerShell:  "envx GREETING=hello | tailx -n 1",
			},
			{
				Description: "Run a command with nothing but the variables given:",
				Bash:        "envx -i PATH=/usr/bin HOME=/tmp make test",
				PowerShell:  "envx -i \"PATH=$env:PATH\" \"TEMP=$env:TEMP\" make test",
			},
			{
				Description: "Print an empty environment plus a single variable:",
				Bash:        "envx -i LANG=C.UTF-8",
				PowerShell:  "envx -i LANG=C.UTF-8",
			},
			{
				Description: "Drop variables for one command (-u may be repeated):",
				Bash:        "envx -u HOME -u USER gnux shell -c 'echox \"[$HOME] [$USER]\"'",
				PowerShell:  "envx -u HOME -u USER gnux shell -c 'echox \"[$HOME] [$USER]\"'",
			},
			{
				Description: "Drop a variable and run in another directory:",
				Bash:        "envx -u DEBUG -C build ./configure",
				PowerShell:  "envx -u DEBUG -C build cmake ..",
			},
			{
				Description: "Run a tool in another directory without leaving this one:",
				Bash:        "envx -C build pwdx",
				PowerShell:  "envx -C build pwdx",
			},
			{
				Description: "Print the environment NUL-separated, with the NULs made visible:",
				Bash:        "envx -0 -i A=1 B=2 | catx -v",
				PowerShell:  "envx -0 -i A=1 B=2 | catx -v",
			},
			{
				Description: "Split one string into a command and its arguments, as on a '#!' line:",
				Bash:        "envx -S 'gnux shell -e' build.sh",
				PowerShell:  "envx -S 'gnux shell -e' build.sh",
			},
			{
				Description: "Trace each step envx takes before running the command:",
				Bash:        "envx -v -C build pwdx",
				PowerShell:  "envx -v -C build pwdx",
			},
			{
				Description: "Run the gnux binary as rmx by setting its zeroth argument:",
				Bash:        "envx -a rmx gnux -v old.log",
				PowerShell:  "envx -a rmx gnux -v old.log",
			},
			{
				Description: "Tell a missing command (status 127) from a failing one:",
				Bash:        "envx no-such-command; echox \"status: $?\"",
				PowerShell:  "envx no-such-command; echox \"status: $LASTEXITCODE\"",
			},
			{
				Description: "Set a variable for a command run from the gnux shell:",
				Bash:        "gnux shell -c 'envx GREETING=hi envx | tailx -n 1'",
				PowerShell:  "gnux shell -c 'envx GREETING=hi envx | tailx -n 1'",
			},
		},
	}
}

// Register installs the envx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("ignore-environment", "i", false, "start with an empty environment")
	fs.StringArrayP("unset", "u", nil, "remove variable `NAME` from the environment")
	fs.BoolP("null", "0", false, "end each output line with NUL, not newline")
	fs.StringP("chdir", "C", "", "change working directory to `DIR`")
	fs.StringP("split-string", "S", "", "process and split `S` into separate arguments;\n"+
		"used to pass multiple arguments on shebang lines")
	fs.StringP("argv0", "a", "", "pass `ARG` as the zeroth argument of COMMAND")
	fs.BoolP("debug", "v", false, "print verbose information for each processing step")
}

// buildOptions folds the parsed flags into env.Options. -S is not part of it:
// it rewrites the command line itself and is handled by expandSplit.
func buildOptions(fs *pflag.FlagSet) env.Options {
	var o env.Options
	o.IgnoreEnvironment, _ = fs.GetBool("ignore-environment")
	// pflag hands back an empty slice when -u was never given; keep the zero
	// value so untouched options compare equal to env.Options{}.
	if unset, _ := fs.GetStringArray("unset"); len(unset) > 0 {
		o.Unset = unset
	}
	o.Null, _ = fs.GetBool("null")
	o.Chdir, _ = fs.GetString("chdir")
	o.Argv0, _ = fs.GetString("argv0")
	o.Debug, _ = fs.GetBool("debug")
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, args, err := expandSplit(fs, args, os.LookupEnv)
	if err != nil {
		var usage *core.UsageError
		if errors.As(err, &usage) {
			return err
		}
		// A malformed -S string is envx failing rather than a usage error:
		// GNU prints the bare diagnostic, without the --help hint.
		core.NewDiag(env.Name, s).Errorf("%v", err)
		return &core.ExitError{Code: env.ExitCanceled}
	}
	return env.New(opts, s).Run(args)
}

// expandSplit returns the options and operands in effect once every
// -S/--split-string has been expanded. The words a string splits into take
// its place ahead of the operands and are parsed as a command line of their
// own, so they may carry further options (another -S included), assignments
// and the command; lookup resolves the ${NAME} references in the string.
func expandSplit(
	fs *pflag.FlagSet, args []string, lookup func(string) (string, bool),
) (env.Options, []string, error) {
	opts := buildOptions(fs)
	for fs.Changed("split-string") {
		s, _ := fs.GetString("split-string")
		words, err := env.SplitString(s, lookup)
		if err != nil {
			return env.Options{}, nil, err
		}

		sub := pflag.NewFlagSet(env.Name, pflag.ContinueOnError)
		sub.SetOutput(io.Discard)
		sub.SetInterspersed(false)
		Register(sub)
		if err := tool.ParseFlagSet(sub, append(words, args...), true); err != nil {
			return env.Options{}, nil, core.Usagef("%v", err)
		}
		merge(&opts, buildOptions(sub))
		fs, args = sub, sub.Args()
	}
	return opts, args, nil
}

// merge adds the options found inside a split string to those given before it.
func merge(o *env.Options, more env.Options) {
	o.IgnoreEnvironment = o.IgnoreEnvironment || more.IgnoreEnvironment
	o.Unset = append(o.Unset, more.Unset...)
	o.Null = o.Null || more.Null
	o.Debug = o.Debug || more.Debug
	if more.Chdir != "" {
		o.Chdir = more.Chdir
	}
	if more.Argv0 != "" {
		o.Argv0 = more.Argv0
	}
}
