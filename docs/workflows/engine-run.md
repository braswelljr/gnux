# Engine Run

What happens inside a tool's engine once the adapter has built its `Options`: construction with `New`, the operand loop in `Run`, diagnostics through `core.Diag`, prompting through `core.Prompter` or an injected function, the decision between sequential and concurrent work, and the error that tells `cmd.Main` how the run went. The worked example is `core/rm`, the most complete engine, followed by how `core/mv` and `core/cp` differ, in particular the cross-filesystem move that falls back from a rename to an archive copy. Everything here is public library API, so the same path runs whether the engine is reached from the command line, the shell, the playground or a Go program.

## Where the engine sits

An engine is the package `core/<gnu>/`. It never sees cobra or pflag (NFR-39): the adapter in
`cmd/<gnu>x/` has already turned flags into a plain struct, and
[command-line-to-tool-run](command-line-to-tool-run.md) has already chosen the tool. What the
engine receives is two values and a slice:

```go
engine := rm.New(opts, streams) // opts rm.Options, streams core.Streams
err := engine.Run(args)         // args []string, the operands
```

and what it returns is one of four things (FR-33):

| Return            | Meaning                                                  |
|-------------------|----------------------------------------------------------|
| `nil`             | every operand succeeded                                  |
| `core.ErrFailed`  | it ran, something failed, the diagnostics are written    |
| `*core.ExitError` | a specific status (`lsx` 2, `envx` 125 to 127)           |
| `*core.UsageError`| the operands themselves are wrong (`missing operand`)    |

## Step 1: `Options`, where the zero value is the plain tool

`core/rm/model.go` defines `Options` so that `rm.Options{}` is `rm` with no flags (FR-31,
NFR-36):

```go
type Options struct {
	Force         bool // -f
	Recursive     bool // -r/-R
	Dir           bool // -d
	Verbose       bool // -v
	Interactive   Mode // -i/-I/--interactive
	OneFileSystem bool // --one-file-system
	// NoPreserveRoot (--no-preserve-root) allows recursing on '/'. It is the
	// negative so the zero value keeps the failsafe on.
	NoPreserveRoot bool
}
```

Two design choices carry the weight here:

- **Safety-critical defaults are spelled as negatives.** A library caller writing
  `rm.Options{Recursive: true}` gets the `/` failsafe without knowing it exists. A field named
  `PreserveRoot` would default to `false` and silently remove it.
- **`Mode`'s zero value is `PromptDefault`.** `PromptDefault` is GNU's behavior with none of
  `-f`, `-i`, `-I`: prompt only for a write-protected file, and only on a terminal.

## Step 2: `New` captures streams and environment once

```go
func New(opts Options, s core.Streams) *Rm {
	return &Rm{
		Options: opts,
		diag:    core.NewDiag(Name, s),
		prompt:  core.NewPrompter(Name, s),
		isTTY:   core.IsTerminal(s.In),
	}
}
```

- `core.NewDiag(Name, s)` builds the diagnostic writer. `Name` is the constant `"rmx"`, so every
  message is prefixed with the tool's name whichever way it was invoked (FR-5).
- `core.NewPrompter(Name, s)` wraps `s.In` in one `bufio.Reader`. One reader per engine matters:
  two readers on the same stdin would each buffer ahead and steal each other's answers.
- `core.IsTerminal(s.In)` is checked once. It is `true` only for an `*os.File` attached to a
  terminal (or a Cygwin pty), so a test buffer or a shell pipe is never a terminal and never
  triggers the write-protected prompt (NFR-30).

The engine also exposes injection points as exported fields (FR-34):

```go
PromptFn func(question string) bool // answers prompts instead of reading stdin
RemoveFn func(path string) error    // replaces os.Remove, and disables the fast path
```

## Step 3: `Run` validates, then picks sequential or concurrent

`Run` in `core/rm/rm.go`, in order:

1. **No operands.** `-f` with no operands succeeds silently; otherwise it is
   `core.Usagef("missing operand")`, which `cmd.Main` prints with the `--help` hint.
2. **`-I` asks once.** With `PromptOnce`, a recursive removal or more than three operands asks
   `remove N arguments?` (or `... recursively?`). A "no" returns `nil`: declining is not a
   failure. Then `Interactive` is set to `PromptNever` for the rest of the run.
3. **Choose the strategy.**

   ```go
   if r.mayPrompt() || r.Verbose {
   	for _, p := range paths {
   		r.removeArg(p)
   	}
   } else {
   	r.runConcurrent(paths)
   }
   return r.diag.Result()
   ```

   `mayPrompt` is true for `PromptAlways`, or for `PromptDefault` on a terminal. Prompting runs
   sequentially because every prompt reads the one stdin. `-v` runs sequentially because its
   lines must come out in argument order (NFR-10). Everything else fans out (NFR-9).

4. **Return `diag.Result()`**: `core.ErrFailed` if any error was counted, otherwise `nil`.

Because step 2 mutates `r.Interactive`, an `Rm` is built for one `Run`. Build a new engine with
`New` for each invocation, as the adapter does.

### Concurrency: `runConcurrent`

```go
workers := min(runtime.NumCPU(), len(paths))
jobs := make(chan string)
// workers goroutines: for p := range jobs { r.removeArg(p) }
```

One worker per CPU, capped at the number of operands, pulling top-level operands from an
unbuffered channel. A single operand skips the goroutines entirely. The workers share one
`core.Diag`, which is safe because every `Diag` method takes its mutex; the `errCount` that
`Result` reads is guarded by the same lock. The suite runs under `-race` (NFR-15, NFR-31), and
`TestConcurrentBulkRemoval` drives this path with six files.

Only *top-level* operands run in parallel. A single directory is not split up; it is handed to
one worker and removed by the fast path or by `removeTree`.

## Step 4: one operand, `removeArg`

```text
removeArg(path)
  base is "." or ".."           -> Errorf "refusing to remove '.' or '..' directory: skipping '%s'"
  os.Lstat(path) fails
      not exist and -f          -> silent
      not exist                 -> Errorf "cannot remove '%s': No such file or directory"
      other                     -> Errorf "cannot remove '%s': <ErrText>"
  directory
      -r, not NoPreserveRoot, core.IsRoot(path)
                                -> Errorf "it is dangerous to operate recursively on '%s'"
                                   Warnf  "use --no-preserve-root to override this failsafe"
      otherwise                 -> removeDirArg
  anything else                 -> removeFileEntry
```

`os.Lstat` is used throughout, so a symbolic link is removed and never followed (FR-15). Each
failure is written and counted, and the loop moves on to the next operand: a failing operand
never stops the rest (FR-8). The `Warnf` line is a second line of the same failure, so it does
not count twice.

Real output (FR-12):

```text
$ rmx . ..
rmx: refusing to remove '.' or '..' directory: skipping '.'
rmx: refusing to remove '.' or '..' directory: skipping '..'
$ rmx -r /
rmx: it is dangerous to operate recursively on '/'
rmx: use --no-preserve-root to override this failsafe
```

## Step 5: directories, `removeDirArg`

| Flags        | What happens                                                                                  |
|--------------|-----------------------------------------------------------------------------------------------|
| `-r`, fast path allowed | `os.RemoveAll(path)` in one call                                                   |
| `-r` otherwise | `removeTree(path, dev, haveDev)`, the post-order walk                                       |
| `-d`         | `os.ReadDir`; non-empty is `Directory not empty`; else `confirmDir`, `del`, optional `removed directory '%s'` |
| neither      | `cannot remove '%s': Is a directory`                                                          |

The `-d` branch checks emptiness itself instead of trusting the `rmdir` error, because the OS
text for a non-empty directory differs by platform, and GNU's wording is the contract (NFR-6).

```text
$ rmx d
rmx: cannot remove 'd': Is a directory
$ rmx -d d
rmx: cannot remove 'd': Directory not empty
```

### The fast path

```go
func (r *Rm) fastPathOK() bool {
	return r.RemoveFn == nil && !r.mayPrompt() && !r.Verbose && !r.OneFileSystem
}
```

`os.RemoveAll` is much faster than a walk that stats every entry, but it makes no per-entry
decisions. So it is used only when none is needed: nobody can be asked anything, nothing is
reported per entry, no device boundary has to be checked, and no test has injected `RemoveFn`
to observe each removal.

### Post-order recursion: `removeTree`

`removeTree` reads the directory, then:

1. With `-i` and a non-empty directory, asks `descend into directory '%s'?`. "No" leaves the
   whole subtree.
2. For each child, `os.Lstat`s it. A subdirectory on another device under `--one-file-system`
   is skipped with a warning (`core.DeviceID`, which is `syscall.Stat_t.Dev` on Unix and
   unsupported elsewhere, so the flag is a no-op off Unix: NFR-5). Subdirectories recurse;
   everything else goes to `removeFileEntry`.
3. If any child was skipped or failed, it returns `false` **without** trying to remove the
   directory: removing it would only fail with `Directory not empty`, a second diagnostic for
   the same cause.
4. Otherwise `confirmDir` (with `-i`, `remove directory '%s'?`), then `del`, then the
   `removed directory '%s'` line under `-v`.

Children are removed before their parent, which is the order `-v` reports:

```text
$ gnux shell -c 'mkdirx -p t/a && touchx t/a/f t/g && rmx -rv t'
removed 't/a/f'
removed directory 't/a'
removed 't/g'
removed directory 't'
```

### Non-directories: `removeFileEntry`

```go
switch {
case r.Interactive == PromptAlways:
	ask("remove %s '%s'?", core.Describe(info), path)
case r.Interactive == PromptDefault && r.isTTY && !core.Writable(path):
	ask("remove write-protected %s '%s'?", core.Describe(info), path)
}
```

`core.Describe` supplies GNU's noun (`regular empty file`, `symbolic link`, `fifo`, ...).
`core.Writable` is `access(2)` with `W_OK` on Unix (`core/fs_unix.go`) and the owner write bit
on Windows and WebAssembly (`core/fs_other.go`), which is how the read-only attribute surfaces.
After a successful `del`, `-v` prints `removed '%s'`. Under `-f`, a file that vanished between
the walk and the unlink is not an error.

## Step 6: prompting

All prompts go through `r.ask`, which prefers the injected `PromptFn` and otherwise calls
`core.Prompter.Ask`:

```go
fmt.Fprintf(p.err, "%s: %s ", p.prog, question) // to standard error, like GNU
line, err := p.reader.ReadString('\n')
if err != nil && line == "" {
	return false                                // end of input means "no"
}
return line != "" && (line[0] == 'y' || line[0] == 'Y')
```

End of input counting as "no" is the safe reading of a closed stdin: a script that pipes nothing
into `rmx -i` removes nothing (FR-13).

```text
$ echo n | rmx -i d/f
rmx: remove regular empty file 'd/f'?          # status 0, file kept
```

The precedence among `-f`, `-I`, `--interactive=WHEN` and `-i` is not decided here: the adapter
(`cmd/rmx/force` then `cmd/rmx/interactive`) has already resolved it toward the most cautious
mode (FR-9).

## Step 7: diagnostics and the result

`core.Diag` has three writers:

| Method   | Stream | Counts toward failure | Used for                                      |
|----------|--------|-----------------------|-----------------------------------------------|
| `Errorf` | stderr | yes                   | every failed operation                        |
| `Warnf`  | stderr | no                    | the second line of a refusal, a skipped device |
| `Printf` | stdout | no                    | `-v` reports                                  |

Every message goes through `core.ErrText(err)` for the system part, so `No such file or
directory`, `Permission denied`, `File exists` and `Directory not empty` read the same on every
platform (FR-5, NFR-6). At the end `diag.Result()` turns "anything counted" into
`core.ErrFailed`, and `cmd.Main` maps that to status 1 without printing anything more (NFR-45).
See [error-handling-and-exit-status](error-handling-and-exit-status.md).

## How `core/mv` differs

`core/mv/mv.go` has the same shape (`Name = "mvx"`, `Options`, `New`, `Run`, `Diag`,
`Prompter`, `PromptFn`) with these differences:

- **Operand forms are parsed in `Run`.** `SOURCE DEST`, `SOURCE... DIRECTORY` and `-t DIR
  SOURCE...` are resolved first, and malformed lines are usage errors: `missing file operand`,
  `missing destination file operand after '%s'`, `extra operand '%s'`,
  `target '%s' is not a directory`, and `-t` with `-T`.
- **Always sequential.** No fan-out: the destination of one operand can be the source or the
  target of another, and `-v` order matters.
- **Duplicate destinations are refused.** `moveOperand` records each destination in `moved`, so
  `mv a/f b/f dir` refuses the second with `will not overwrite just-created '%s' with '%s'`.
- **Checks before the rename.** Same file (`'%s' and '%s' are the same file`), a directory into
  itself (`cp.IsWithin`), `-u` and `-n` skips, directory/non-directory mismatches, and
  `confirmOverwrite` (`-i`, or a write-protected destination on a terminal without `-f`). A
  case-only rename (`foo` to `Foo` on a case-insensitive filesystem) is recognized by
  `sameNameIgnoringCase` and treated as a rename, not an overwrite of itself.
- **An existing empty directory is removed first**, because `os.Rename` refuses every existing
  directory while `rename(2)` would replace an empty one.

### The cross-filesystem move (FR-16)

The rename is attempted through `m.RenameFn` (default `os.Rename`). If it fails with a
cross-device error, `move` falls back to `moveAcross`:

```text
RenameFn(src, dst)
  ok                      -> "renamed '%s' -> '%s'" under -v
  isCrossDevice(err)      -> moveAcross(src, dst, exists)
  other                   -> "cannot move '%s' to '%s': <errText>"

moveAcross
  1. if dst exists: os.Remove(dst)       (replace, never merge)
  2. copier := cp.NewNamed("mvx", cp.Options{Recursive: true,
                 Dereference: cp.DerefNever, Preserve: cp.PreserveAll()}, streams)
  3. copier.Copy(src, dst)               failure -> copyFailed = true, source untouched
  4. os.RemoveAll(src)                   only after the copy completed
```

Why it is built this way:

- **`isCrossDevice` is build-tagged.** `core/mv/errors_unix.go` checks `syscall.EXDEV`;
  `core/mv/errors_windows.go` also checks `ERROR_NOT_SAME_DEVICE` (errno 17), which is what
  `MoveFileEx` returns between volumes (NFR-4).
- **The copy is an archive copy.** Symbolic links stay links, and mode, ownership, timestamps
  and hard links are preserved, so the moved tree is indistinguishable from a renamed one.
- **`cp.NewNamed` keeps the diagnostics under `mvx`.** A failure deep in the copy reads
  `mvx: ...`, not `cpx: ...`.
- **One copier per run.** It is built on first use and kept, so hard links between two operands
  survive the move.
- **The copier has its own `Diag`.** Its error count is not `m.diag`'s, so `Run` tracks
  `copyFailed` and returns `core.ErrFailed` if it is set, even when `m.diag` counted nothing.
- **The source goes last.** Nothing is deleted until the destination is complete; a failed copy
  leaves the source exactly as it was.
- **`errText` corrects one platform quirk.** The standard library folds `ENOTEMPTY` into "file
  exists", so `mv` maps it back to `Directory not empty` (`isNotEmpty`, also build-tagged).

`RenameFn` exists so the suite can simulate a second filesystem: `TestCrossDeviceFile`,
`TestCrossDeviceDirectory`, `TestCrossDeviceNeverMergesIntoNonEmptyDirectory` and
`TestCrossDeviceKeepsSourceWhenCopyFails` in `core/mv/mv_test.go` replace it with a function
that returns `EXDEV`.

## How `core/cp` differs

`core/cp/cp.go` is also sequential, and adds:

- **`New` is `NewNamed(Name, ...)`.** `NewNamed` is the hook `mv` uses, above.
- **`Copy(src, dst)` is exported.** It copies one entry to exactly `dst` (never "into"), which is
  what `mv` needs.
- **Destinations are created with `O_CREATE|O_EXCL`** and the source's permission bits, so the
  kernel's umask applies exactly as for GNU `cp` without `-p`. An existing destination is opened
  with `O_TRUNC`; under `-f`, one that cannot be opened is removed and created again.
- **Hard links are tracked by (device, inode)** in `links` when `Preserve.Links` is set, so a
  file with several names is copied once and its other names become links to the copy.
- **Attributes are applied after the data** by `preserveAttrs`, ownership first because `chown`
  clears setuid and setgid. The platform half (`inode` and `chown`) is in `attr_unix.go` and
  `attr_windows.go`; on Windows both report "unsupported", so ownership is best effort and hard
  links between sources are not detected there (NFR-5).
- **The dereference policy is a `Deref` value** (`DerefDefault`, `DerefNever`,
  `DerefCommandLine`, `DerefAlways`) chosen by the adapter from `-L`, `-H`, `-P` and `-d`,
  with `-P`/`-d` over `-H` over `-L` (FR-9).
- **Duplicate destinations** are refused as in `mv`, except that same-named directories merge.

## Tests that pin this workflow

| Test                                                                                   | What it proves                         | Requirement |
|----------------------------------------------------------------------------------------|----------------------------------------|-------------|
| `TestRemoveNonexistent`, `TestRemoveDirectoryRequiresRecursive`, `TestRemoveNonEmptyDirWithDirFlag` (`core/rm`) | exact diagnostics and `ErrFailed` | FR-5, FR-8 |
| `TestRefusesDotDirectory` (`core/rm`)                                                  | `.` and `..` are refused               | FR-12       |
| `TestInteractivePrompt`, `TestPromptOnceDecline` (`core/rm`)                           | prompt answers via `PromptFn`; `-I` asks once | FR-13 |
| `TestConcurrentBulkRemoval` (`core/rm`)                                                | the fan-out path, under `-race`        | NFR-9, NFR-15 |
| `TestVerboseOutput`, `TestInjectedRemoveFn` (`core/rm`)                                | ordered `-v` output; every entry flows through `RemoveFn` | NFR-10 |
| `Example` (`core/rm/example_test.go`)                                                  | the engine as a library                | FR-30       |
| `TestCrossDevice*` (`core/mv`)                                                         | the cross-filesystem fallback          | FR-16       |
| `TestErrText` (`core/core_test.go`)                                                    | platform-normalized error text         | NFR-6       |

## Related

- [command-line-to-tool-run](command-line-to-tool-run.md) for how the engine is reached.
- [error-handling-and-exit-status](error-handling-and-exit-status.md) for what the returned
  error becomes.
- [adding-a-tool](adding-a-tool.md) for writing a new engine in this shape.
- WF-4 in [srs-workflows](../requirements/srs-workflows.md) for embedding an engine from Go.
- [functional-requirements](../requirements/functional-requirements.md) sections 3 and 6, and
  the [rules](../rules/README.md).
