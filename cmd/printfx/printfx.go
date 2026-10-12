// Package printfx wires the printfx command (GNU printf) to its engine.
package printfx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/printf"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the printfx tool. Its arguments are a format and the values
// it formats, which may look like options (printfx '%d\n' -5), so flag
// parsing is left to the engine.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "printf",
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for printfx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    printf.Name,
		SeeAlso: []string{"echox", "seqx", "gnux shell"},
		Use:     printf.Name + " FORMAT [ARGUMENT]...",
		Short:   "Format and print data.",
		Long: "printfx writes the ARGUMENTs to standard output under the control of FORMAT, " +
			"like C's printf. FORMAT is printed as written except for backslash escapes " +
			"(\\\", \\\\, \\a, \\b, \\c, \\e, \\f, \\n, \\r, \\t, \\v, \\NNN octal, \\xHH, \\uHHHH, " +
			"\\UHHHHHHHH) and conversion specifications: %d %i %o %u %x %X for integers, " +
			"%f %F %e %E %g %G %a %A for floating point, %c for a character, %s for a string, " +
			"%b for a string with its backslash escapes interpreted (octal as \\0NNN), %q for " +
			"a string quoted for reuse as shell input, and %% for a percent sign. A " +
			"conversion may carry the flags - + space # 0, a width and a precision, either " +
			"of which may be * to take it from the next ARGUMENT. FORMAT is reused as often " +
			"as needed to consume every ARGUMENT; a missing ARGUMENT reads as an empty string " +
			"or zero. A numeric ARGUMENT may be decimal, 0x hexadecimal, 0 octal (integers), " +
			"or a quote followed by a character, meaning that character's code. An ARGUMENT " +
			"that is not a valid number is diagnosed, printed as far as it converted, and " +
			"makes the exit status 1 without stopping the output. \\c ends all output. Unlike " +
			"the shell's builtin printf, printfx behaves the same in bash, PowerShell and the " +
			"gnux shell. Differences from GNU printf: floating-point values are held in double " +
			"precision rather than long double, and \\c ends output with the status the run " +
			"already had rather than always 0.",
		Permissions: "Needs no filesystem permissions; it only writes to standard output.",
		UseCases: []string{
			"Printing text without a trailing newline, or with exact control over every byte.",
			"Aligning columns in a report with field widths.",
			"Formatting numbers: fixed decimals, zero padding, hexadecimal or octal.",
			"Converting between number bases, or printing a character's code.",
			"Repeating a line template over a list of arguments.",
			"Quoting a value with %q so it can be pasted back into a shell safely.",
			"Writing files with exact content (escapes, no trailing newline) from scripts.",
			"Getting identical formatted output in bash, PowerShell and the gnux shell.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print a line, adding the newline explicitly:",
				Bash:        `printfx 'Hello, %s\n' world`,
				PowerShell:  `printfx 'Hello, %s\n' world`,
			},
			{
				Description: "Format a number with two decimal places:",
				Bash:        `printfx '%.2f\n' 3.14159`,
				PowerShell:  `printfx '%.2f\n' 3.14159`,
			},
			{
				Description: "Pad a number with zeros to five digits:",
				Bash:        `printfx '%05d\n' 42`,
				PowerShell:  `printfx '%05d\n' 42`,
			},
			{
				Description: "Align columns with field widths:",
				Bash:        `printfx '%-10s|%5d\n' apples 3 pears 12`,
				PowerShell:  `printfx '%-10s|%5d\n' apples 3 pears 12`,
			},
			{
				Description: "Reuse the format for every argument:",
				Bash:        `printfx '- %s\n' one two three`,
				PowerShell:  `printfx '- %s\n' one two three`,
			},
			{
				Description: "Convert a number to hexadecimal and octal:",
				Bash:        `printfx '%x %o\n' 255 255`,
				PowerShell:  `printfx '%x %o\n' 255 255`,
			},
			{
				Description: "Convert hexadecimal to decimal:",
				Bash:        `printfx '%d\n' 0xff`,
				PowerShell:  `printfx '%d\n' 0xff`,
			},
			{
				Description: "Print the character code of a letter:",
				Bash:        `printfx '%d\n' "'A"`,
				PowerShell:  `printfx '%d\n' "'A"`,
			},
			{
				Description: "Take the field width from an argument:",
				Bash:        `printfx '[%*s]\n' 8 right`,
				PowerShell:  `printfx '[%*s]\n' 8 right`,
			},
			{
				Description: "Interpret escapes in an argument with %b:",
				Bash:        `printfx '%b\n' 'tab:\there'`,
				PowerShell:  `printfx '%b\n' 'tab:\there'`,
			},
			{
				Description: "Quote a value for reuse as shell input:",
				Bash:        `printfx '%q\n' "it's a file.txt"`,
				PowerShell:  `printfx '%q\n' "it's a file.txt"`,
			},
			{
				Description: "Print a Unicode character by code point:",
				Bash:        `printfx 'é\n'`,
				PowerShell:  `printfx 'é\n'`,
			},
			{
				Description: "Write a file with no trailing newline:",
				Bash:        "printfx 'v1.2.3' > VERSION",
				PowerShell:  "printfx 'v1.2.3' | Out-File -NoNewline VERSION",
			},
			{
				Description: "Notice a value that is not a number (exit status 1):",
				Bash:        `printfx '%d\n' abc || echox "exit status $?"`,
				PowerShell:  `printfx '%d\n' abc; if ($LASTEXITCODE -ne 0) { "exit status $LASTEXITCODE" }`,
			},
			{
				Description: "Format a table in the portable gnux shell:",
				Bash:        `gnux shell -c 'printfx "%-6s %3s\n" name qty pen 4 ink 12'`,
				PowerShell:  `gnux shell -c 'printfx "%-6s %3s\n" name qty pen 4 ink 12'`,
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return printf.New(printf.Options{}, s).Run(args)
}
