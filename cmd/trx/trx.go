// Package trx wires the trx command (GNU tr) to its engine.
package trx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tr"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the trx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "tr",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for trx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    tr.Name,
		SeeAlso: []string{"catx", "wcx", "echox"},
		Use:     tr.Name + " [OPTION]... STRING1 [STRING2]",
		Short:   "Translate, squeeze, and/or delete characters from standard input.",
		Long: "trx copies standard input to standard output, translating each byte of SET1 into " +
			"the byte at the same position in SET2. SET2 is padded with its last byte to the " +
			"length of SET1, unless -t truncates SET1 to the length of SET2 instead; when a byte " +
			"appears more than once in SET1, its last mapping wins. -d deletes the bytes of SET1, " +
			"and -s squeezes each run of a repeated byte from the last SET given into one; with " +
			"-d and -s together, SET1 is deleted and SET2 squeezed. -c (or -C) uses the " +
			"complement of SET1: every byte not in it, in ascending order. A SET is a string of " +
			"bytes with these constructs: \\NNN (an octal byte), \\\\, \\a, \\b, \\f, \\n, \\r, \\t, " +
			"\\v; CHAR1-CHAR2 for the bytes from CHAR1 to CHAR2 in ascending order; [CHAR*] in " +
			"SET2 for as many copies of CHAR as make SET2 as long as SET1, and [CHAR*REPEAT] for " +
			"REPEAT copies (octal if it starts with 0); [:CLASS:] for alnum, alpha, blank, cntrl, " +
			"digit, graph, lower, print, punct, space, upper or xdigit; and [=CHAR=] for the " +
			"bytes equivalent to CHAR. When translating, only [:lower:] and [:upper:] may appear " +
			"in SET2, and only opposite a case class in SET1, so that letters map to their other " +
			"case. trx reads only standard input. Like GNU tr, it works on bytes, not " +
			"characters: a multibyte UTF-8 character is several separate bytes to it, and the " +
			"classes are those of the C locale (ASCII).",
		Permissions: "Needs no file permissions: trx reads standard input and writes standard " +
			"output. Redirect a file into it to read one; the shell, not trx, opens it.",
		UseCases: []string{
			"Changing the case of text with '[:lower:]' '[:upper:]'.",
			"Converting Windows line endings by deleting carriage returns with -d '\\r'.",
			"Squeezing runs of spaces or blank lines into one with -s.",
			"Splitting text into one word per line with -cs '[:alnum:]' '\\n'.",
			"Removing digits, punctuation or control characters from input with -d.",
			"Turning a PATH-style list into lines by translating ':' to '\\n'.",
			"Replacing NUL separators with newlines to read find -print0 output.",
			"Applying a simple substitution cipher such as ROT13.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Convert text to upper case:",
				Bash:        "echox 'hello world' | trx '[:lower:]' '[:upper:]'",
				PowerShell:  "'hello world' | trx '[:lower:]' '[:upper:]'",
			},
			{
				Description: "Convert text to lower case with ranges:",
				Bash:        "echox 'Hello World' | trx A-Z a-z",
				PowerShell:  "'Hello World' | trx A-Z a-z",
			},
			{
				Description: "Strip Windows carriage returns from a file:",
				Bash:        "trx -d '\\r' < dos.txt > unix.txt",
				PowerShell:  "Get-Content -Raw dos.txt | trx -d '\\r' | Set-Content -NoNewline unix.txt",
			},
			{
				Description: "Squeeze runs of spaces into one:",
				Bash:        "echox 'a    b     c' | trx -s ' '",
				PowerShell:  "'a    b     c' | trx -s ' '",
			},
			{
				Description: "Delete every digit:",
				Bash:        "echox 'r2d2 c3po' | trx -d '[:digit:]'",
				PowerShell:  "'r2d2 c3po' | trx -d '[:digit:]'",
			},
			{
				Description: "Put each word on a line of its own:",
				Bash:        "catx essay.txt | trx -cs '[:alnum:]' '\\n'",
				PowerShell:  "catx essay.txt | trx -cs '[:alnum:]' '\\n'",
			},
			{
				Description: "List the directories of PATH, one per line:",
				Bash:        "echox \"$PATH\" | trx : '\\n'",
				PowerShell:  "$env:PATH | trx ';' '\\n'",
			},
			{
				Description: "Keep only the printable characters and newlines:",
				Bash:        "trx -cd '[:print:]\\n' < binary.log",
				PowerShell:  "catx binary.log | trx -cd '[:print:]\\n'",
			},
			{
				Description: "Map a set onto a shorter one without padding it (-t):",
				Bash:        "echox abcde | trx -t a-e xy",
				PowerShell:  "'abcde' | trx -t a-e xy",
			},
			{
				Description: "Replace every vowel with a star, using a repeat:",
				Bash:        "echox 'programming' | trx aeiou '[**]'",
				PowerShell:  "'programming' | trx aeiou '[**]'",
			},
			{
				Description: "Encode text with ROT13:",
				Bash:        "echox 'Hello' | trx A-Za-z N-ZA-Mn-za-m",
				PowerShell:  "'Hello' | trx A-Za-z N-ZA-Mn-za-m",
			},
			{
				Description: "Delete digits and squeeze the spaces left behind:",
				Bash:        "echox 'a 1 2 b' | trx -ds '[:digit:]' ' '",
				PowerShell:  "'a 1 2 b' | trx -ds '[:digit:]' ' '",
			},
			{
				Description: "Normalize text in the portable gnux shell:",
				Bash:        "gnux shell -c \"catx names.txt | trx '[:upper:]' '[:lower:]' > lower.txt\"",
				PowerShell:  "gnux shell -c \"catx names.txt | trx '[:upper:]' '[:lower:]' > lower.txt\"",
			},
		},
	}
}

// Register installs the trx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("complement", "c", false, "use the complement of SET1")
	fs.BoolP("complement-upper", "C", false, "equivalent to -c")
	fs.BoolP("delete", "d", false, "delete characters in SET1, do not translate")
	fs.BoolP("squeeze-repeats", "s", false,
		"replace each sequence of a repeated character that is listed in the last specified SET, with a single occurrence of that character")
	fs.BoolP("truncate-set1", "t", false, "first truncate SET1 to length of SET2")
}

// buildOptions folds the parsed flags into tr.Options.
func buildOptions(fs *pflag.FlagSet) tr.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	return tr.Options{
		Complement: on("complement") || on("complement-upper"),
		Delete:     on("delete"),
		Squeeze:    on("squeeze-repeats"),
		Truncate:   on("truncate-set1"),
	}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return tr.New(buildOptions(fs), s).Run(args)
}
