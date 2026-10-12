# Troubleshooting

What gnux's errors, prompts and exit statuses mean and what to do about them: usage errors and unknown flags, "No such file or directory" and "Permission denied", write-protected prompts you did not expect, the preserve-root failsafe, "Is a directory" and "Directory not empty", the quiet stop of a pipeline whose reader has gone, `gnux shell` statuses 2 and 127, installation problems such as a tool name that is not found, and a FAQ. Every message quoted here is real output from gnux.

## Reading a diagnostic

Every diagnostic goes to standard error and starts with the tool's name:

```text
rmx: cannot remove 'missing.txt': No such file or directory
```

- The prefix is the x name (`rmx`), even when you typed `rm`.
- The rest is GNU's wording, so searching for the GNU message (`rm: cannot remove ... No such file or directory`) finds the same explanation.
- The system error text at the end is the same on every platform: Windows errors are normalized to the Unix wording.
- One failing operand never stops the others. `rmx a missing b` removes `a` and `b`, reports `missing`, and exits 1.

## Exit statuses

| Status | From | Meaning |
|--------|------|---------|
| 0 | every tool | Success |
| 1 | every tool | An operation failed (see the diagnostic), or a command-line mistake (followed by a `Try ... --help` line) |
| 1 | `falsex` | Always |
| 2 | `lsx` | Serious trouble, such as an operand that cannot be accessed. Minor trouble (an entry that cannot be examined) is 1 |
| 2 | `gnux shell` | Syntax error, or a usage error such as two script files |
| 125 | `envx` | `envx` itself failed (a bad option, an unknown `-C` directory) |
| 126 | `envx` | The command was found but could not be run |
| 127 | `envx` | The command was not found |
| 127 | `gnux shell` | Unknown command, or the script file could not be opened |
| 128+N | `envx` | The command was killed by signal N |
| anything | `envx`, `gnux shell` | The status of the command run, or of the script's last command |

Check it right after the command:

```bash
lsx nope; echo $?
```

```text
lsx: cannot access 'nope': No such file or directory
2
```

```powershell
lsx nope; $LASTEXITCODE
```

## Usage errors

A mistake on the command line is reported with a pointer to `--help`, and nothing is done:

```text
rmx: missing operand
Try 'rmx --help' for more information.
```

| Message | Cause | Fix |
|---------|-------|-----|
| `invalid argument "bad" for '--interactive'` | `--interactive` takes `never`, `once` or `always` | Pick one of the three; the message lists them |
| `missing operand` | No file given | Name the files. `rmx -f` with no operands is allowed and does nothing |
| `unknown flag: --no-such-flag` | The option does not exist, or is a GNU option gnux does not implement | Check `TOOL --help` and [Adoption](adoption.md) |
| `unknown shorthand flag: 'f' in -f` | The same, for a short option | As above. For `lsx -f`, use `lsx -aU` |
| `invalid date format 'yesterday'` | `touchx -d` accepts dates, times, RFC 3339, `@SECONDS` and `now` only | Write the date: `touchx -d 2026-10-10` |
| `invalid time interval '1m-bad'` | `sleepx` could not parse an operand | Use NUMBER with an optional `s`, `m`, `h` or `d` |
| `gnux: unknown tool 'frobx'` | The first argument to `gnux` is not a tool | `gnux --list` shows the names |

A usage error never touches the filesystem: `sleepx` checks every operand before pausing, and `rmx` with a bad `--interactive` value removes nothing.

## Files and directories

### No such file or directory

```text
rmx: cannot remove 'missing.txt': No such file or directory
catx: nope: No such file or directory
headx: cannot open 'notes' for reading: No such file or directory
```

The path does not exist relative to the current directory. Check it with `lsx` or `realpathx -e PATH`. On Windows in PowerShell, a wildcard such as `*.log` is not expanded for native programs, so the tool looks for a file literally named `*.log`: see [Windows and PowerShell](windows-and-powershell.md#wildcards).

For cleanup that should not care, use `rmx -f`, which ignores missing files and exits 0.

### Permission denied

```text
rmx: cannot remove 'dir/file': Permission denied
mkdirx: cannot create directory 'protected/new': Permission denied
```

Removing or creating an entry needs write and search permission on the **directory that contains it**, not on the entry itself. Check the parent with `lsx -ld PARENT`. Reading a file needs read permission on the file and search permission on every directory above it. Each tool's reference page has a Permissions section that says exactly what it needs.

### Is a directory

```text
rmx: cannot remove 'build': Is a directory
catx: dir: Is a directory
```

`rmx` without `-r` or `-d` does not remove directories. Use `rmx -r build` for a directory and its contents, or `rmx -d build` (or `rmdirx build`) for an empty one. `catx`, `headx` and `tailx` cannot print a directory.

### Directory not empty

```text
rmx: cannot remove 'build': Directory not empty
rmdirx: failed to remove 'keep/full': Directory not empty
```

`rmx -d` and `rmdirx` remove only empty directories, by design. Use `rmx -r` to remove the contents too, or `rmdirx --ignore-fail-on-non-empty` to skip non-empty ones silently.

### Omitting directory

```text
cpx: -r not specified; omitting directory 'd'
```

Add `-r` (or `-a`) to copy a directory.

### Into itself, or the same file

```text
cpx: cannot copy a directory, 'd', into itself, 'd/inside'
mvx: cannot move 'd' to a subdirectory of itself, 'd/sub'
cpx: 'a.txt' and 'a.txt' are the same file
```

The source and destination overlap. Pick a destination outside the source.

### File exists

```text
lnx: failed to create hard link 'hard.txt': File exists
```

`lnx` does not replace an existing destination unless told to. Use `lnx -f` to replace it, or `lnx -i` to be asked.

### A copy or move that silently did nothing

With `-n`, an existing destination is left alone and the status is still 0. With `-u`, a destination that is not older than the source is left alone. Add `-v` to see what was actually copied or moved.

## Prompts

### A prompt I did not ask for

```text
rmx: remove write-protected regular empty file 'c.tmp'?
```

Without `-f` or `-i`, `rmx` asks before removing a file you cannot write to, **when standard input is a terminal**. GNU `rm` does the same. Answer `y` to remove it, or use `rmx -f` to remove without asking. On Windows, a file with the read-only attribute counts as write-protected.

`mvx` asks the same question before replacing a write-protected destination on a terminal, unless `-f` is given.

When input is not a terminal (a pipe, CI, a Go program), there is no such prompt.

### A prompt that seems to be ignored

Prompts read the answer from standard input. Anything that does not start with `y` or `Y`, including end of input, means no. `echo n | rmx -i file` keeps the file; `yesx | rmx -ri dir` answers yes to everything.

### `-f` did not stop the prompts

Option order is not visible to gnux, so `-i` and `-I` win over `-f` regardless of where they appear: `rmx -if x` prompts. Drop the `-i`. See [Adoption](adoption.md#conflicting-options-resolve-toward-the-most-cautious).

### One prompt for many files

```text
rmx: remove 4 arguments?
rmx: remove 1 argument recursively?
```

That is `-I`: one confirmation before removing more than three files, or before any recursive removal.

## Refusals

### Preserve root

```text
rmx: it is dangerous to operate recursively on '/'
rmx: use --no-preserve-root to override this failsafe
```

`rmx -r /` is refused by default, on every platform and in the library (`rm.Options{}` keeps the failsafe on). If you truly mean it, `--no-preserve-root` turns it off. You almost certainly do not mean it: the usual cause is an empty variable, as in `rmx -rf $DIR/` with `DIR` unset.

### Dot and dot-dot

```text
rmx: refusing to remove '.' or '..' directory: skipping '.'
```

`rmx` never removes `.` or `..`. To empty the current directory, `cd ..` and remove it by name, or remove its contents with a glob.

## Pipelines that stop early

```bash
yesx | headx -n 3
```

`yesx` writes forever, `headx` exits after three lines, and `yesx` then finds its output closed. That is a quiet stop, not an error: no diagnostic, and the pipeline's status is `headx`'s. Every gnux tool treats a closed pipe (`EPIPE`) this way, in a system shell and in `gnux shell` alike. Any other write failure, such as a full disk, is reported and exits 1.

## The gnux shell

### Syntax error, status 2

```text
gnux: syntax error: unterminated double quote
gnux: syntax error: expected a command after '&&'
```

The line could not be parsed, so nothing on it ran. A script stops at a syntax error; an interactive session reports it and carries on. A quoted string cannot span lines.

### Command not found, status 127

```text
gnux: frob: command not found
```

Only the tools (by either name) and the builtins `cd`, `exit` and `help` are commands. External programs go through `envx`: `envx git status`. Loops, functions and `if` are not part of the language, so `for`, `while` and `if` are unknown commands too.

### The script file cannot be opened, status 127

```text
gnux: nofile.gnux: No such file or directory
```

### Extra operand, status 2

```text
gnux shell: extra operand 'b'
Try 'gnux shell --help' for more information.
```

`gnux shell` takes one script file, or `-c STRING`, not both and not two files.

### The script continued after a failure

Without `-e`, a failing command does not stop the script. Run it with `gnux shell -e`. With `-e`, a command tested by `&&` or `||` still does not stop the script, so `cmd || truex` is the way to say "this may fail".

### A variable did not expand where I expected

- `X=1 echox $X` prints an empty line: the line is expanded before the assignment runs, as in sh. The next line sees `X`.
- `F="*.go"; echox $F` prints `*.go`: a variable's value is never re-globbed or re-split, by design.
- `C:\Users\me\*.txt` loses the backslash before `*` and the glob does not match: write `C:/Users/me/*.txt`.

### `cd -` fails

```text
gnux: cd: -: No such file or directory
```

`cd -` is not supported. Store the directory in a variable first: `BACK=$PWD`, then `cd $BACK`.

## Installation problems

| Symptom | Cause | Fix |
|---------|-------|-----|
| `rmx: command not found` | The tool names were not linked, or their directory is not on `PATH` | Run the linking loop in [Getting started](getting-started.md#link-the-tool-names); `gnux rmx` works meanwhile |
| `gnux: command not found` | `$(go env GOPATH)/bin` is not on `PATH` | Add it |
| `ln` or `New-Item -ItemType SymbolicLink` fails on Windows | Symbolic links need Developer Mode or elevation | Use copies or hard links: [Windows and PowerShell](windows-and-powershell.md#make-the-per-tool-commands) |
| `rm -rf build` in PowerShell fails with "A parameter cannot be found" | `rm` is PowerShell's alias for `Remove-Item` | Use `rmx` |
| `gnux --version` prints `gnux dev` | Built without the version link flag, as plain `go install` does | Harmless; use `make install` for a stamped version |
| `install.sh` says `the Go toolchain is required` | Go is missing | Install Go 1.25 or newer |
| A link named `rm` still runs the system `rm` | An earlier `PATH` entry, or a shell alias, wins | Check `command -v rm`; put the link's directory first |

## FAQ

**Is gnux a replacement for GNU coreutils?**
No. gnux is an independent, GNU-compatible implementation that complements GNU coreutils by bringing their interface to platforms and programs where they are not available. For the 29 tools it ships, it is compatible with scripts written for the GNU tools: the same flags, semantics, diagnostics and exit statuses, with every difference documented in [Adoption](adoption.md). The rest of coreutils is on the roadmap. gnux is not affiliated with or endorsed by the GNU Project or the Free Software Foundation.

**Why the `x` suffix?**
So gnux never collides with the system's own tools or PowerShell's aliases, and both can be installed side by side. The GNU names still work as aliases when you want them.

**Can I make `rm` mean `rmx`?**
Yes: a link or copy named `rm` (or `rm.exe`) selects `rmx`. Inside `gnux shell` both names always work.

**Why did `rmx -if` prompt when GNU `rm -if` would not?**
gnux cannot see option order, so it resolves conflicts toward the most cautious option.

**Does gnux need bash, Git Bash or WSL on Windows?**
No. It is a native, static Windows program, and `gnux shell` replaces the shell for scripts.

**Why does `wcx -l *.go` fail in PowerShell?**
PowerShell does not expand wildcards for native programs. Use `gnux shell -c 'wcx -l *.go'` or `wcx -l (Get-ChildItem *.go).Name`.

**Why does `lsx` show `-` for owner and group on Windows?**
NTFS has no POSIX owner or group; those columns hold placeholders.

**Why does `lsx` sort `B` before `a`?**
`lsx` always sorts by byte value, as GNU `ls` does in the C locale, so the order is the same everywhere.

**Does `tailx -f` use inotify?**
No. It polls every `-s` seconds (1 by default), which works the same on every platform and filesystem, including network shares.

**Why does `head.Options{}` print nothing from Go?**
`Count` is zero in the zero value. Set `Count: head.DefaultLines` for `headx`'s default. See the [Go library guide](go-library.md#options-and-the-zero-value).

**Can I run two gnux shells at once in one Go process?**
Not safely: `cd` changes the process-wide working directory while a script runs.

**Where are the man pages?**
Generated in `documentation/man/` alongside markdown, reStructuredText and YAML. The [reference index](../markdown/index.md) is the place to start.

## See also

- [Getting started](getting-started.md)
- [Adoption](adoption.md)
- The failure paths of each user workflow in [SRS workflows](../../docs/requirements/srs-workflows.md)
- [Rules](../../docs/rules/README.md), including "a failure is never reported as success"
