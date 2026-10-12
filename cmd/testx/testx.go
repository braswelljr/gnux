// Package testx wires the testx command (GNU test) to its engine.
package testx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/test"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the testx tool. Its arguments are an expression whose
// operators look like options (-f, -n, -eq), so flag parsing is left to the
// engine.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:      Meta(),
		GNU:       "test",
		Run:       run,
		RawArgs:   true,
		UsageExit: test.ExitError,
	}
}

// Meta returns the documentation metadata for testx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    test.Name,
		SeeAlso: []string{"exprx", "truex", "falsex", "gnux shell"},
		Use:     test.Name + " EXPRESSION",
		Short:   "Check file types and compare values.",
		Long: "testx evaluates EXPRESSION and exits with status 0 when it is true and 1 when it " +
			"is false; it prints nothing. With no EXPRESSION it is false. Strings: -n STRING " +
			"(not empty), -z STRING (empty), STRING alone (not empty), S1 = S2 (also ==), " +
			"S1 != S2, and S1 < S2 or S1 > S2, comparing bytes (the C locale). Integers: N1 " +
			"-eq, -ne, -lt, -le, -gt or -ge N2, where an integer is an optional sign and " +
			"decimal digits of any length, with surrounding blanks allowed. Files: -e (exists), " +
			"-f (regular file), -d (directory), -h or -L (symbolic link, not followed), -b and " +
			"-c (block and character devices), -p (named pipe), -S (socket), -s (size above " +
			"zero), -r, -w and -x (readable, writable, executable by you), -u, -g and -k " +
			"(set-user-ID, set-group-ID and sticky bits), -O and -G (owned by your effective " +
			"user or group), -t FD (descriptor FD is a terminal), and F1 -nt, -ot or -ef F2 " +
			"(newer, older, or the same file). Expressions combine with ! (not), -a (and, " +
			"binding tighter), -o (or) and parentheses, which must be separate arguments " +
			"(quote them from the shell). As POSIX requires, one to four arguments are read " +
			"by their count first, so testx -n = -n compares two strings. A malformed " +
			"expression is diagnosed with GNU's wording ('a': unary operator expected, " +
			"invalid integer 'x', missing argument after '-f', extra argument 'c') and exits " +
			"with status 2. Differences from GNU test: the [ form is not provided (a gnux " +
			"tool's name is its GNU name with an x appended), a lone --help or --version " +
			"prints help or the version where GNU test would treat it as a non-empty string, " +
			"the -l STRING length operand is not implemented, and off Unix -O and -G are " +
			"always false while -r, -w and -x are read from the permission bits.",
		Permissions: "Reads file metadata only: -e, -f, -d and the other file tests need search " +
			"permission on the directories leading to the file; -r, -w and -x ask the " +
			"operating system whether you could read, write or execute it, without trying. " +
			"Nothing is ever modified.",
		UseCases: []string{
			"Checking that a file or directory exists before using it in a script.",
			"Branching on whether a variable is empty with -z or -n.",
			"Comparing version numbers or counts with -lt, -ge and the other integer tests.",
			"Rebuilding only when a source file is newer than its output with -nt.",
			"Checking that a script is writing to a terminal with -t 1 before using colors.",
			"Guarding a destructive step with && in the portable gnux shell.",
			"Verifying permissions (-r, -w, -x) before a deployment step runs.",
		},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "EXPRESSION is true"},
			{Code: "1", Meaning: "EXPRESSION is false, or missing"},
			{Code: "2", Meaning: "EXPRESSION is malformed (diagnosed on standard error)"},
		},
		Examples: []docmeta.Example{
			{
				Description: "Check that a file exists:",
				Bash:        "testx -e config.json && echox found",
				PowerShell:  "testx -e config.json; if ($LASTEXITCODE -eq 0) { 'found' }",
			},
			{
				Description: "Check for a directory:",
				Bash:        "testx -d build || mkdirx build",
				PowerShell:  "testx -d build; if ($LASTEXITCODE -ne 0) { mkdirx build }",
			},
			{
				Description: "Check that a variable is not empty:",
				Bash:        `testx -n "$NAME" && echox "hello, $NAME"`,
				PowerShell:  `testx -n "$env:NAME"; if ($LASTEXITCODE -eq 0) { "hello, $env:NAME" }`,
			},
			{
				Description: "Compare two strings:",
				Bash:        `testx "$BRANCH" = main && echox "on main"`,
				PowerShell:  `testx "$env:BRANCH" = main; if ($LASTEXITCODE -eq 0) { 'on main' }`,
			},
			{
				Description: "Compare two numbers:",
				Bash:        "testx 10 -gt 9 && echox bigger",
				PowerShell:  "testx 10 -gt 9; if ($LASTEXITCODE -eq 0) { 'bigger' }",
			},
			{
				Description: "Rebuild only when the source is newer than the output:",
				Bash:        "testx main.go -nt app && go build -o app .",
				PowerShell:  "testx main.go -nt app; if ($LASTEXITCODE -eq 0) { go build -o app . }",
			},
			{
				Description: "Combine conditions with -a and -o:",
				Bash:        "testx -f go.mod -a -d .git && echox 'a Go repository'",
				PowerShell:  "testx -f go.mod -a -d .git; if ($LASTEXITCODE -eq 0) { 'a Go repository' }",
			},
			{
				Description: "Negate and group (parentheses are separate, quoted arguments):",
				Bash:        "testx ! '(' -e a.lock -o -e b.lock ')' && echox unlocked",
				PowerShell:  "testx '!' '(' -e a.lock -o -e b.lock ')'; if ($LASTEXITCODE -eq 0) { 'unlocked' }",
			},
			{
				Description: "Check that a file is not empty:",
				Bash:        "testx -s report.csv || echox 'report is empty'",
				PowerShell:  "testx -s report.csv; if ($LASTEXITCODE -ne 0) { 'report is empty' }",
			},
			{
				Description: "Check whether standard output is a terminal:",
				Bash:        "testx -t 1 && echox interactive",
				PowerShell:  "testx -t 1; if ($LASTEXITCODE -eq 0) { 'interactive' }",
			},
			{
				Description: "See the status of a malformed expression (2):",
				Bash:        `testx 1 -eq one; echox "exit status $?"`,
				PowerShell:  `testx 1 -eq one; "exit status $LASTEXITCODE"`,
			},
			{
				Description: "Guard a step in the portable gnux shell:",
				Bash:        "gnux shell -c 'testx -d dist && rmx -r dist'",
				PowerShell:  "gnux shell -c 'testx -d dist && rmx -r dist'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return test.New(test.Options{}, s).Run(args)
}
