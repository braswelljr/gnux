# Adding a Tool

The technical detail of adding a tool to gnux: an engine package under `core/<gnu>/` with `Name`, `Options`, `New` and `Run`, an adapter package under `cmd/<gnu>x/` with `Spec`, `Meta`, `Register`, `buildOptions` and `run`, one line in the registry, and regenerated documentation. It gives a complete, compiling skeleton for a hypothetical `nprocx` (GNU `nproc`) copied from the real shape of `core/pwd` and `cmd/pwdx`, explains what each piece is for, lists the test or workflow that fails for each common mistake (with the real failure output), and ends with a checklist. Registration alone makes the tool a subcommand, an `argv[0]` command under both names, a shell command, a playground command and a page in every reference format.

## What registration buys

Every consumer of a tool is built from `cmd.Tools()` in `cmd/registry.go` (NFR-26). Adding one
`Spec()` call there is what makes the tool:

| Consumer                                   | Built from                                   | Workflow |
|--------------------------------------------|----------------------------------------------|----------|
| `gnux nprocx`, `gnux nproc`                | `NewRoot` adds `Commands()`                  | [command-line-to-tool-run](command-line-to-tool-run.md) |
| `nprocx`, `nproc` as program names         | `Lookup` indexes `Tools()`                    | same     |
| `gnux --list`, hence `make links`, `install.sh`, the `Dockerfile` | `Tools()`             | same     |
| a command in `gnux shell`                  | `cmd.RunTool`                                | [shell-execution](shell-execution.md) |
| a command in the playground                | the same shell, compiled to wasm             | [wasm-playground](wasm-playground.md) |
| reference pages in four formats and the website | `cmd/docs` iterates `Commands()` and `Tools()` | [documentation-generation](documentation-generation.md) |
| the registry invariants and help/version tests | `cmd/main_test.go` iterates `Tools()`      | below    |

There is no second list to keep in step, which is the point.

## Step 1: the engine, `core/<gnu>/`

The engine owns all behavior and knows nothing of flags (NFR-39). This skeleton is
`core/pwd/pwd.go` with the names changed; it compiles and passes its tests against the current
tree:

```go
// Package nproc implements the engine behind nprocx, a GNU-compatible
// implementation of the nproc command: print the number of processing units available.
package nproc

import (
	"fmt"
	"runtime"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "nprocx"

// Options mirrors the GNU nproc flag surface. The zero value is plain nproc.
type Options struct {
	// Ignore (--ignore=N) excludes N processing units, never going below one.
	Ignore uint
}

// Nproc is the engine. Construct it with New.
type Nproc struct {
	Options

	// NumCPU is the injection point tests use to fix the processor count; it
	// defaults to runtime.NumCPU.
	NumCPU func() int

	streams core.Streams
	diag    *core.Diag
}

// New builds an nproc engine writing to s.
func New(opts Options, s core.Streams) *Nproc {
	return &Nproc{
		Options: opts,
		NumCPU:  runtime.NumCPU,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run prints the count. nproc takes no operands.
func (n *Nproc) Run(args []string) error {
	if len(args) > 0 {
		return core.Usagef("extra operand '%s'", args[0])
	}

	count := n.NumCPU()
	if uint(count) > n.Ignore {
		count -= int(n.Ignore)
	} else {
		count = 1
	}

	if _, err := fmt.Fprintln(n.streams.Out, count); err != nil && !core.IsClosedPipe(err) {
		n.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return n.diag.Result()
}
```

What each part is for:

| Part                           | Why                                                                                         |
|--------------------------------|---------------------------------------------------------------------------------------------|
| `const Name = "nprocx"`        | The diagnostic prefix, and `Meta.Name` in the adapter. One constant, so they cannot drift   |
| `Options`, zero value = default | Library callers write `nproc.Options{}` and get plain `nproc` (FR-31). A default-on safeguard is named for the negative (`NoPreserveRoot`) (NFR-36) |
| Embedded `Options`             | `n.Ignore` reads naturally, and tests can still set fields after `New`                      |
| Exported `NumCPU func() int`   | An injection point (FR-34): the test fixes the count instead of depending on the machine    |
| `core.NewDiag(Name, s)`        | GNU-format diagnostics and the error count behind `Result()` (FR-5)                         |
| `core.Usagef`                  | A command-line mistake: `cmd.Main` adds the `--help` hint. Never print the hint yourself    |
| `core.IsClosedPipe`            | When the reader of a pipeline goes away first, the tool stops quietly instead of reporting a write error (NFR-14) |
| `return n.diag.Result()`       | `core.ErrFailed` if anything was reported, else `nil`: status 1 or 0, with no second message |

Rules the skeleton follows that are easy to break in a bigger engine:

- **Only `n.streams`**, never `os.Stdout`, `os.Stdin` or `fmt.Println` (FR-32). An engine that
  writes to the process streams cannot be tested, embedded or run in the playground.
- **A failing operand never stops the rest** (FR-8): report with `Errorf` and `continue`.
- **Platform differences go in build-tagged files**, `*_unix.go` with a `!unix` (or
  `*_windows.go` plus `!windows`) fallback, never `runtime.GOOS` branches (NFR-4). Remember that
  `js/wasm` is neither `unix` nor `windows`: it takes the `!unix` file, and the Test workflow
  compiles it.
- **GNU's exact wording**, with `core.ErrText(err)` for the system part and `core.Describe(info)`
  for file-type nouns (NFR-6).

### The engine test

A black-box test in package `nproc_test`, against in-memory streams, asserting exact output
(NFR-30):

```go
package nproc_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nproc"
)

func newEngine(opts nproc.Options) (*nproc.Nproc, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	n := nproc.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	n.NumCPU = func() int { return 8 }
	return n, &out, &errb
}

func TestCount(t *testing.T) {
	tests := []struct {
		name   string
		ignore uint
		want   string
	}{
		{"all of them", 0, "8\n"},
		{"some ignored", 3, "5\n"},
		{"never below one", 99, "1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, out, errb := newEngine(nproc.Options{Ignore: tt.ignore})
			if err := n.Run(nil); err != nil {
				t.Fatalf("Run: %v (stderr %q)", err, errb.String())
			}
			if out.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestExtraOperand(t *testing.T) {
	n, out, _ := newEngine(nproc.Options{})
	var usage *core.UsageError
	if err := n.Run([]string{"x"}); !errors.As(err, &usage) {
		t.Fatalf("err = %v, want a *core.UsageError", err)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
}
```

Filesystem engines use `t.TempDir()` for every path and compare standard error byte for byte,
as `TestGetwdFailure` in `core/pwd/pwd_test.go` does:

```go
if want := "pwdx: cannot get current directory: No such file or directory\n"; errb.String() != want {
```

## Step 2: the adapter, `cmd/<gnu>x/`

The adapter declares flags, folds them into `Options`, and carries the documentation. It is
`cmd/pwdx/pwdx.go` with the names changed:

```go
// Package nprocx wires the nprocx command (GNU nproc) to its engine.
package nprocx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nproc"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the nprocx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "nproc",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for nprocx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:  nproc.Name,
		Use:   nproc.Name + " [OPTION]...",
		Short: "Print the number of processing units available.",
		Long: "nprocx prints the number of processing units available to the current process. " +
			"--ignore=N excludes N of them, but the result is never less than one.",
		Permissions: "Reads no files and needs no permissions.",
		UseCases: []string{
			"Choosing a parallelism level in a build script.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the processor count:",
				Bash:        "nprocx",
				PowerShell:  "nprocx",
			},
			{
				Description: "Leave one processor free:",
				Bash:        "make -j \"$(nprocx --ignore=1)\"",
				PowerShell:  "make -j (nprocx --ignore=1)",
			},
		},
	}
}

// Register installs the nprocx flags on fs. --help and --version are added by
// tool.Spec.Command; never define them here.
func Register(fs *pflag.FlagSet) {
	fs.Uint("ignore", 0, "if possible, exclude N processing units")
}

// buildOptions folds the parsed flags into nproc.Options.
func buildOptions(fs *pflag.FlagSet) nproc.Options {
	ignore, _ := fs.GetUint("ignore")
	return nproc.Options{Ignore: ignore}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return nproc.New(buildOptions(fs), s).Run(args)
}
```

What each `Spec` field controls, and when to set the optional ones:

| Field             | Set it when                                                                                 |
|-------------------|---------------------------------------------------------------------------------------------|
| `Meta`            | Always. `Name` = `nproc.Name`; `Use` starts with it; `Short`, `Long` and at least one example are required |
| `GNU`             | Always, for a tool: its GNU counterpart's name, which becomes the alias and the `argv[0]` name |
| `Flags`           | The tool has options. `Register` must not define `help` or `version`                        |
| `Run`             | Always                                                                                      |
| `RawArgs`         | The GNU tool does not parse options (`echo`, `true`, `false`, `yes`): arguments arrive untouched (FR-74) |
| `NoInterspersed`  | Options stop at the first operand, because the operands are another command line (`env`)   |
| `UsageExit`       | GNU uses a status other than 1 for a bad command line (`ls` 2, `env` 125)                   |
| `Topics`          | Each flag deserves its own reference page (only `rmx` today, FR-83)                         |

Write `Meta` with care: it *is* the documentation. `Long` must state any deviation from GNU,
including how conflicting flags are resolved when position cannot be tracked (toward the most
cautious, FR-9, FR-10). Give a `PowerShell` form for every example where the syntax differs
(FR-40). Use US spelling: `misspell` runs with `locale: US` (NFR-42).

### The adapter test

`buildOptions` is unexported and table-tested from inside the package, one row per flag mapping
and precedence rule (NFR-32). This is `cmd/pwdx/pwdx_test.go`'s shape:

```go
package nprocx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/nproc"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want nproc.Options
	}{
		{"defaults", nil, nproc.Options{}},
		{"ignore", []string{"--ignore=2"}, nproc.Options{Ignore: 2}},
		{"ignore as a separate word", []string{"--ignore", "3"}, nproc.Options{Ignore: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(nproc.Name, pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("parse %v: %v", tt.args, err)
			}
			if got := buildOptions(fs); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}
```

When `buildOptions` can fail (an invalid enum value, as `cmd/rmx/interactive.Apply` does for
`--interactive`), return `(Options, error)` and make the error a `core.Usagef`.

## Step 3: register it

Add the import and the call to `cmd/registry.go`, keeping both alphabetical, since `Tools()` is
also the order of `--list`, the help and the documentation:

```go
"github.com/braswelljr/gnux/cmd/mvx"
"github.com/braswelljr/gnux/cmd/nprocx"
// ...
		mvx.Spec(),
		nprocx.Spec(),
		pwdx.Spec(),
```

With the skeleton above registered in a scratch copy of the tree and built with `make build`,
the real binary behaves as the contract says:

```text
$ ./bin/gnux nprocx
15
$ ./bin/gnux nprocx --ignore=2
13
$ ./bin/gnux nproc x
nprocx: extra operand 'x'
Try 'nprocx --help' for more information.          # status 1
$ ./bin/gnux nprocx --ignore=-1
nprocx: invalid argument "-1" for "--ignore" flag: strconv.ParseUint: parsing "-1": invalid syntax
Try 'nprocx --help' for more information.          # status 1
$ ./bin/gnux nprocx --version
nprocx (gnux) dev
```

## Step 4: read the generated pages and update the hand-written lists

```bash
make docs        # documentation/{markdown,man,yaml,rest} and web/content/docs; not committed
```

Then, by hand, in the same change:

- the tool table in `README.md`, and remove the tool from its Roadmap;
- the tool list in `AGENTS.md`;
- a new `FR-n` section for the tool in
  [functional-requirements](../requirements/functional-requirements.md) (synopsis, option
  table, behavior, deviations, "Verified by"), using the next free number;
- optionally, an example in `web/src/config/playground.ts`, whose `needs` field hides it if
  the wasm build lacks the tool.

The generated reference needs no hand edits: see
[documentation-generation](documentation-generation.md).

## Which check catches which mistake

| Mistake                                                         | What fails                                                    | Requirement |
|-----------------------------------------------------------------|---------------------------------------------------------------|-------------|
| Name is not the GNU name plus `x` (`GNU: "nprocs"`)             | `TestRegistryIsConsistent`: `tool "nprocx" mirrors GNU "nprocs"; want name "nprocsx"` | FR-1 |
| `Use` does not start with the name                              | `TestRegistryIsConsistent`: `Use "[OPTION]..." must start with the tool name` | FR-39 |
| `Short`, `Long` or every example missing                        | `TestRegistryIsConsistent`: `Short, Long and at least one example are required` | FR-39 |
| `Run` not set                                                   | `TestRegistryIsConsistent`: `has no Run`                      | NFR-25      |
| Name or GNU name already registered                             | `TestRegistryIsConsistent`: `name "..." is registered twice`  | NFR-26      |
| `Register` defines `--version` or `--help`                      | `TestEveryToolAnswersHelpAndVersion` (and every test that builds the root) panics: `nprocx flag redefined: version` | FR-4 |
| A flag maps to the wrong field, or a precedence rule is wrong   | `TestBuildOptions` in `cmd/<gnu>x`, if it has the row          | FR-2, NFR-32 |
| Engine writes to `os.Stdout`                                    | The engine test's exact-output assertion finds an empty buffer | FR-32      |
| Wrong diagnostic wording                                        | The engine test's exact-stderr assertion                       | FR-5        |
| Platform code with no fallback for `js/wasm` or Windows         | The "Build for WebAssembly" step of the Test workflow; the Build workflow | NFR-3, NFR-4 |
| A data race in a concurrent engine                              | `go test -race` (`make test`, the Test workflow)              | NFR-15      |
| A per-flag `Topic` page missing or a broken link in markdown    | `TestGenerateAll`, `TestMarkdownLinksResolve` (`cmd/docs/docs_test.go`) | FR-42, FR-83 |
| Metadata the generator cannot render                            | The Docs workflow and `make docs`                             | FR-41       |
| A misspelling or lint finding                                   | The Lint workflow (`make lint`)                               | NFR-28, NFR-42 |
| The engine imports cobra or pflag                               | Nothing automatic; review                                     | NFR-39      |
| The `Spec()` call never added to `Tools()`                      | Nothing automatic: every check iterates the registry, so an unregistered tool is invisible to all of them | NFR-26 |

The real `TestRegistryIsConsistent` and `TestEveryToolAnswersHelpAndVersion` failures quoted
above were produced by breaking the skeleton on purpose in a scratch copy of the tree:

```text
--- FAIL: TestEveryToolAnswersHelpAndVersion (0.00s)
    --- FAIL: TestEveryToolAnswersHelpAndVersion/nprocx (0.00s)
panic: nprocx flag redefined: version [recovered, repanicked]
--- FAIL: TestRegistryIsConsistent (0.00s)
    main_test.go:44: tool "nprocx" mirrors GNU "nprocs"; want name "nprocsx"
```

`TestGenerateAll` and `TestGenerateMDX` in `cmd/docs/docs_test.go` require a page per
registered command in every format, so a registered tool is documented automatically (NFR-43).

## Checklist

1. `core/<gnu>/<gnu>.go`: `Name`, `Options` (zero value = default; negatives for default-on
   safeguards), engine struct with exported injection points, `New`, `Run`.
2. Diagnostics only through `core.NewDiag(Name, s)`; system text through `core.ErrText`.
3. All I/O through the `core.Streams` from `New`; closed pipes stop quietly.
4. Every operand attempted; `Run` returns `nil`, `core.ErrFailed`, `*core.ExitError` or
   `core.Usagef(...)`.
5. Platform code in `*_unix.go` plus a `!unix` fallback that compiles for Windows and `js/wasm`.
6. `core/<gnu>/<gnu>_test.go`: black-box, `t.TempDir()`, in-memory streams, exact output.
7. `cmd/<gnu>x/<gnu>x.go`: `Spec`, `Meta` (deviations and precedence in `Long`, Bash and
   PowerShell examples), `Register` (no `--help`/`--version`), `buildOptions`, `run`.
8. `cmd/<gnu>x/<gnu>x_test.go`: `TestBuildOptions` with a row per mapping and precedence rule.
9. `cmd/registry.go`: import and `Spec()` call, alphabetical.
10. `make docs`, and read the new pages (they are not committed).
11. `README.md` table and Roadmap, `AGENTS.md` list, a new `FR-n` in the functional
    requirements.
12. `make ci` (vet, lint, race tests, docs generation), and `make web/test` if the tool matters
    to the playground.
13. Commit code, tests and the hand-written updates together.

## Related

- The user-facing version of this flow: WF-7 in
  [srs-workflows](../requirements/srs-workflows.md).
- [engine-run](engine-run.md) for a full-featured engine (`core/rm`), including concurrency and
  prompting.
- [error-handling-and-exit-status](error-handling-and-exit-status.md) for the error contract.
- The [rules](../rules/README.md) the code follows, and the [guides](../../documentation/guides/README.md).
