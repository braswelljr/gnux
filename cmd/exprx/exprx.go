// Package exprx wires the exprx command (GNU expr) to its engine.
package exprx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/expr"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the exprx tool. Its arguments are an expression whose
// operators and operands may look like options (exprx -5 + 1), so flag
// parsing is left to the engine.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:      Meta(),
		GNU:       "expr",
		Run:       run,
		RawArgs:   true,
		UsageExit: expr.ExitInvalid,
	}
}

// Meta returns the documentation metadata for exprx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    expr.Name,
		SeeAlso: []string{"testx", "seqx", "printfx", "gnux shell"},
		Use:     expr.Name + " EXPRESSION",
		Short:   "Evaluate an expression and print its value.",
		Long: "exprx evaluates EXPRESSION, given as separate arguments, and prints its value. " +
			"From the loosest binding: A | B is A if it is neither empty nor 0, otherwise B " +
			"(or 0); A & B is A if neither is empty nor 0, otherwise 0; A = B (also ==), " +
			"A != B, and the orderings compare as integers when both sides are integers and " +
			"as strings otherwise, giving 1 or 0; + and - add and subtract; * / and % " +
			"multiply, divide (truncating toward zero) and take the remainder; STRING : " +
			"REGEX matches. Integers are an optional minus sign and decimal digits, of any " +
			"size. A match is anchored at the start of STRING: when REGEX has a \\( \\) group " +
			"the value is the text the first group matched (or empty), otherwise the number " +
			"of bytes matched (or 0). REGEX is a POSIX basic regular expression with GNU's " +
			"\\+, \\? and \\| extensions and character classes such as [[:digit:]]. The " +
			"keywords match STRING REGEX, substr STRING POS LENGTH, index STRING CHARS and " +
			"length STRING do the same as their names, and + TOKEN takes TOKEN as a string " +
			"even if it is a keyword or an operator. Parentheses group. Operators must be " +
			"separate arguments, and shell characters such as * | & ( ) must be quoted. A " +
			"leading -- is skipped. The exit status is 0 when the value is neither empty " +
			"nor 0, 1 when it is, 2 for an invalid expression (syntax error, non-integer " +
			"argument, division by zero, a bad regular expression) and 3 for other errors. " +
			"Differences from GNU expr: strings are measured and sliced in bytes and " +
			"compared byte by byte (the C locale), and back references (\\1) and the " +
			"[= =] and [. .] bracket forms are not supported in a REGEX.",
		Permissions: "Needs no filesystem permissions; it only writes to standard output.",
		UseCases: []string{
			"Doing integer arithmetic in a script, with no size limit.",
			"Incrementing a counter in the portable gnux shell, where there is no $((...)).",
			"Extracting part of a string with a regular expression group.",
			"Getting the length of a string or a substring of it.",
			"Checking that a value is numeric before using it.",
			"Comparing two values numerically or as strings and branching on the status.",
		},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "the value is neither empty nor 0"},
			{Code: "1", Meaning: "the value is empty or 0"},
			{Code: "2", Meaning: "EXPRESSION is invalid, or missing"},
			{Code: "3", Meaning: "another error occurred, such as a failed write"},
		},
		Examples: []docmeta.Example{
			{
				Description: "Add two numbers:",
				Bash:        "exprx 2 + 3",
				PowerShell:  "exprx 2 + 3",
			},
			{
				Description: "Multiply (quote the * from the shell):",
				Bash:        "exprx 6 '*' 7",
				PowerShell:  "exprx 6 '*' 7",
			},
			{
				Description: "Integer division and remainder:",
				Bash:        "exprx 17 / 5 && exprx 17 % 5",
				PowerShell:  "exprx 17 / 5; exprx 17 % 5",
			},
			{
				Description: "Increment a counter in a script:",
				Bash:        `n=$(exprx "$n" + 1)`,
				PowerShell:  `$n = exprx $n + 1`,
			},
			{
				Description: "Work with numbers of any size:",
				Bash:        "exprx 99999999999999999999 '*' 3",
				PowerShell:  "exprx 99999999999999999999 '*' 3",
			},
			{
				Description: "Compare two values (prints 1 or 0):",
				Bash:        "exprx 10 '>' 9",
				PowerShell:  "exprx 10 '>' 9",
			},
			{
				Description: "Get the length of a string:",
				Bash:        "exprx length 'hello world'",
				PowerShell:  "exprx length 'hello world'",
			},
			{
				Description: "Take a substring (position 1, three bytes):",
				Bash:        "exprx substr release-2024 9 4",
				PowerShell:  "exprx substr release-2024 9 4",
			},
			{
				Description: "Strip an extension with a regular expression group:",
				Bash:        `exprx report.txt : '\(.*\)\.txt'`,
				PowerShell:  `exprx report.txt : '\(.*\)\.txt'`,
			},
			{
				Description: "Check that a value is all digits:",
				Bash:        `exprx "$VALUE" : '[0-9]*$' > /dev/null && echox numeric`,
				PowerShell:  `exprx "$env:VALUE" : '[0-9]*$' | Out-Null; if ($LASTEXITCODE -eq 0) { 'numeric' }`,
			},
			{
				Description: "Find the first vowel:",
				Bash:        "exprx index gnux aeiou",
				PowerShell:  "exprx index gnux aeiou",
			},
			{
				Description: "Use a keyword as a plain string:",
				Bash:        "exprx + length",
				PowerShell:  "exprx + length",
			},
			{
				Description: "See the status of an invalid expression (2):",
				Bash:        `exprx 1 / 0; echox "exit status $?"`,
				PowerShell:  `exprx 1 / 0; "exit status $LASTEXITCODE"`,
			},
			{
				Description: "Count in the portable gnux shell:",
				Bash:        `gnux shell -c 'N=4; exprx $N + 1'`,
				PowerShell:  `gnux shell -c 'N=4; exprx $N + 1'`,
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return expr.New(expr.Options{}, s).Run(args)
}
