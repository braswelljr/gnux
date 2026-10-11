# Functional Requirements

Functional requirements define what gnux must do: the behavior every tool shares, the options
each of the 22 tools must support, the safeguards on destructive operations, how a tool is
invoked, the portable shell, the Go library, installation, the generated documentation, the
website, the playground and its discoverability. Each requirement has a stable identifier
(`FR-n`), a rationale, acceptance criteria and the check that verifies it.

**This document stands on its own.** It is the register of record for what the software must
do. Why it must is in [business-requirements](business-requirements.md); how well is in
[non-functional-requirements](non-functional-requirements.md); the step-by-step flows are in
[srs-workflows](srs-workflows.md).

Requirement levels: **must** is mandatory, **should** is expected unless there is a stated
reason not to, **may** is optional.

## How to read this register

- **Identifiers are stable, not sequential.** `FR-1` to `FR-50` come from the first draft of
  this register and keep their numbers. `FR-51` to `FR-72` are the 22 tools, in alphabetical
  order. `FR-73` onward are requirements added since. The document is ordered by area, so the
  numbers do not run in order down the page.
- **"Verified by" names something that exists.** A test file, a CI workflow, or the word
  "review" where nothing automated checks the requirement. A requirement with no automated
  check says so.
- **Per-tool option tables are taken from the code.** Each table lists what the tool's
  `Register` function in `cmd/<tool>x/<tool>x.go` defines. An option that is not in the table
  is not supported, and giving it is a usage error.

## 1. Behavior common to every tool

### FR-1: naming

Each tool must be named after its GNU counterpart with an `x` appended (`rm` is `rmx`).

- **Rationale.** The name is the whole of the learning curve, and it must never collide with
  the system's own tool.
- **Acceptance.** For every registered tool, the command name equals the GNU name plus `x`, and
  no name is registered twice.
- **Verified by.** `TestRegistryIsConsistent` in `cmd/main_test.go`.

### FR-2: options and their meaning

Each tool must accept the GNU tool's commonly used short and long options, with the same
meaning.

- **Rationale.** BR-1. Muscle memory is in the options.
- **Acceptance.** The options listed for the tool in section 2 are accepted and map to the
  behavior stated there.
- **Verified by.** `cmd/<tool>x/<tool>x_test.go` (`TestBuildOptions`), which pins every option
  to the engine field it sets; `core/<tool>/*_test.go` for the behavior behind each field.

### FR-3: option syntax

Options must follow GNU conventions: combinable short flags (`-rvf`), `--name=value`, and `--`
to end option parsing.

- **Rationale.** Existing command lines are written this way.
- **Acceptance.** `rmx -rvf dir` equals `rmx -r -v -f dir`; a long option takes its value after
  `=`; operands after `--` are not parsed as options. The four tools whose arguments are not
  GNU-style options (`echox`, `truex`, `falsex`, `yesx`) parse their own arguments, as their
  GNU counterparts do.
- **Verified by.** The adapter suites; `cmd/echox`, `cmd/yesx`, `cmd/truex` and `cmd/falsex`
  tests for the four exceptions.

### FR-4: help and version

Every tool must accept `--help` and `--version`. `-h` must remain available for tools that use
it for something else (`lsx -h`, `touchx -h`).

- **Rationale.** Every GNU tool answers both, and GNU tools take `--help` only.
- **Acceptance.** Both options exit 0 and print to standard output for every registered tool.
  `--help` has no shorthand. The `--help` screen has a usage line, the description, the options,
  the first six examples, the exit statuses and a pointer to `gnux help TOOL` and the online
  reference (`TestToolHelpPointsToTheManual`).
- **Verified by.** `TestEveryToolAnswersHelpAndVersion` in `cmd/main_test.go`; the `lsx` and
  `touchx` adapter suites for `-h`.

### FR-5: diagnostics

Diagnostics must go to standard error as `<tool>: <message>`, using GNU's wording.

- **Rationale.** Scripts and people both match on these messages.
- **Acceptance.** `rmx missing` writes `rmx: cannot remove 'missing': No such file or directory`
  to standard error and nothing to standard output. The system-error part of a message is the
  same text on every platform for the common cases (not found, permission denied, exists,
  directory not empty).
- **Verified by.** The engine suites, which compare standard error exactly; `TestErrText` in
  `core/core_test.go`.

### FR-6: usage errors

A command-line mistake must print the diagnostic followed by
`Try '<tool> --help' for more information.`

- **Rationale.** A refusal the user can correct should say what was wrong and where to look.
- **Acceptance.** A missing operand, an unknown option and an invalid option value each print
  `<tool>: <what was wrong>` and then the hint, on standard error. An operation that fails at
  run time (a file that does not exist) prints its diagnostic without the hint.
- **Verified by.** `TestDiagnostics` in `cmd/main_test.go`.

### FR-7: exit statuses

Exit statuses must match GNU: 0 on success, 1 on failure, and tool-specific values where GNU
defines them.

- **Rationale.** Scripts branch on the status.
- **Acceptance.** `lsx` exits 1 for minor problems and 2 for serious trouble, including a bad
  command line. `envx` exits 125 when it fails itself, 126 when the command cannot be run, 127
  when it cannot be found, and otherwise the command's own status. `falsex` exits 1. Every
  other tool exits 0 or 1.
- **Verified by.** `cmd/main_test.go`; `core/ls`, `core/env` and `cmd/envx` suites.

### FR-8: one failing operand does not stop the rest

A failing operand must not prevent the remaining operands from being processed.

- **Rationale.** `rm a b c` with a missing `b` still removes `a` and `c`. A tool that stopped at
  `b` would leave work half done in a way GNU does not.
- **Acceptance.** Every operand is attempted, each failure is reported, and the exit status is
  nonzero if any failed.
- **Verified by.** `TestErrorsDoNotStopLaterOperands` (`core/cat`, `core/head`, `core/tail`),
  `TestFailureDoesNotStopOtherOperands` (`core/cp`), `TestFailureDoesNotStopLaterOperands`
  (`core/mkdir`), `TestQuietAndPartialFailure` (`core/realpath`) and the other engine suites.

### FR-9: conflicting options

Where conflicting options cannot be resolved by position, the tool must choose the most
cautious behavior and document the rule.

- **Rationale.** GNU lets the last of conflicting options win. The command-line framework does
  not report option order, so a tool that cannot track it has to pick, and the safe pick is the
  one that destroys least.
- **Acceptance.** `rmx`: `-i` over `--interactive=WHEN` over `-I` over `-f`. `mvx`: `-n` over
  `-i` over `-f`. `cpx`: `-n` over `-i`; `-P` or `-d` over `-H` over `-L`. `lnx`: `-i` over
  `-f`. Each rule is stated in the tool's reference page. `lsx`, `headx` and `tailx` track
  option order themselves and keep GNU's last-one-wins.
- **Verified by.** The `rmx`, `mvx`, `cpx`, `lnx`, `lsx`, `headx` and `tailx` adapter suites.

### FR-10: documented deviations

Any option of the GNU tool that is not implemented, and any behavior that differs, must be
documented.

- **Rationale.** BR-11.
- **Acceptance.** Behavior differences are in the tool's reference page. Unimplemented options
  are listed in section 2 of this document.
- **Verified by.** Review. The generated reference pages do not carry a list of unimplemented
  options; this register is where that list lives.

### FR-11: standard input

Tools that read input must accept standard input when no file operand, or `-`, is given.

- **Rationale.** Pipelines.
- **Acceptance.** `catx`, `headx`, `tailx` and `wcx` read standard input with no operand and
  when an operand is `-`.
- **Verified by.** `core/cat`, `core/head`, `core/tail` and `core/wc` suites (`TestStdinFile`
  for `wcx`).

### FR-73: an unknown option is refused

A tool must refuse an option it does not define, and must never ignore it.

- **Rationale.** An ignored `--no-preserve-root` or `-n` is how a safe command becomes an
  unsafe one. A refusal is the only honest answer to an option that is not implemented.
- **Acceptance.** An undefined option is a usage error under FR-6, and nothing is done.
- **Verified by.** `TestDiagnostics` in `cmd/main_test.go` (`rmx --no-such-flag`).

### FR-74: tools that parse their own arguments

`echox`, `truex`, `falsex` and `yesx` must receive their arguments untouched, and must still
honor `--help` and `--version` when either is the sole argument.

- **Rationale.** `echo -x` prints `-x`; `true --anything` succeeds. Their arguments are data,
  not options.
- **Acceptance.** `echox --help` prints help; `echox --help extra` prints both words.
- **Verified by.** `TestLoneVersion` and `TestCommand` in `cmd/echox`, `cmd/truex`,
  `cmd/falsex` and `cmd/yesx`.

## 2. The tools

One requirement per tool. Each states the synopsis, the options that must be supported, the
behavior that must hold, and the documented deviations from the GNU tool. `--help` and
`--version` apply to every tool under FR-4 and are not repeated.

Several GNU options exist only as a single letter. The command-line framework requires a long
name for every option, so those letters also answer to a long name of gnux's own, which is
shown in the tables and noted as a deviation.

### FR-51: basenamex

`basenamex` must strip the directory and, optionally, a suffix from file names, as GNU
`basename` does.

```text
basenamex [OPTION]... NAME [SUFFIX]
```

| Option             | Meaning                                                      |
|--------------------|--------------------------------------------------------------|
| `-a`, `--multiple` | Treat every operand as a NAME                                |
| `-s`, `--suffix=SUFFIX` | Remove a trailing SUFFIX from each name; implies `-a`   |
| `-z`, `--zero`     | End each output line with NUL, not newline                   |

- **Behavior.** Trailing slashes are ignored; a name made only of slashes yields the root. A
  suffix is not removed when that would leave nothing. The operands are treated as text: the
  filesystem is never touched. No operand, or more than two without `-a`, is a usage error.
- **Deviations.** None known.
- **Verified by.** `core/basename/basename_test.go`, `cmd/basenamex/basenamex_test.go`.

### FR-52: catx

`catx` must concatenate files to standard output, as GNU `cat` does.

```text
catx [OPTION]... [FILE]...
```

| Option                     | Meaning                                              |
|----------------------------|------------------------------------------------------|
| `-A`, `--show-all`         | Equivalent to `-vET`                                 |
| `-b`, `--number-nonblank`  | Number nonempty output lines; overrides `-n`         |
| `-e`                       | Equivalent to `-vE`                                  |
| `-E`, `--show-ends`        | Display `$` at the end of each line                  |
| `-n`, `--number`           | Number all output lines                              |
| `-s`, `--squeeze-blank`    | Suppress repeated empty output lines                 |
| `-t`                       | Equivalent to `-vT`                                  |
| `-T`, `--show-tabs`        | Display TAB characters as `^I`                       |
| `-u`                       | Accepted and ignored                                 |
| `-v`, `--show-nonprinting` | Use `^` and `M-` notation, except for LFD and TAB    |

- **Behavior.** With no options the bytes are copied untouched, so binary data is safe. Line
  numbering and blank-line squeezing continue across files. A directory operand is reported as
  `Is a directory`. Writing to a closed pipe ends the tool quietly.
- **Deviations.** `-e`, `-t` and `-u` also answer to the long names `--show-nonprinting-ends`,
  `--show-nonprinting-tabs` and `--unbuffered`, which GNU does not have.
- **Verified by.** `core/cat/cat_test.go`, `cmd/catx/catx_test.go`.

### FR-53: cpx

`cpx` must copy files and directories, as GNU `cp` does.

```text
cpx [OPTION]... SOURCE... DIRECTORY
```

| Option                          | Meaning                                                                  |
|---------------------------------|--------------------------------------------------------------------------|
| `-a`, `--archive`               | Same as `-dR --preserve=all`                                             |
| `-d`                            | Same as `--no-dereference --preserve=links`                              |
| `-f`, `--force`                 | If an existing destination cannot be opened, remove it and try again     |
| `-i`, `--interactive`           | Prompt before overwrite                                                  |
| `-H`                            | Follow command-line symbolic links in SOURCE                             |
| `-l`, `--link`                  | Hard link files instead of copying                                       |
| `-L`, `--dereference`           | Always follow symbolic links in SOURCE                                   |
| `-n`, `--no-clobber`            | Do not overwrite an existing file                                        |
| `-P`, `--no-dereference`        | Never follow symbolic links in SOURCE                                    |
| `-p`                            | Same as `--preserve=mode,ownership,timestamps`                           |
| `--preserve[=LIST]`             | Preserve the listed attributes: mode, ownership, timestamps, links, all  |
| `--no-preserve=LIST`            | Do not preserve the listed attributes                                    |
| `--parents`                     | Use the full source file name under DIRECTORY                            |
| `-r`, `-R`, `--recursive`       | Copy directories recursively                                             |
| `-s`, `--symbolic-link`         | Make symbolic links instead of copying                                   |
| `--strip-trailing-slashes`      | Remove any trailing slashes from each SOURCE                             |
| `-t`, `--target-directory=DIRECTORY` | Copy all SOURCE arguments into DIRECTORY                            |
| `-T`, `--no-target-directory`   | Treat DEST as a normal file                                              |
| `-u`, `--update[=WHEN]`         | Which existing files are updated: `all`, `none`, or `older` (the default for `-u`) |
| `-v`, `--verbose`               | Explain what is being done                                               |
| `-x`, `--one-file-system`       | Stay on this file system                                                 |

- **Behavior.** Directories are copied only with `-r`. A directory is never copied into itself,
  and a source and destination that are the same file are refused. A new file takes the
  source's permission bits filtered by the umask. Symbolic links are followed unless copying
  recursively. Hard links between copied sources are kept when links are preserved. A
  destination written for an earlier operand is not silently replaced by a later one with the
  same base name. `-l` with `-s`, and `-t` with `-T`, are usage errors.
- **Deviations.** Option order is not visible, so `-n` overrides `-i`, and an explicit `-P` or
  `-d` overrides `-H`, which overrides `-L`. `-d`, `-H` and `-p` also answer to the long names
  `--no-dereference-links`, `--dereference-args` and `--preserve-default`. On Windows only the
  read-only attribute and the modification time are carried over. Not implemented: `--backup`
  and `-b`, `--suffix` and `-S`, `--reflink`, `--sparse`, `--attributes-only`,
  `--remove-destination`, `--copy-contents`, `--keep-directory-symlink`, `-Z` and `--context`,
  `--debug`.
- **Verified by.** `core/cp/cp_test.go`, `cmd/cpx/cpx_test.go`.

### FR-54: dirnamex

`dirnamex` must strip the last component from file names, as GNU `dirname` does.

```text
dirnamex [OPTION] NAME...
```

| Option         | Meaning                                        |
|----------------|------------------------------------------------|
| `-z`, `--zero` | End each output line with NUL, not newline     |

- **Behavior.** A name with no slash yields `.`; anything directly under the root yields the
  root. The rest of the path is left as written: no links are resolved and no `.` or `..` is
  collapsed. The filesystem is never touched. No operand is a usage error.
- **Deviations.** None known.
- **Verified by.** `core/dirname/dirname_test.go`, `cmd/dirnamex/dirnamex_test.go`.

### FR-55: echox

`echox` must write its arguments to standard output, as GNU `echo` does.

```text
echox [SHORT-OPTION]... [STRING]...
```

| Option | Meaning                                              |
|--------|------------------------------------------------------|
| `-n`   | Do not output the trailing newline                   |
| `-e`   | Enable interpretation of backslash escapes           |
| `-E`   | Disable interpretation of backslash escapes (default) |

- **Behavior.** Strings are separated by single spaces and followed by a newline. Options are
  recognized only as leading arguments made up solely of those letters (`-n`, `-ne`, `-En`);
  the first argument that is anything else, `--` included, ends option parsing and is printed.
  With `-e` the escapes `\\`, `\a`, `\b`, `\c`, `\e`, `\f`, `\n`, `\r`, `\t`, `\v`, `\0NNN` and
  `\xHH` are interpreted, and `\c` stops all further output.
- **Deviations.** None known. `--help` and `--version` are honored only as the sole argument,
  as in GNU `echo`.
- **Verified by.** `core/echo/echo_test.go`, `cmd/echox/echox_test.go`.

### FR-56: envx

`envx` must print the environment, or run a program in a modified one, as GNU `env` does.

```text
envx [OPTION]... [-] [NAME=VALUE]... [COMMAND [ARG]...]
```

| Option                       | Meaning                                                             |
|------------------------------|---------------------------------------------------------------------|
| `-i`, `--ignore-environment` | Start with an empty environment (a lone `-` means the same)         |
| `-u`, `--unset=NAME`         | Remove variable NAME from the environment; repeatable               |
| `-0`, `--null`               | End each output line with NUL, not newline                          |
| `-C`, `--chdir=DIR`          | Change working directory to DIR                                     |
| `-S`, `--split-string=S`     | Process and split S into separate arguments, for shebang lines      |
| `-a`, `--argv0=ARG`          | Pass ARG as the zeroth argument of COMMAND                          |
| `-v`, `--debug`              | Print verbose information for each processing step                  |

- **Behavior.** Option parsing stops at the first operand, so the command's own options are not
  taken as `envx` options. Printed variables keep their inherited order; assigning an existing
  name replaces it in place. COMMAND is looked up on the `PATH` of the modified environment.
  The exit status is 125 if `envx` itself fails (a usage error included), 126 if COMMAND is
  found but cannot be run, 127 if it cannot be found, and otherwise the status of COMMAND, or
  128 plus the signal number when a signal killed it. `-0`, `-C` and `-a` each have a usage
  rule about whether a command is present.
- **Deviations.** GNU `env` replaces itself with the command. `envx` runs it as a child process
  connected to the same standard streams and exits with its status, so the behavior is the
  same on platforms that cannot replace a running process. Not implemented: `--block-signal`,
  `--default-signal`, `--ignore-signal`, `--list-signal-handling`.
- **Verified by.** `core/env/env_test.go`, `core/env/env_unix_test.go`,
  `cmd/envx/envx_test.go`.

### FR-57: falsex

`falsex` must do nothing, unsuccessfully, as GNU `false` does.

```text
falsex [ignored command line arguments]
```

- **Behavior.** Exits with status 1. Reads no input, writes no output and ignores every
  argument. It has no options of its own.
- **Deviations.** None. `falsex` has no engine package: the adapter is the whole tool.
- **Verified by.** `cmd/falsex/falsex_test.go`.

### FR-58: headx

`headx` must print the first part of each file, as GNU `head` does.

```text
headx [OPTION]... [FILE]...
```

| Option                    | Meaning                                                                        |
|---------------------------|--------------------------------------------------------------------------------|
| `-c`, `--bytes=NUM`       | Print the first NUM bytes; with a leading `-`, all but the last NUM bytes      |
| `-n`, `--lines=NUM`       | Print the first NUM lines instead of 10; with a leading `-`, all but the last NUM |
| `-q`, `--quiet`, `--silent` | Never print headers giving file names                                        |
| `-v`, `--verbose`         | Always print headers giving file names                                         |
| `-z`, `--zero-terminated` | Line delimiter is NUL, not newline                                             |
| `-NUM`                    | The historical spelling of `-n NUM`                                            |

- **Behavior.** The default is 10 lines. NUM takes a multiplier suffix: `b` is 512; `K`, `M`,
  `G`, `T`, `P` and `E` are powers of 1024 (also written `KiB`, `MiB`, and so on); `KB`, `MB`,
  and so on are powers of 1000. With more than one file each is preceded by a
  `==> name <==` header. When `-n`, `-c` and `-NUM` are combined, the last one given wins, as
  in GNU. Reading stops as soon as enough has been seen.
- **Deviations.** None known.
- **Verified by.** `core/head/head_test.go`, `cmd/headx/headx_test.go`.

### FR-59: lnx

`lnx` must make hard and symbolic links, as GNU `ln` does.

```text
lnx [OPTION]... TARGET... [LINK_NAME|DIRECTORY]
```

| Option                             | Meaning                                                                  |
|------------------------------------|--------------------------------------------------------------------------|
| `-f`, `--force`                    | Remove existing destination files                                        |
| `-i`, `--interactive`              | Prompt whether to remove destinations                                    |
| `-L`, `--logical`                  | Dereference TARGETs that are symbolic links                              |
| `-n`, `--no-dereference`           | Treat LINK_NAME as a normal file if it is a symbolic link to a directory |
| `-P`, `--physical`                 | Make hard links directly to symbolic links (default)                     |
| `-r`, `--relative`                 | With `-s`, create links relative to the link's location                  |
| `-s`, `--symbolic`                 | Make symbolic links instead of hard links                                |
| `-t`, `--target-directory=DIRECTORY` | The directory in which to create the links                             |
| `-T`, `--no-target-directory`      | Treat LINK_NAME as a normal file always                                  |
| `-v`, `--verbose`                  | Print the name of each linked file                                       |

- **Behavior.** All four forms of `ln` are accepted. An existing destination is an error unless
  `-f` or `-i` is given. A replacement is atomic: the new link is created under a temporary
  name beside the destination and renamed into place, so the old entry survives a failure. A
  hard link to the same file as its destination is refused. `-r` without `-s`, and `-t` with
  `-T`, are usage errors.
- **Deviations.** Option order is not visible, so `-i` overrides `-f` and `-L` overrides `-P`.
  On Windows, creating symbolic links needs Developer Mode or an elevated prompt. Not
  implemented: `--backup` and `-b`, `--suffix` and `-S`, `-d`, `-F` and `--directory`.
- **Verified by.** `core/ln/ln_test.go`, `cmd/lnx/lnx_test.go`.

### FR-60: lsx

`lsx` must list directory contents, as GNU `ls` does.

```text
lsx [OPTION]... [FILE]...
```

| Option                          | Meaning                                                                 |
|---------------------------------|-------------------------------------------------------------------------|
| `-a`, `--all`                   | Do not ignore entries starting with `.`                                 |
| `-A`, `--almost-all`            | Do not list implied `.` and `..`                                        |
| `-b`, `--escape`                | Print C-style escapes for nongraphic characters                         |
| `-B`, `--ignore-backups`        | Do not list implied entries ending with `~`                             |
| `-c`                            | Use the status change time, for sorting and for display                 |
| `-C`                            | List entries by columns                                                 |
| `--color[=WHEN]`                | Color the output: `always` (when WHEN is omitted), `auto` or `never`    |
| `-d`, `--directory`             | List directories themselves, not their contents                         |
| `-F`, `--classify`              | Append an indicator (one of `*/=@` or a vertical bar) to entries        |
| `--file-type`                   | Likewise, except do not append `*`                                      |
| `--full-time`                   | Like `-l --time-style=full-iso`                                         |
| `-g`                            | Like `-l`, but do not list owner                                        |
| `--group-directories-first`     | Group directories before files                                          |
| `-G`, `--no-group`              | In a long listing, do not print group names                             |
| `-h`, `--human-readable`        | With `-l` and `-s`, print sizes like 1K 234M 2G                         |
| `--si`                          | Likewise, but use powers of 1000, not 1024                              |
| `-H`, `--dereference-command-line` | Follow symbolic links listed on the command line                     |
| `--hide=PATTERN`                | Do not list implied entries matching PATTERN (overridden by `-a`, `-A`) |
| `-i`, `--inode`                 | Print the index number of each file                                     |
| `-I`, `--ignore=PATTERN`        | Do not list implied entries matching PATTERN                            |
| `-l`                            | Use a long listing format                                               |
| `-L`, `--dereference`           | Show information for the file a symbolic link references                |
| `-m`                            | Fill width with a comma-separated list of entries                       |
| `-n`, `--numeric-uid-gid`       | Like `-l`, but list numeric user and group IDs                          |
| `-o`                            | Like `-l`, but do not list group information                            |
| `-p`                            | Append `/` to directories                                               |
| `-q`, `--hide-control-chars`    | Print `?` instead of nongraphic characters                              |
| `-Q`, `--quote-name`            | Enclose entry names in double quotes                                    |
| `-r`, `--reverse`               | Reverse order while sorting                                             |
| `-R`, `--recursive`             | List subdirectories recursively                                         |
| `-s`, `--size`                  | Print the allocated size of each file, in blocks                        |
| `-S`                            | Sort by file size, largest first                                        |
| `--sort=WORD`                   | Sort by WORD: none, size, time, version, extension, name, width         |
| `--time=WORD`                   | Which timestamp: atime, access, use; ctime, status; mtime, modification; birth, creation |
| `--time-style=TIME_STYLE`       | Time format with `-l`: full-iso, long-iso, iso, locale, or `+FORMAT`    |
| `-t`                            | Sort by time, newest first                                              |
| `-T`, `--tabsize=COLS`          | Assume tab stops at each COLS instead of 8                              |
| `-u`                            | Use the access time, for sorting and for display                        |
| `-U`                            | Do not sort; list entries in directory order                            |
| `-v`                            | Natural sort of version numbers within text                             |
| `-w`, `--width=COLS`            | Set output width to COLS; 0 means no limit                              |
| `-x`                            | List entries by lines instead of by columns                             |
| `-X`                            | Sort alphabetically by entry extension                                  |
| `-1`                            | List one file per line                                                  |

- **Behavior.** Operands that are not directories are listed first, then each directory under a
  header. On a terminal names are arranged in columns sized to the window; when output is piped
  or redirected there is one name per line. In the long format, timestamps from the last six
  months show the time of day and older ones the year. Entries that cannot be examined are
  still listed, with `?` in place of the unknown fields. When several options of one family are
  given (the formats, the sort keys, `-a` and `-A`, the time selectors) the last one on the
  command line wins, as in GNU. The exit status is 0 on success, 1 for minor problems, and 2
  for serious trouble such as an operand that cannot be accessed or a bad command line.
- **Deviations.** Names are sorted by plain byte comparison (the C locale), whatever the
  environment's locale. On Windows, which has no POSIX owner, group, inode or block count, those
  columns hold placeholders. The single-letter options `-c`, `-C`, `-g`, `-l`, `-m`, `-o`,
  `-p`, `-S`, `-t`, `-u`, `-U`, `-v`, `-x`, `-X` and `-1` also answer to long names of gnux's
  own (`--ctime`, `--columns`, `--long-no-owner`, `--long`, `--commas`, `--long-no-group`,
  `--slash`, `--sort-size`, `--sort-time`, `--atime`, `--unsorted`, `--sort-version`,
  `--across`, `--sort-extension`, `--one-per-line`). Not implemented: `--author`,
  `--block-size`, `-D` and `--dired`, `-f`, `--format`, `--hyperlink`, `--indicator-style`,
  `-k` and `--kibibytes`, `-N` and `--literal`, `--quoting-style`, `--show-control-chars`,
  `-Z` and `--context`, `--zero`.
- **Verified by.** `core/ls/ls_test.go`, `cmd/lsx/lsx_test.go`.

### FR-61: mkdirx

`mkdirx` must create directories, as GNU `mkdir` does.

```text
mkdirx [OPTION]... DIRECTORY...
```

| Option            | Meaning                                                       |
|-------------------|---------------------------------------------------------------|
| `-m`, `--mode=MODE` | Set the file mode (as in chmod), not a=rwx minus the umask  |
| `-p`, `--parents` | No error if existing; make parent directories as needed       |
| `-v`, `--verbose` | Print a message for each created directory                    |

- **Behavior.** Without `-p` the parent must exist and the directory must not. MODE is octal
  (`755`) or symbolic (`u=rwx,go=rx`). A mode given with `-m` is applied exactly, without the
  umask, and only to the operand: parents made by `-p` keep the default mode, plus owner write
  and search so that the next level can be created inside them. An invalid mode is a usage
  error.
- **Deviations.** On Windows only the read-only attribute is representable, so most of a mode
  has no effect there. Not implemented: `-Z` and `--context`.
- **Verified by.** `core/mkdir/mkdir_test.go`, `core/mkdir/mkdir_unix_test.go`,
  `core/mkdir/mode_test.go`, `cmd/mkdirx/mkdirx_test.go`.

### FR-62: mvx

`mvx` must move and rename files, as GNU `mv` does.

```text
mvx [OPTION]... SOURCE... DIRECTORY
```

| Option                             | Meaning                                                                 |
|------------------------------------|-------------------------------------------------------------------------|
| `-f`, `--force`                    | Do not prompt before overwriting                                        |
| `-i`, `--interactive`              | Prompt before overwrite                                                 |
| `-n`, `--no-clobber`               | Do not overwrite an existing file                                       |
| `--strip-trailing-slashes`         | Remove any trailing slashes from each SOURCE                            |
| `-t`, `--target-directory=DIRECTORY` | Move all SOURCE arguments into DIRECTORY                              |
| `-T`, `--no-target-directory`      | Treat DEST as a normal file                                             |
| `-u`, `--update[=WHEN]`            | Which existing files are updated: `all`, `none`, or `older` (the default for `-u`) |
| `-v`, `--verbose`                  | Explain what is being done                                              |

- **Behavior.** Within one filesystem a move is an atomic rename. Across filesystems `mvx`
  copies the source with its attributes, symbolic links and hard links intact, and removes the
  original only after the copy has fully succeeded (FR-16). A directory is never moved into
  itself, and a directory only replaces an empty directory. A symbolic link is replaced, never
  its target. On a terminal, replacing a write-protected destination asks for confirmation
  unless `-f` is given.
- **Deviations.** Option order is not visible, so `-n` overrides `-i`, which overrides `-f`.
  Not implemented: `--backup` and `-b`, `--suffix` and `-S`, `-Z` and `--context`, `--no-copy`,
  `--exchange`, `--debug`.
- **Verified by.** `core/mv/mv_test.go`, `cmd/mvx/mvx_test.go`.

### FR-63: pwdx

`pwdx` must print the current working directory, as GNU `pwd` does.

```text
pwdx [OPTION]...
```

| Option             | Meaning                                                         |
|--------------------|-----------------------------------------------------------------|
| `-L`, `--logical`  | Use `PWD` from the environment, even if it contains symlinks    |
| `-P`, `--physical` | Resolve all symlinks (default)                                  |

- **Behavior.** With `-L` the value of `PWD` is printed only when it names the current
  directory.
- **Deviations.** When both are given, GNU lets the last one win; `pwdx` prefers `-L`.
- **Verified by.** `core/pwd/pwd_test.go`, `cmd/pwdx/pwdx_test.go`.

### FR-64: realpathx

`realpathx` must print the resolved absolute file name, as GNU `realpath` does.

```text
realpathx [OPTION]... FILE...
```

| Option                          | Meaning                                                  |
|---------------------------------|----------------------------------------------------------|
| `-e`, `--canonicalize-existing` | All components of the path must exist                    |
| `-m`, `--canonicalize-missing`  | No path components need exist or be a directory          |
| `-L`, `--logical`               | Resolve `..` components before symlinks                  |
| `-P`, `--physical`              | Resolve symlinks as encountered (default)                |
| `-q`, `--quiet`                 | Suppress most error messages                             |
| `--relative-to=DIR`             | Print the resolved path relative to DIR                  |
| `--relative-base=DIR`           | Print absolute paths unless paths are below DIR          |
| `-s`, `--strip`, `--no-symlinks` | Do not expand symlinks                                  |
| `-z`, `--zero`                  | End each output line with NUL, not newline               |

- **Behavior.** By default all components but the last must exist. The files themselves are
  never opened. A name that cannot be resolved is reported (unless `-q`), the exit status
  reflects it, and the remaining names are still processed. A symbolic link loop is an error,
  not a hang.
- **Deviations.** When both `-e` and `-m` are given, `-e` wins. Not implemented: `-E` and
  `--canonicalize` (which is the default behavior).
- **Verified by.** `core/realpath/realpath_test.go`, `cmd/realpathx/realpathx_test.go`.

### FR-65: rmdirx

`rmdirx` must remove empty directories, as GNU `rmdir` does.

```text
rmdirx [OPTION]... DIRECTORY...
```

| Option                       | Meaning                                                              |
|------------------------------|----------------------------------------------------------------------|
| `--ignore-fail-on-non-empty` | Ignore each failure that is solely because a directory is non-empty  |
| `-p`, `--parents`            | Remove DIRECTORY and its ancestors                                   |
| `-v`, `--verbose`            | Output a diagnostic for every directory processed                    |

- **Behavior.** A directory that still has entries is reported and left untouched, and so is
  anything that is not a directory. A symbolic link to a directory is not a directory: it is
  refused rather than followed. With `-p`, removal stops at the first ancestor that cannot be
  removed.
- **Deviations.** None known.
- **Verified by.** `core/rmdir/rmdir_test.go`, `cmd/rmdirx/rmdirx_test.go`.

### FR-66: rmx

`rmx` must remove files and directories, as GNU `rm` does.

```text
rmx [OPTION]... [FILE]...
```

| Option                 | Meaning                                                                          |
|------------------------|----------------------------------------------------------------------------------|
| `-f`, `--force`        | Ignore nonexistent files and arguments, never prompt                             |
| `-i`                   | Prompt before every removal                                                      |
| `-I`                   | Prompt once before removing more than three files, or when removing recursively  |
| `--interactive[=WHEN]` | Prompt according to WHEN: `never` (or `no`, `none`), `once`, or `always` (or `yes`; the default when omitted) |
| `-r`, `-R`, `--recursive` | Remove directories and their contents recursively                             |
| `-d`, `--dir`          | Remove empty directories                                                         |
| `-v`, `--verbose`      | Explain what is being done                                                       |
| `--one-file-system`    | When removing recursively, skip directories on a different file system           |
| `--preserve-root`      | Do not remove `/` (default)                                                      |
| `--no-preserve-root`   | Do not treat `/` specially                                                       |

- **Behavior.** Directories are not removed without `-r` or `-d`. `.` and `..` are refused, and
  so is recursion on the filesystem root unless overridden (FR-12). Symbolic links are removed,
  never followed. Recursive removal is post-order: with `-i`, `rmx` asks to descend, then about
  each child, then about the directory. On a terminal, with neither `-f` nor `-i`, a
  write-protected entry triggers a confirmation prompt. No operand is a usage error unless `-f`
  is given. Independent arguments are removed concurrently whenever no prompt or ordered output
  is involved.
- **Deviations.** Option order is not visible, so `-i` overrides `--interactive=WHEN`, which
  overrides `-I`, which overrides `-f`; `-f` still suppresses the error for a missing file
  whatever the final prompt mode. `-i` and `-I` also answer to the long names `--prompt-each`
  and `--prompt-once`. Off Unix, `--one-file-system` has no effect, because no device number is
  available. Not implemented: `--preserve-root=all`.
- **Verified by.** `core/rm/rm_test.go`, `cmd/rmx/rmx_test.go`.

### FR-67: sleepx

`sleepx` must pause for a given time, as GNU `sleep` does.

```text
sleepx NUMBER[SUFFIX]...
```

- **Behavior.** It has no options of its own. SUFFIX is `s` (the default), `m`, `h` or `d`.
  NUMBER may be a fraction, an exponent or a hexadecimal floating-point value, or `infinity`
  (or `inf`) to pause until interrupted. Several arguments pause for the sum of their values.
  Every operand is checked before the pause begins, so an invalid interval fails immediately.
  No operand, or an invalid one, is a usage error.
- **Deviations.** None known.
- **Verified by.** `core/sleep/sleep_test.go`, `cmd/sleepx/sleepx_test.go`.

### FR-68: tailx

`tailx` must print the last part of each file, optionally following it, as GNU `tail` does.

```text
tailx [OPTION]... [FILE]...
```

| Option                      | Meaning                                                                        |
|-----------------------------|--------------------------------------------------------------------------------|
| `-c`, `--bytes=NUM`         | Output the last NUM bytes; `+NUM` starts output at byte NUM                    |
| `-f`, `--follow[=HOW]`      | Output appended data as the file grows; HOW is `name` or `descriptor` (default) |
| `-F`                        | Same as `--follow=name --retry`                                                |
| `-n`, `--lines=NUM`         | Output the last NUM lines instead of 10; `+NUM` starts output at line NUM      |
| `--pid=PID`                 | With `-f`, terminate after process PID dies                                    |
| `-q`, `--quiet`, `--silent` | Never output headers giving file names                                         |
| `--retry`                   | Keep trying to open a file if it is inaccessible                               |
| `-s`, `--sleep-interval=N`  | With `-f`, sleep for approximately N seconds (default 1.0) between iterations  |
| `-v`, `--verbose`           | Always output headers giving file names                                        |
| `-z`, `--zero-terminated`   | Line delimiter is NUL, not newline                                             |
| `-NUM`                      | The historical spelling of `-n NUM`                                            |

- **Behavior.** The default is 10 lines. NUM takes the same multiplier suffixes as `headx`.
  `--follow=name` tracks the file name rather than the open file, so it survives log rotation.
  When `-n`, `-c` and `-NUM` are combined, the last one given wins, as in GNU. On a regular
  file `tailx` seeks to the end and reads backward, so its cost does not depend on the file's
  size. A negative sleep interval or PID, or an unknown `--follow` value, is a usage error.
- **Deviations.** Following is done by polling at the sleep interval. `-F` also answers to the
  long name `--follow-retry`. Not implemented: `--max-unchanged-stats`.
- **Verified by.** `core/tail/tail_test.go`, `cmd/tailx/tailx_test.go`.

### FR-69: touchx

`touchx` must change file timestamps, creating files as needed, as GNU `touch` does.

```text
touchx [OPTION]... FILE...
```

| Option                   | Meaning                                                                   |
|--------------------------|---------------------------------------------------------------------------|
| `-a`                     | Change only the access time                                               |
| `-c`, `--no-create`      | Do not create any files                                                   |
| `-d`, `--date=STRING`    | Parse STRING and use it instead of the current time                       |
| `-h`, `--no-dereference` | Affect each symbolic link instead of any referenced file                  |
| `-m`                     | Change only the modification time                                         |
| `-r`, `--reference=FILE` | Use this file's times instead of the current time                         |
| `-t`                     | Use a `[[CC]YY]MMDDhhmm[.ss]` stamp instead of the current time           |
| `--time=WORD`            | Which time to change: access, atime or use (like `-a`); modify or mtime (like `-m`) |

- **Behavior.** An existing file's contents are never changed. `-a` and `-m` leave the other
  timestamp exactly as it was. `-d` accepts a date (`2024-01-02`), a date and time, an RFC 3339
  timestamp, `@SECONDS` since the epoch, or `now`. Giving times from more than one source is a
  usage error.
- **Deviations.** `-d` accepts the forms listed above, not the whole of GNU's free-form date
  syntax. Changing the times of a symbolic link itself (`-h`) is only possible on Linux;
  elsewhere `touchx` reports the link rather than silently touching its target. `-a`, `-m` and
  `-t` also answer to the long names `--access-time`, `--modification-time` and `--stamp`. Not
  implemented: `-f` (which GNU accepts and ignores).
- **Verified by.** `core/touch/touch_test.go`, `cmd/touchx/touchx_test.go`.

### FR-70: truex

`truex` must do nothing, successfully, as GNU `true` does.

```text
truex [ignored command line arguments]
```

- **Behavior.** Exits with status 0. Reads no input, writes no output and ignores every
  argument. It has no options of its own.
- **Deviations.** None. `truex` has no engine package: the adapter is the whole tool.
- **Verified by.** `cmd/truex/truex_test.go`.

### FR-71: wcx

`wcx` must print line, word and byte counts, as GNU `wc` does.

```text
wcx [OPTION]... [FILE]...
```

| Option                    | Meaning                                                                      |
|---------------------------|------------------------------------------------------------------------------|
| `-c`, `--bytes`           | Print the byte counts                                                        |
| `-m`, `--chars`           | Print the character counts                                                   |
| `-l`, `--lines`           | Print the newline counts                                                     |
| `--files0-from=F`         | Read input from the files named, NUL-terminated, in file F; `-` is standard input |
| `-L`, `--max-line-length` | Print the maximum display width                                              |
| `-w`, `--words`           | Print the word counts                                                        |
| `--total=WHEN`            | When to print a line with total counts: auto, always, only, never            |

- **Behavior.** With no option the counts are lines, words and bytes. Whatever the order of the
  options, counts are printed in the order lines, words, characters, bytes, maximum line
  length. Input is decoded as UTF-8; bytes that are not valid UTF-8 count toward words and
  bytes but not characters. Columns are right-aligned to a width derived from the files' sizes.
  File operands cannot be combined with `--files0-from`. A read error keeps the partial counts.
- **Deviations.** None known in the options listed. Not implemented: `--debug`.
- **Verified by.** `core/wc/wc_test.go`, `cmd/wcx/wcx_test.go`.

### FR-72: yesx

`yesx` must print a string repeatedly, as GNU `yes` does.

```text
yesx [STRING]...
```

- **Behavior.** It has no options of its own. It writes a line made of every STRING joined by
  single spaces, or `y` when none is given, until it is killed or its output is closed. When
  the reader goes away (`yesx | headx`) it stops quietly; any other write failure is reported
  and exits with status 1. A first `--` is taken as the end of options and not printed.
- **Deviations.** None known.
- **Verified by.** `core/yes/yes_test.go`, `cmd/yesx/yesx_test.go`.

## 3. Destructive operations

### FR-12: refusals in rmx

`rmx` must refuse to remove `.` and `..`, and must refuse to recurse on the filesystem root
unless explicitly overridden.

- **Rationale.** These are the two commands whose damage cannot be undone and is almost never
  intended.
- **Acceptance.** `rmx -rf .` reports `refusing to remove '.' or '..' directory` and removes
  nothing. `rmx -rf /` reports that it is dangerous to operate recursively on the root and
  names `--no-preserve-root` as the override. The failsafe is on in the zero value of
  `rm.Options`.
- **Verified by.** `TestRefusesDotDirectory` in `core/rm/rm_test.go`; `cmd/rmx/rmx_test.go` for
  the option mapping. The root refusal itself is not exercised by a test, for the obvious
  reason; it is checked by review of `core/rm/rm.go`.

### FR-13: prompting

`rmx`, `cpx`, `mvx` and `lnx` must prompt before overwriting or removing when asked to (`-i`),
and `rmx` and `mvx` must not prompt when `-f` is given without `-i`.

- **Rationale.** `-i` is the user asking for a last chance. `-f` is the user declining one.
- **Acceptance.** With `-i`, each tool asks its question on standard error and acts only on an
  answer beginning with `y` or `Y`; end of input counts as no. With `-f` alone, `rmx` and `mvx`
  ask nothing. When both are given, FR-9 applies and the tool prompts.
- **Verified by.** `TestInteractivePrompt` and `TestPromptOnceDecline` (`core/rm`),
  `TestInteractiveReadsAnswerFromInput` (`core/cp`), `TestOverwritePolicies` (`core/mv`),
  `TestInteractive` (`core/ln`).

### FR-14: self-copies and same files

`cpx` and `mvx` must refuse to copy or move a directory into itself, and `cpx`, `mvx` and `lnx`
must detect that source and destination are the same file.

- **Rationale.** A directory copied into itself recurses until the disk is full. A file copied
  onto itself is truncated before it is read.
- **Acceptance.** Each case is reported with GNU's message and nothing is changed.
- **Verified by.** `TestDiagnostics` in `core/mv/mv_test.go` (same file, and a directory into
  itself); `TestForceHardLinkRefusesSameFile` in `core/ln/ln_test.go`. The two `cpx` refusals
  are implemented in `core/cp/cp.go` and have no test of their own: they are checked by review.

### FR-15: symbolic links

Removing and moving must operate on a symbolic link itself, never on its target. Every other
tool must follow GNU's default for that tool, and the options that change it.

- **Rationale.** Following a link during a removal deletes something the user did not name.
  Elsewhere GNU's defaults differ by tool, and gnux matches each rather than imposing one.
- **Acceptance.** `rmx` removes a link and leaves its target. `mvx` replaces a link, not what
  it points at. `rmdirx` refuses a link to a directory. `cpx` follows links unless copying
  recursively, with `-P`, `-d`, `-H` and `-L` changing that. `touchx` follows links unless
  `-h` is given. `lsx` shows the link itself unless `-L` or `-H` says otherwise.
- **Verified by.** `TestMoveReplacesSymlinkNotItsTarget` (`core/mv`),
  `TestSymlinkToDirectoryIsNotADirectory` (`core/rmdir`), `TestSymlinkPolicies` (`core/cp`),
  `TestFollowsSymlinkByDefault` and `TestNoDereference` (`core/touch`), `TestSymlinks`
  (`core/ls`).

### FR-16: cross-filesystem moves

A move across filesystems must remove the source only after the copy has completely succeeded.

- **Rationale.** A failed copy followed by a removal is data loss.
- **Acceptance.** When the rename fails because source and destination are on different
  devices, `mvx` copies, and removes the source only if every part of the copy succeeded. A
  failed copy leaves the source intact and is reported.
- **Verified by.** `TestCrossDeviceFile`, `TestCrossDeviceDirectory`,
  `TestCrossDeviceKeepsSourceWhenCopyFails` and
  `TestCrossDeviceNeverMergesIntoNonEmptyDirectory` in `core/mv/mv_test.go`, which simulate a
  second filesystem through the injected `RenameFn`.

## 4. Invocation

### FR-17: one binary

All tools must ship in one binary, `gnux`.

- **Rationale.** BR-3. One file to install, one file to copy into an image.
- **Verified by.** The Build workflow builds exactly one binary per platform; `main.go` is the
  only command package other than the generator and the WebAssembly entry point.

### FR-18: subcommand form

A tool must be runnable as a subcommand: `gnux rmx ...`.

- **Verified by.** `TestDispatch` ("subcommand") in `cmd/main_test.go`.

### FR-19: own-name form

A tool must be runnable under its own name: when the binary is invoked as `rmx`, it must behave
as `rmx`.

- **Rationale.** This is what makes a link or a copy of the binary a command.
- **Verified by.** `TestDispatch` ("argv0"); the multi-call smoke test in the Test workflow,
  which copies the built binary to `pwdx` and runs it on every operating system.

### FR-20: GNU names

Both forms must also accept the GNU name (`gnux rm ...`, a link called `rm`).

- **Rationale.** It lets scripts written for the GNU names run unchanged where GNU coreutils is
  not installed.
- **Verified by.** `TestDispatch` ("subcommand by GNU name", "argv0 by GNU name").

### FR-21: listing the tools

`gnux --list` must print every tool name, one per line, for use by installers.

- **Acceptance.** The output has exactly one name per registered tool. `gnux shell` is not in
  it.
- **Verified by.** `TestRootListAndVersion` in `cmd/main_test.go`.

### FR-22: root help

`gnux --help` must list every tool with a one-line summary.

- **Acceptance.** `gnux --help`, `gnux help` and `gnux` with no arguments print the same
  overview: every tool with its GNU name and summary, the gnux commands, and how to read a manual.
- **Verified by.** `TestHelpOverviewListsEveryCommand` in `cmd/help_test.go`.

### FR-87: a manual for every tool and every option

`gnux help TOOL` must print the tool's full manual, and `gnux help TOOL OPTION` must explain any
one of its options, for every tool and every option, from the same metadata the reference is
generated from.

- **Rationale.** `--help` stays short, as GNU's does; the depth of the reference must still be
  reachable offline, in the terminal, where the tool is used.
- **Acceptance.**
  - `gnux help TOOL` accepts either name (`rmx`, `rm`) and `shell`, and prints NAME, SYNOPSIS,
    DESCRIPTION, OPTIONS, PERMISSIONS, USE CASES, EXAMPLES (every example, bash and PowerShell),
    OPTIONS IN DEPTH (when the tool has per-flag topics), EXIT STATUS and SEE ALSO.
  - `gnux help TOOL OPTION` accepts `--force`, `force`, `-f` or `f`. A flag with its own topic
    (`rmx`) prints that topic in full; any other prints its usage, default and every example
    that passes it to the tool.
  - An unknown tool or option is a usage error with status 1, prefixed `gnux help:`.
  - Color is used only on a terminal, never when `NO_COLOR` is set or `TERM` is `dumb`.
  - Exit statuses and related tools come from `docmeta.Meta.ExitStatus` and `SeeAlso`
    (defaulting to the GNU convention) and appear on the generated reference pages too.
- **Verified by.** `TestHelpManualForEveryCommand`, `TestHelpForEveryOption`,
  `TestHelpTopicsAreShownInFull`, `TestHelpErrors`, `TestHelpHelp` in `cmd/help_test.go`;
  `TestUsesFlag`, `TestFlagLabel`, `TestColorOnlyWhenAsked` in `internal/tool/help_test.go`.

### FR-75: unknown tools

`gnux NAME` must be a usage error when NAME is not a tool or a gnux command.

- **Acceptance.** It prints `gnux: unknown tool 'NAME'` and the hint, and exits 1.
- **Verified by.** `TestDiagnostics` ("unknown tool") in `cmd/main_test.go`.

### FR-76: recognizing the program name

The name gnux was started under must be reduced to a bare tool name before it is looked up.

- **Rationale.** `C:\tools\RMX.EXE` and `/usr/local/bin/rmx` are both `rmx`.
- **Acceptance.** The directory is stripped (either separator, on every platform), a trailing
  `.exe` is removed whatever its case, and the result is lower-cased.
- **Verified by.** `TestProgName` in `cmd/main_test.go`.

## 5. The portable shell

### FR-23: sources of commands

`gnux shell` must run commands interactively, from a script file, from standard input, or from
a string (`-c`).

- **Acceptance.** With no operand it reads standard input, prompting with `gnux> ` when that is
  a terminal. With one operand it runs that file. With `-c STRING` it runs the string. A second
  operand, or an operand with `-c`, is a usage error with status 2. A script that cannot be
  opened is reported and exits 127.
- **Verified by.** `TestShellCommand` in `cmd/shell_test.go`.

### FR-24: the language

The shell must support pipelines, `&&`, `||`, `;`, input and output redirection (`<`, `>`,
`>>`), `NAME=value` variables with `$NAME`, `${NAME}` and `$?`, single and double quotes,
backslash escapes, globbing (`*`, `?`, `[...]`), `~` expansion and `#` comments.

- **Rationale.** This is the subset a build, clean or packaging script actually uses.
- **Acceptance.** Each construct behaves as in POSIX sh, within the differences of FR-78. A
  pattern that matches nothing is passed through unchanged. Matches are sorted.
- **Verified by.** `TestScripts`, `TestRedirection` and `TestGlob` in
  `core/shell/shell_test.go`.

### FR-25: in-process, no system shell

The shell must run tools in-process and must not depend on any system shell or utility.

- **Rationale.** BR-5.
- **Acceptance.** `core/shell` reaches tools only through its `Exec` function. It imports no
  process-spawning package.
- **Verified by.** `core/shell/shell_test.go`, which runs the interpreter against a fake
  command set; review of the package's imports.

### FR-26: builtins

The shell must provide `cd`, `exit` and `help` as builtins.

- **Acceptance.** `cd` with no argument goes to the home directory; `exit` takes an optional
  status, masked to 0 to 255; `help` prints the language summary.
- **Verified by.** `TestCd` and `TestSetAndHelp` in `core/shell/shell_test.go`.

### FR-27: exit status

The shell's exit status must be that of the last command; 2 for a syntax error; 127 for an
unknown command.

- **Acceptance.** A syntax error aborts a non-interactive script. An interactive session
  reports it and carries on.
- **Verified by.** `TestSyntaxErrors` and `TestDiagnostics` in `core/shell/shell_test.go`;
  `TestShellCommand` in `cmd/shell_test.go`.

### FR-28: errexit

With `-e`, the shell must stop at the first failing command whose status is not tested by `&&`
or `||`.

- **Verified by.** `TestErrExit` in `core/shell/shell_test.go`.

### FR-29: the same result everywhere

A script must produce the same result on every supported platform, given the same files.

- **Acceptance.** A backslash escapes only a special character, so a Windows path such as
  `C:\tools\bin` survives unquoted. Redirection creates files with mode 0644 before the umask.
- **Verified by.** The shell suites in the Test workflow on all three operating systems.

### FR-77: a command, not a tool

`gnux shell` must be a gnux command and not a tool.

- **Rationale.** It mirrors no GNU program, so it has no `x` name and must not be reachable as
  one.
- **Acceptance.** It is not in `gnux --list`, cannot be selected through the program name, and
  its diagnostics from the command line are prefixed `gnux shell`.
- **Verified by.** `TestShellCommand` ("shell is not a tool", "usage error names gnux shell").

### FR-78: deliberate differences from sh

The shell's differences from POSIX sh must be few, deliberate and documented.

- **Acceptance.** A variable's value is never re-split or re-globbed, so a path with spaces
  needs no quoting once stored. `NAME=value cmd` sets the variable for the rest of the script,
  not only for `cmd`. A variable is looked up among the shell's own variables first, then in
  the environment. There are no loops, functions, conditionals beyond `&&` and `||`, command
  substitution or here-documents. External programs are not looked up: they are reached through
  `envx PROGRAM ARG...`.
- **Verified by.** `TestScripts` and `TestGlob` in `core/shell/shell_test.go`; review of the
  shell's reference page for the documentation half.

### FR-79: pipelines

The stages of a pipeline must run concurrently, with every redirection opened before any stage
starts, and a stage must not be able to change the shell's own state.

- **Rationale.** `lsx | wcx -l > count` must count `count`, as a POSIX shell guarantees.
  `yesx | headx` must end when `headx` does.
- **Acceptance.** Expansion, assignments and redirections happen in order before the stages
  run. A stage whose reader has gone sees a closed-pipe error and stops quietly. `cd` and
  `exit` inside a stage do not affect the shell. The pipeline's status is that of its last
  stage.
- **Verified by.** `TestScripts` and `TestRedirection` in `core/shell/shell_test.go`;
  `TestRunRepeatsUntilThePipeCloses` in `core/yes/yes_test.go`.

## 6. The Go library

### FR-30: importable engines

Each tool's behavior must be available as an importable engine package that does not depend on
the command-line framework.

- **Acceptance.** `core/<gnu>` exposes `Name`, `Options` where the tool has any, a constructor
  and `Run`. Nothing under `core/` imports cobra or pflag. `truex` and `falsex` are the stated
  exception: they have no engine.
- **Verified by.** The engine suites, which import only `core` and the engine. There is no
  automated check of the import rule; it is checked by review.

### FR-31: the zero value

The zero value of an engine's options must be the tool's default behavior, with every safeguard
on.

- **Acceptance.** `rm.Options{}` is plain `rm`: no recursion, default prompting, the root
  failsafe on. A field whose default is "on" is named for the negative (`NoPreserveRoot`).
- **Verified by.** The engine suites construct options from the zero value throughout;
  `core/rm/example_test.go`.

### FR-32: streams only

Engines must read and write only through the streams they are given.

- **Acceptance.** No engine refers to the process's standard streams. `core.Stdio()` is the one
  place they are named.
- **Verified by.** The engine suites capture all output in buffers and assert on it exactly. A
  write that bypassed the streams would not be caught by them; review covers that.

### FR-33: one error convention

Engines must report outcomes through one error convention: success, failed-and-reported, usage
error, or explicit exit status.

- **Acceptance.** `nil`; `core.ErrFailed`; `*core.UsageError`; `*core.ExitError`.
- **Verified by.** The engine suites; `TestDiagnostics` in `cmd/main_test.go` for the mapping
  to exit statuses.

### FR-34: injection points

Anything that needs a terminal, a clock or the environment must be replaceable by the caller.

- **Acceptance.** Exported fields such as `PromptFn`, `RemoveFn`, `RenameFn`, `Now`, `Getenv`,
  `Getwd`, `Environ`, `Umask` and `Sleep` exist where the engine needs them.
- **Verified by.** The engine suites use them: `TestInjectedRemoveFn` (`core/rm`),
  `TestDefaultClock` (`core/touch`), `TestRunSleepsForTheTotal` (`core/sleep`), and others.

### FR-35: tools and scripts by name

A program must be able to run a tool by name, and a whole shell script, without starting a
process.

- **Acceptance.** `cmd.RunTool(name, args, streams)` returns the exit status and whether the
  tool exists. `cmd.NewShell(opts, streams)` returns a shell whose commands are the registered
  tools.
- **Verified by.** `TestRunTool` and `TestNewShell` in `cmd/shell_test.go`.

### FR-80: embedding the whole command line

A program must be able to run the entire gnux command line and receive the exit status instead
of exiting.

- **Acceptance.** `cmd.Main(argv, streams)` returns the status. `main.go` is the only caller of
  `os.Exit`.
- **Verified by.** Every test in `cmd/main_test.go` runs through `Main`.

### FR-81: a session for interactive hosts

The shell must offer a way to run one line at a time while keeping its state, and a way to
replace its command set.

- **Rationale.** A REPL, and the playground, drive the shell line by line. An application may
  want to restrict a script to a few tools or add commands of its own.
- **Acceptance.** `Shell.Eval(line)` returns the line's status and keeps variables, `$?` and
  the working directory for the next call. `Shell.Run` restores the process's working
  directory when it returns. `shell.New` accepts any `ExecFunc`.
- **Verified by.** `TestEvalKeepsState` and `TestCd` in `core/shell/shell_test.go`.

## 7. Installation and distribution

### FR-36: ways to install

gnux must be installable with `go install`, from source with `make`, by script, and as a
container image.

- **Acceptance.** `go install github.com/braswelljr/gnux@latest`; `make build` and
  `make install`; `./install.sh` with `--from-source` and `--no-links`; `docker build`.
- **Acceptance (archives).** A pushed `v*` tag publishes a GitHub release carrying one archive
  per supported platform and a `checksums.txt` of their SHA-256 sums.
- **Verified by.** The Build workflow covers compilation for every platform, and the Release
  workflow builds and publishes the archives. The install
  script, the Makefile's install targets and the Docker image are not exercised by CI; they are
  checked by review.

### FR-37: per-tool commands

Installation must be able to create the per-tool commands automatically.

- **Acceptance.** `make links`, `make install/links`, `install.sh` and the Dockerfile each link
  every name printed by `gnux --list` to the binary.
- **Verified by.** The smoke test in the Test workflow proves dispatch on the program name with
  a copy of the binary. The link-creating loops themselves are checked by review.

### FR-38: the version

The binary must report the version it was built from.

- **Acceptance.** `gnux --version` prints `gnux VERSION`, and `TOOL --version` prints
  `TOOL (gnux) VERSION`. The version is injected at link time by `make build`, `make install`,
  `install.sh --from-source`, the Dockerfile, the Build workflow and the Release workflow.
  Without link flags, as with a plain `go install github.com/braswelljr/gnux@v1.2.3`, the
  version falls back to the module version Go records in the build info (`v1.2.3`, or a
  pseudo-version for a build from a checkout), and to `dev` only when there is none.
- **Verified by.** `TestRootListAndVersion` and `TestEveryToolAnswersHelpAndVersion` check that
  a version line is printed; `TestBuildInfoVersion` (`internal/common`) checks the fallback.

## 8. Documentation

### FR-39: a reference page per tool

Every tool must have a reference page covering its synopsis, description, permissions, use
cases, options and examples.

- **Verified by.** `TestRegistryIsConsistent` (a summary, a description and at least one example
  are required to register); `TestGenerateAll` (a page per tool in every format).

### FR-40: examples for two shells

Examples must be given for both a POSIX shell and PowerShell.

- **Acceptance.** Each example carries a `bash` form, and a `powershell` form where it differs
  or adds something.
- **Verified by.** Review. The PowerShell form may be empty by design, so nothing fails when
  one is missing.

### FR-41: generated from the program's own definitions

The reference must be generated from the same definitions the program uses, so it cannot drift
from the code.

- **Verified by.** The Docs workflow; `make docs/check`.

### FR-42: formats

The reference must be published as markdown, man pages, YAML and reStructuredText, and as
website content.

- **Verified by.** `TestGenerateAll`, `TestMarkdownLinksResolve` and `TestGenerateMDX` in
  `cmd/docs/docs_test.go`.

### FR-43: stated requirements

The documentation must state the requirements for running, installing, embedding and
contributing.

- **Acceptance.** The README's Requirements section, the guide's Requirements section and
  `AGENTS.md` agree with `go.mod` and the build matrix.
- **Verified by.** Review. Nothing checks the three against each other.

### FR-44: described workflows

The documentation must describe the common workflows: everyday use, scripting, embedding and
contributing.

- **Verified by.** Review of the guide, the README, [srs-workflows](srs-workflows.md) and
  [workflows](../workflows/README.md).

### FR-82: the hand-written guides are published

The hand-written usage guides must be published on the website with the generated reference.
The requirements, workflows and rules are for contributors and stay in the repository; they are
not published.

- **Acceptance.** Each markdown file in `documentation/guides` becomes a page in a section of the same
  name, and nothing from `docs/requirements`, `docs/workflows` or `docs/rules` appears under
  `web/content/docs`; `README.md` is the section's index. The page
  title is the file's first heading and its description the first paragraph. Links between the
  documents become site links, and links to anything else in the repository point at GitHub.
- **Verified by.** `TestHandwrittenPages` and `TestGenerateMDX` in `cmd/docs/docs_test.go`.

### FR-83: per-flag pages

A tool may document its flags on pages of their own, and when it does, each must be generated
in every format.

- **Acceptance.** `rmx` declares a topic per flag and gets `rmx/flags/<flag>.*` in each format.
- **Verified by.** `TestGenerateAll`.

## 9. The website

### FR-45: navigation and search

The website must present the documentation with navigation and full-text search.

- **Acceptance.** The docs layout shows the page tree; a search endpoint answers queries over
  the same content.
- **Verified by.** The Web workflow builds both. Their behavior in a browser is not tested.

### FR-46: generated content

The website's documentation must be produced by the generator, not written separately.

- **Verified by.** The Docs workflow, which regenerates `web/content/docs` and fails on a
  difference.

## 10. The playground

### FR-47: the real tools, in memory

The playground must run the real tools in the browser, against a filesystem that exists only in
memory.

- **Acceptance.** The page loads the WebAssembly build of gnux and gives it an in-memory
  filesystem. No file on the visitor's machine is read or written.
- **Verified by.** `web/test/wasm.test.ts` and `web/test/memfs.test.ts`, which run the real
  `gnux.wasm` in Node against the same filesystem code, in the Web workflow.

### FR-48: the session's conveniences

The playground must offer command history, completion with an inline suggestion, example
commands, a full-screen terminal, and its own commands: a line starting with `/` belongs to the
page, not to the shell. `/help` lists those commands and every tool with what it does,
`/examples [N]` lists or runs an example, `/docs [TOOL]` opens the reference, `/history`,
`/clear`, `/fullscreen`, and `/refresh` resets the session. Esc or Ctrl+C while a command is
running also resets it, since a running command cannot otherwise be stopped.

- **Verified by.** `web/test/complete.test.ts` for completion and `web/test/commands.test.ts`
  for parsing, completing and suggesting `/` commands and the `/help` text. History, the
  examples panel, full screen and what each `/` command does are interface behavior with no
  automated test.

### FR-84: the WebAssembly API

The WebAssembly build must publish a small, stable JavaScript API.

- **Acceptance.** On start it sets `globalThis.gnux` with `version`, `tools`, `run(line)`,
  `cwd()` and an assignable `onOutput(fd, chunk)`, and calls `globalThis.gnuxReady` if it is a
  function. `run` returns a promise of the exit status. One line runs at a time.
- **Verified by.** "publishes its API" and the rest of `web/test/wasm.test.ts`.

### FR-85: one continuous session

Lines run in the playground must share one shell session, and resetting must start a new one.

- **Acceptance.** Variables and `cd` carry over from line to line. A syntax error does not end
  the session. Separate sessions do not see each other's files.
- **Verified by.** "cd carries over between lines", "variables carry over and expand" and
  "sessions are independent" in `web/test/wasm.test.ts`.

## 11. Search and sharing

### FR-49: page metadata

Every page must carry a unique title and description, a canonical URL, and social-sharing
metadata.

- **Verified by.** The Web workflow builds the metadata routes. Correctness is checked by
  review.

### FR-50: crawler files and structured data

The website must publish `robots.txt`, a sitemap covering every page, and structured data
describing the software.

- **Acceptance.** The sitemap is built from the documentation source, so a newly generated page
  is listed with no further change. `robots.txt` disallows the search endpoint.
- **Verified by.** The Web workflow builds them. Correctness is checked by review.

### FR-86: preview images

Every documentation page should have a generated preview image carrying its title and
description.

- **Verified by.** The Web workflow builds the image route. The images are not inspected
  automatically.
