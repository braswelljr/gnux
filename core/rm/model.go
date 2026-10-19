package rm

import (
	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic and prompt.
const Name = "rmx"

// Mode controls when rmx prompts before removing a path.
type Mode int

const (
	// PromptDefault prompts only for write-protected files, and only when
	// standard input is an interactive terminal (mirrors rm with no -f/-i/-I).
	PromptDefault Mode = iota
	// PromptNever never prompts (rm -f / --interactive=never).
	PromptNever
	// PromptOnce prompts a single time before a bulk or recursive removal
	// (rm -I / --interactive=once).
	PromptOnce
	// PromptAlways prompts before every removal (rm -i / --interactive=always).
	PromptAlways
)

// Options mirrors the GNU rm flag surface. The zero value is plain `rm`: no
// recursion, default prompting, and the '/' failsafe on.
type Options struct {
	Force         bool // -f: ignore nonexistent files, never prompt, suppress errors
	Recursive     bool // -r/-R: remove directories and their contents recursively
	Dir           bool // -d: remove empty directories
	Verbose       bool // -v: explain what is being done
	Interactive   Mode // -i/-I/--interactive
	OneFileSystem bool // --one-file-system: don't cross device boundaries
	// NoPreserveRoot (--no-preserve-root) allows recursing on '/'. It is the
	// negative so the zero value keeps the failsafe on.
	NoPreserveRoot bool
}

// Rm is the removal engine. Construct it with New.
type Rm struct {
	Options

	// PromptFn, when non-nil, is consulted instead of reading from standard
	// input. It is the injection point used by tests to answer confirmation
	// prompts.
	PromptFn func(question string) bool

	// RemoveFn, when non-nil, replaces fsys.Remove for the actual unlink. Tests
	// use it to record removals without touching the filesystem; setting it
	// also disables the fsys.RemoveAll fast path so every entry flows through it.
	RemoveFn func(path string) error

	diag   *core.Diag
	prompt *core.Prompter
	isTTY  bool
}
