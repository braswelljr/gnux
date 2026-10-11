// Package docmeta defines the documentation metadata that every tool (and,
// where a tool documents them individually, every flag) exposes. It has no
// dependencies, so it is safe to import from packages that link into the gnux
// binary - the doc generator (cmd/docs) turns this metadata into reference pages.
package docmeta

import (
	"fmt"
	"strings"
)

// Meta describes a single documentation topic: a tool (e.g. "rmx") or one of
// its flags (e.g. "force"). The flag details themselves (short/long name,
// default, usage) are read from the registered flag set; Meta supplies the
// prose, permissions notes, use cases and worked examples around them.
type Meta struct {
	Name        string    // topic/file base name, e.g. "rmx" or "force"
	Use         string    // synopsis line, e.g. "rmx [OPTION]... [FILE]..."
	Short       string    // one-line summary
	Long        string    // full description
	Permissions string    // how the topic interacts with filesystem permissions
	UseCases    []string  // when to reach for it
	Examples    []Example // worked examples

	// ExitStatus lists the statuses the topic can end with. Empty means the
	// GNU default for a tool: 0 on success, 1 on failure (see DefaultExitStatus).
	ExitStatus []Status

	// SeeAlso names related tools or topics ("cpx", "gnux shell").
	SeeAlso []string
}

// Status documents one exit status. Code is a string so that ranges and
// derived values ("128+N", "the command's") can be written down too.
type Status struct {
	Code    string
	Meaning string
}

// DefaultExitStatus is the exit status table of a tool that follows the plain
// GNU convention, with usageExit the status of a command-line mistake.
func DefaultExitStatus(usageExit int) []Status {
	if usageExit == 0 || usageExit == 1 {
		return []Status{
			{"0", "success: every operand was processed"},
			{"1", "failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake"},
		}
	}
	return []Status{
		{"0", "success: every operand was processed"},
		{"1", "failure: an operand could not be processed (diagnosed on standard error)"},
		{fmt.Sprint(usageExit), "a command-line mistake"},
	}
}

// Example is one worked example: a description plus the equivalent command in
// each shell. PowerShell may be empty when it adds nothing over Bash.
type Example struct {
	Description string
	Bash        string
	PowerShell  string
}

// ExampleBlock renders examples as a plain, indented shell block: the form
// terminal --help output and the man/yaml/reST references use, which support
// only a single example section.
func ExampleBlock(examples []Example) string {
	var b strings.Builder
	for i, e := range examples {
		if i > 0 {
			b.WriteString("\n")
		}
		if e.Description != "" {
			fmt.Fprintf(&b, "  # %s\n", strings.TrimRight(e.Description, ":"))
		}
		if e.Bash != "" {
			fmt.Fprintf(&b, "  %s\n", e.Bash)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
