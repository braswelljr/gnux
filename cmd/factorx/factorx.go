// Package factorx wires the factorx command (GNU factor) to its engine.
package factorx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/factor"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the factorx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "factor",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for factorx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    factor.Name,
		SeeAlso: []string{"seqx", "exprx", "xargsx"},
		Use:     factor.Name + " [OPTION] [NUMBER]...",
		Short:   "Print the prime factors of each specified integer NUMBER.",
		Long: "factorx prints each NUMBER followed by a colon and its prime factors in ascending " +
			"order, a repeated factor as many times as it divides the number. With no NUMBER, it " +
			"reads numbers separated by spaces, tabs and newlines from standard input, and " +
			"answers each line as soon as it is read. 0 and 1 have no prime factors and print " +
			"just the colon. A NUMBER may have leading spaces and one '+', and must otherwise be " +
			"decimal digits; anything else is reported as 'is not a valid positive integer', the " +
			"remaining numbers are still factored, and the exit status is 1. -h (--exponents) " +
			"prints a repeated factor once, as p^e. As in GNU factor, a NUMBER has no size limit. " +
			"Numbers below 2^64 are factored with trial division, the Miller-Rabin test and " +
			"Pollard's rho, and take microseconds; larger ones use arbitrary-precision arithmetic, " +
			"which is fast when the number has at most one large prime factor and can take " +
			"minutes when it is the product of two primes of more than 15 digits each.",
		Permissions: "Needs no permissions. Reads only its operands or standard input.",
		UseCases: []string{
			"Checking whether a number is prime: a prime's only factor is itself.",
			"Finding the divisors of a number in a script or a puzzle.",
			"Simplifying a fraction or finding a common divisor by hand.",
			"Teaching or checking arithmetic exercises.",
			"Generating test data with known factorizations.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Factor a number:",
				Bash:        "factorx 360",
				PowerShell:  "factorx 360",
				Output:      "360: 2 2 2 3 3 5",
			},
			{
				Description: "Show repeated factors as powers:",
				Bash:        "factorx -h 360",
				PowerShell:  "factorx -h 360",
				Output:      "360: 2^3 3^2 5",
			},
			{
				Description: "Factor several numbers at once:",
				Bash:        "factorx 12 97 1001",
				PowerShell:  "factorx 12 97 1001",
				Output:      "12: 2 2 3\n97: 97\n1001: 7 11 13",
			},
			{
				Description: "Check whether a number is prime:",
				Bash:        "[ \"$(factorx 1000000007 | cutx -d' ' -f2-)\" = 1000000007 ] && echox prime",
				PowerShell:  "if ((factorx 1000000007) -eq '1000000007: 1000000007') { 'prime' }",
			},
			{
				Description: "Factor numbers read from standard input:",
				Bash:        "seqx 10 15 | factorx",
				PowerShell:  "seqx 10 15 | factorx",
			},
			{
				Description: "List the primes up to 50:",
				Bash:        "seqx 2 50 | factorx | grepx -E '^([0-9]+): \\1$' | cutx -d: -f1",
				PowerShell:  "seqx 2 50 | factorx | grepx -E '^([0-9]+): \\1$' | cutx -d: -f1",
			},
			{
				Description: "Factor the largest 64-bit unsigned number:",
				Bash:        "factorx 18446744073709551615",
				PowerShell:  "factorx 18446744073709551615",
				Output:      "18446744073709551615: 3 5 17 257 641 65537 6700417",
			},
			{
				Description: "Factor a number larger than 64 bits:",
				Bash:        "factorx 1234567890123456789012345678901234567890",
				PowerShell:  "factorx 1234567890123456789012345678901234567890",
			},
			{
				Description: "Count the prime factors of a number:",
				Bash:        "factorx 1024 | trx ' ' '\\n' | tailx -n +2 | wcx -l",
				PowerShell:  "factorx 1024 | trx ' ' '\\n' | tailx -n +2 | wcx -l",
			},
			{
				Description: "Factor numbers in the portable gnux shell:",
				Bash:        "gnux shell -c 'for n in 6 28 496; do factorx -h $n; done'",
				PowerShell:  "gnux shell -c 'for n in 6 28 496; do factorx -h $n; done'",
			},
		},
	}
}

// Register installs the factorx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("exponents", "h", false, "print repeated factors in form p^e unless e is 1")
}

// buildOptions folds the parsed flags into factor.Options.
func buildOptions(fs *pflag.FlagSet) factor.Options {
	exponents, _ := fs.GetBool("exponents")
	return factor.Options{Exponents: exponents}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return factor.New(buildOptions(fs), s).Run(args)
}
