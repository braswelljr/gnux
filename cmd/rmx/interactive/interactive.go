// Package interactive implements rm's prompting flags: -i (prompt before every
// removal), -I (prompt once before a bulk or recursive removal) and
// --interactive[=WHEN].
package interactive

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rm"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// Meta returns the documentation metadata for the interactive flags.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:  "interactive",
		Use:   "interactive [FILE]...",
		Short: "Prompt before removals: -i (every file), -I (once), --interactive[=WHEN].",
		Long: "rmx offers three levels of confirmation. -i prompts before every single removal. " +
			"-I prompts just once before removing more than three files or before any recursive " +
			"removal - less intrusive than -i while still guarding against the worst mistakes. " +
			"--interactive[=WHEN] selects the level explicitly, where WHEN is never, once, or " +
			"always (bare --interactive means always). When several are combined, rmx picks the " +
			"most cautious: -i > --interactive=WHEN > -I > -f.",
		Permissions: "On a terminal, a write-protected file normally prompts " +
			"'remove write-protected ...?'. -i replaces that with an ordinary prompt for every " +
			"file regardless of mode, while --interactive=never (like -f) suppresses the " +
			"write-protected prompt entirely. The prompts never change what you are permitted to " +
			"delete - only whether you are asked.",
		UseCases: []string{
			"Reviewing each deletion when clearing a directory of mixed important and junk files (-i).",
			"A single safety confirmation before a large or recursive delete (-I).",
			"Scripting a specific prompt policy independent of -f (--interactive=WHEN).",
			"Overriding a -f baked into an alias, since -i is more cautious and wins.",
			"Feeding prepared answers from a pipeline (yesx) in tests.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Ask before removing a single file:",
				Bash:        "rmx -i notes.txt",
				PowerShell:  "rmx -i notes.txt",
			},
			{
				Description: "Ask once, then recursively remove a project directory:",
				Bash:        "rmx -I -r project/",
				PowerShell:  "rmx -I -r project",
			},
			{
				Description: "Delete without any prompt, equivalent to -f for prompting:",
				Bash:        "rmx --interactive=never old.tmp",
				PowerShell:  "rmx --interactive=never old.tmp",
			},
			{
				Description: "Ask once before removing more than three files:",
				Bash:        "rmx --interactive=once a.log b.log c.log d.log",
				PowerShell:  "rmx --interactive=once a.log b.log c.log d.log",
			},
			{
				Description: "-i wins over -f, so every removal is still confirmed:",
				Bash:        "rmx -fi draft.md",
				PowerShell:  "rmx -fi draft.md",
			},
			{
				Description: "Answer yes to every prompt from a pipeline:",
				Bash:        "yesx | rmx -iv a.tmp b.tmp",
				PowerShell:  "yesx | rmx -iv a.tmp b.tmp",
			},
		},
	}
}

// Register installs -i, -I and --interactive on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("prompt-each", "i", false, "prompt before every removal")
	fs.BoolP("prompt-once", "I", false, "prompt once before removing more than three files, or when removing recursively")
	fs.String("interactive", "", "prompt according to WHEN: never, once, or always")
	fs.Lookup("interactive").NoOptDefVal = "always"
}

// Apply resolves the interactive mode into o. Conflicts favor the most
// cautious option: -i (always) > --interactive=WHEN > -I (once) > whatever -f
// already set. Returns an error for an unrecognized --interactive value.
func Apply(fs *pflag.FlagSet, o *rm.Options) error {
	mode := o.Interactive

	if fs.Changed("interactive") {
		when, _ := fs.GetString("interactive")
		switch strings.ToLower(when) {
		case "never", "no", "none":
			mode = rm.PromptNever
		case "once":
			mode = rm.PromptOnce
		case "always", "yes", "":
			mode = rm.PromptAlways
		default:
			return core.Usagef("invalid argument %q for '--interactive'\nValid arguments are:\n  - 'never'\n  - 'once'\n  - 'always'", when)
		}
	}

	if b, _ := fs.GetBool("prompt-once"); b {
		mode = rm.PromptOnce
	}
	if b, _ := fs.GetBool("prompt-each"); b {
		mode = rm.PromptAlways
	}

	o.Interactive = mode
	return nil
}
