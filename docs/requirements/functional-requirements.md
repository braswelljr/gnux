# Functional Requirements

Functional requirements define what gnux must do: the behavior every tool shares, the options
each of the 108 tools must support, the safeguards on destructive operations, how a tool is
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
  this register and keep their numbers. `FR-51` to `FR-72` are the first 22 tools, in alphabetical
  order, `FR-88` to `FR-94` the text tools added after them, and `FR-96` to `FR-147` the
  remaining coreutils tools, in alphabetical order, and `FR-156` to `FR-164` the tools added
  after the first release (`archx`, `factorx`, `sumx`, `tsortx`, `dirx`, `groupsx`, `lognamex`,
  `usersx`, `vdirx`), and `FR-170` to `FR-183` the last of coreutils (`basencx` to `syncx`,
  alphabetically). `FR-73` onward are the other
  requirements added since. The document is ordered by area, so the
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

### FR-165: options are parsed as GNU getopt parses them

Every tool must read its options as glibc's `getopt_long` does, wherever the command-line
framework (pflag) differs.

- **Behavior.** A long option may be abbreviated to any prefix that names one option only
  (`rmx --rec` is `--recursive`); a prefix of several is refused as `option '--co' is ambiguous;
  possibilities: '--color' '--context'`. Only GNU's own long names take part: a long name gnux
  gives an option GNU spells with a letter only (`lsx --long`), or a hidden second spelling,
  answers only when written in full and only when no GNU option starts with it, so an
  abbreviation means what it means to GNU (`mktempx --dir` is `--directory`). An option that
  takes no argument refuses one (`option '--force' doesn't allow an argument`). Mistakes are
  worded as glibc words them: `unrecognized option '--foo'`, `invalid option -- 'z'`, `option
  requires an argument -- 'n'`, `option '--lines' requires an argument`, followed by the usual
  `Try '<tool> --help'` line. Tools whose arguments are not options (`echox`, `testx`) parse
  nothing.
- **Verified by.** `internal/tool/getopt_test.go`, the conformance harness
  (`cmd/conformance_test.go`).

### FR-166: the GNU tools are the oracle

Each tool's output, diagnostics and exit status must be checked against the GNU tool it mirrors,
running both on the same command line.

- **Behavior.** `cmd/conformance_test.go` runs every line of `cmd/testdata/conformance/*.txt` with
  `sh -c` in two identical fresh directories: once with the GNU tools, once with gnux linked
  under the GNU names ahead of them on `PATH`. Standard output, standard error and the exit
  status must be equal, once gnux's own tool names (`rmx:`) are read as GNU's and each run's
  directory as `$DIR`. A documented deviation is not written as a case.
- **Verified by.** The `conformance` job of the Test workflow, against Debian's current GNU
  releases; `make conformance` on Linux and `make conformance/docker` anywhere with Docker.

### FR-167: backups of replaced destinations

`cpx`, `mvx`, `lnx` and `installx` must back up a destination they replace when asked, as GNU's
`--backup` does in `cp`, `mv`, `ln` and `install`.

| Option                | Meaning |
|-----------------------|---------|
| `--backup[=CONTROL]`  | Make a backup of each existing destination file |
| `-b`                  | Like `--backup` but does not accept an argument |
| `-S`, `--suffix=SUFFIX` | Override the usual backup suffix; also asks for backups |

- **Behavior.** The existing destination is renamed to its backup name before it is replaced.
  CONTROL is `none` or `off` (no backups), `simple` or `never` (FILE plus the suffix, replacing
  an earlier simple backup), `numbered` or `t` (FILE`.~N~`, N one past the highest number in
  use, never replacing a file) or `existing` or `nil` (numbered when numbered backups already
  exist, simple otherwise); an unambiguous abbreviation is accepted, and anything else is a
  usage error that lists the valid words. Without CONTROL, `VERSION_CONTROL` chooses, and is
  `existing` when unset or empty; it is read only when backups are asked for. The suffix is
  `-S SUFFIX`, else `SIMPLE_BACKUP_SUFFIX`, else `~`; an empty suffix or one containing a `/`
  means `~`. A name too long for the file system ends in a single `~` instead. Nothing that does
  not exist is backed up. `-v` names the backup: `'SRC' -> 'DEST' (backup: 'BACKUP')` (`cpx`,
  `installx`), `renamed 'SRC' -> 'DEST' (backup: 'BACKUP')` (`mvx`) and `'BACKUP' ~ 'DEST' =>
  'TARGET'` (`lnx`). When the destination cannot then be created, the backup is renamed back.
  A source that is the destination's simple backup is refused (`backing up 'DEST' might
  destroy source;  'SRC' not copied`, or `not moved`). `cpx` merges into a destination
  directory without backing it up, and `-f -b FILE FILE` copies FILE to its backup name;
  `mvx` backs up directories too; with a backup, a directory may replace a non-directory and,
  in `mvx`, the other way round. `--backup`, even `--backup=none`, cannot be combined with `-n`
  or `--update=none` in `cpx` and `mvx`.
- **Verified by.** `core/backup_test.go`, `internal/tool/backup_test.go`, the engines' tests,
  and `cmd/testdata/conformance/backups.txt`.

## 2. The tools

One requirement per tool. Each states the synopsis, the options that must be supported, the
behavior that must hold, and the documented deviations from the GNU tool. `--help` and
`--version` apply to every tool under FR-4 and are not repeated.

Several GNU options exist only as a single letter. The command-line framework requires a long
name for every option, so those letters also answer to a long name of gnux's own, which is
shown in the tables and noted as a deviation. Such a name is accepted only when written in full
and never completes an abbreviation (FR-165), so every GNU command line keeps its meaning.

### FR-156: archx

`archx` must print the machine architecture, as GNU `arch` does.

```text
archx [OPTION]...
```

archx has no options of its own.

- **Behavior.** archx prints the machine hardware name, exactly as unamex -m does, and takes no
  operands. The name comes from the operating system; on Windows and in the playground, gnux
  names the architecture it was built for the way Linux does (x86_64, aarch64, wasm32).
- **Verified by.** `core/arch/arch_test.go`, `cmd/archx/archx_test.go`.

### FR-96: b2sumx

`b2sumx` must print or check BLAKE2b (RFC 7693) (512-bit) checksums, as GNU `b2sum` does.

```text
b2sumx [OPTION]... [FILE]...
```

| Option             | Meaning |
|--------------------|---------|
| `-b`, `--binary`   | Read in binary mode |
| `-c`, `--check`    | Read checksums from the FILEs and check them |
| `--ignore-missing` | Don't fail or report status for missing files |
| `-l`, `--length`   | Digest length in bits; must not exceed the maximum for the blake2 algorithm and must be a multiple of 8 |
| `--quiet`          | Don't print OK for each successfully verified file |
| `--status`         | Don't output anything, status code shows success |
| `--strict`         | Exit non-zero for improperly formatted checksum lines |
| `--tag`            | Create a BSD-style checksum |
| `-t`, `--text`     | Read in text mode (default) |
| `-w`, `--warn`     | Warn about improperly formatted checksum lines |
| `-z`, `--zero`     | End each output line with NUL, not newline, and disable file name escaping |

- **Behavior.** b2sumx prints or checks BLAKE2b (RFC 7693) message digests (512 bits). For each
  FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text mode,
  '*' with -b), and the file name. With no FILE, or when FILE is -, it reads standard input. Files
  are always read as bytes, so -b and -t change only the mark; when both are given, -t wins. --tag
  prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a
  backslash, newline or carriage return is escaped and its line starts with a backslash, unless -z
  ends lines with NUL instead. -c reads such lines (in either style) from the FILEs and checks
  each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches,
  unreadable files and improperly formatted lines; --quiet drops the OK lines, --status prints
  nothing, --ignore-missing skips listed files that do not exist, -w warns about each improperly
  formatted line and --strict fails because of them. The exit status is 0 when everything was read
  (and, with -c, matched), 1 otherwise. -l BITS shortens the digest to BITS bits, a multiple of 8
  up to 512; tagged lines then name the length ('BLAKE2b-256').
- **Verified by.** `core/b2sum/b2sum_test.go`, `cmd/b2sumx/b2sumx_test.go`.

### FR-97: base32x

`base32x` must base32 encode or decode FILE, or standard input, to standard output, as GNU `base32` does.

```text
base32x [OPTION]... [FILE]
```

| Option                   | Meaning |
|--------------------------|---------|
| `-d`, `--decode`         | Decode data |
| `-i`, `--ignore-garbage` | When decoding, ignore non-alphabet characters |
| `-w`, `--wrap`           | Wrap encoded lines after COLS character (default 76); use 0 to disable line wrapping |

- **Behavior.** base32x encodes FILE in base32 (RFC 4648, with padding) and writes the result to
  standard output, or with -d decodes it. With no FILE, or when FILE is -, it reads standard
  input; at most one FILE is accepted. Encoded output is wrapped every 76 characters and ends with
  a newline; -w COLS wraps at COLS instead, and -w 0 writes one line with no newline at all. When
  decoding, newlines are ignored, and -i also ignores every other character outside the alphabet;
  any other stray character, or a truncated input, stops decoding with 'invalid input' and exit
  status 1, after the data decoded so far has been written.
- **Verified by.** `core/base32/base32_test.go`, `cmd/base32x/base32x_test.go`.

### FR-98: base64x

`base64x` must base64 encode or decode FILE, or standard input, to standard output, as GNU `base64` does.

```text
base64x [OPTION]... [FILE]
```

| Option                   | Meaning |
|--------------------------|---------|
| `-d`, `--decode`         | Decode data |
| `-i`, `--ignore-garbage` | When decoding, ignore non-alphabet characters |
| `-w`, `--wrap`           | Wrap encoded lines after COLS character (default 76); use 0 to disable line wrapping |

- **Behavior.** base64x encodes FILE in base64 (RFC 4648, with padding) and writes the result to
  standard output, or with -d decodes it. With no FILE, or when FILE is -, it reads standard
  input; at most one FILE is accepted. Encoded output is wrapped every 76 characters and ends with
  a newline; -w COLS wraps at COLS instead, and -w 0 writes one line with no newline at all. When
  decoding, newlines are ignored, and -i also ignores every other character outside the alphabet;
  any other stray character, or a truncated input, stops decoding with 'invalid input' and exit
  status 1, after the data decoded so far has been written.
- **Verified by.** `core/base64/base64_test.go`, `cmd/base64x/base64x_test.go`.

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

### FR-170: basencx

`basencx` must encode or decode data in one of several encodings, as GNU `basenc` does.

```text
basencx [OPTION]... [FILE]
```

| Option                   | Meaning |
|--------------------------|---------|
| `--base64`               | Same as the base64 program (RFC 4648 section 4) |
| `--base64url`            | File- and URL-safe base64 (section 5) |
| `--base32`               | Same as the base32 program (section 6) |
| `--base32hex`            | Extended hex alphabet base32 (section 7) |
| `--base16`               | Hex encoding (section 8) |
| `--base2msbf`            | Bit string, most significant bit first |
| `--base2lsbf`            | Bit string, least significant bit first |
| `--z85`                  | ZeroMQ Z85; input must be a whole number of groups |
| `-d`, `--decode`         | Decode data |
| `-i`, `--ignore-garbage` | When decoding, ignore non-alphabet characters |
| `-w`, `--wrap=COLS`      | Wrap encoded lines after COLS characters (76); 0 disables wrapping |

- **Behavior.** basencx encodes FILE (or standard input) in the selected encoding, or with -d
  decodes it, through the engine base64x and base32x share (`core/basenc`). An encoding must be
  given ("missing encoding type" otherwise), and the last one given wins. Encoded output wraps at 76
  columns and ends with a newline; -w 0 writes one line without one. Decoding ignores newlines, and
  -i every other character outside the alphabet; anything else, or a truncated input, ends with
  "invalid input" and status 1 after writing what was decoded. Z85 refuses input that is not a
  multiple of 4 bytes with GNU's "invalid input (length must be multiple of 4 characters)".
- **Deviations.** None known.
- **Verified by.** `core/basenc/basenc_test.go`, `cmd/basencx/basencx_test.go`, the GNU conformance
  cases in `cmd/testdata/conformance/more-tools.txt`.

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

### FR-99: chgrpx

`chgrpx` must change group ownership, as GNU `chgrp` does.

```text
chgrpx [OPTION]... GROUP FILE...
```

| Option                     | Meaning |
|----------------------------|---------|
| `-c`, `--changes`          | Like verbose but report only when a change is made |
| `--dereference`            | Affect the referent of each symbolic link (this is the default), rather than the symbolic link itself |
| `-L`, `--dereference-all`  | With -R, traverse every symbolic link to a directory encountered |
| `-H`, `--dereference-args` | With -R, follow symbolic links to directories given on the command line |
| `--from`                   | Change the group of each file only if its current owner and/or group match those specified here; either may be omitted (CURRENT_OWNER:CURRENT_GROUP) |
| `-h`, `--no-dereference`   | Affect symbolic links instead of any referenced file |
| `--no-preserve-root`       | Do not treat '/' specially |
| `-P`, `--physical`         | With -R, do not traverse any symbolic links (default) |
| `--preserve-root`          | Fail to operate recursively on '/' (the default) |
| `--quiet`                  | Same as --silent |
| `-R`, `--recursive`        | Operate on files and directories recursively |
| `--reference`              | Use RFILE's group rather than specifying a GROUP value |
| `-f`, `--silent`           | Suppress most error messages |
| `-v`, `--verbose`          | Output a diagnostic for every file processed |

- **Behavior.** chgrpx changes the group of each FILE to GROUP, a group name or numeric ID. An
  empty GROUP changes nothing but still checks that each FILE exists. --reference=RFILE takes the
  group from RFILE instead, and then every operand is a FILE. --from=CURRENT_OWNER:CURRENT_GROUP
  changes only the files whose current owner and group match; either part may be omitted, as in
  --from=alice or --from=:staff. A symbolic link given as a FILE is followed, and its referent
  changed, unless -h is given; when both -h and --dereference are given, -h wins. -R changes
  directory trees, following no symbolic link by default (-P); -H follows the ones given on the
  command line and -L every one, and when several are given the most cautious wins (-P, then -H,
  then -L). -c reports each change, -v every file, and -f suppresses most error messages. Unlike
  GNU chgrp, which leaves the safeguard off unless --preserve-root is given, chgrpx refuses to work
  recursively on '/' unless --no-preserve-root is given; when both are given, --preserve-root wins.
  On Windows, which has no Unix groups, every change fails with 'Operation not supported'.
- **Verified by.** `core/chgrp/chgrp_test.go`, `cmd/chgrpx/chgrpx_test.go`.

### FR-100: chmodx

`chmodx` must change file mode bits, as GNU `chmod` does.

```text
chmodx [OPTION]... MODE[,MODE]... FILE...
```

| Option                     | Meaning |
|----------------------------|---------|
| `-c`, `--changes`          | Like verbose but report only when a change is made |
| `--dereference`            | Affect the referent of each symbolic link, rather than the symbolic link itself |
| `-L`, `--dereference-all`  | With -R, follow every symbolic link to a directory |
| `-H`, `--dereference-args` | With -R, follow symbolic links given on the command line (the default) |
| `-h`, `--no-dereference`   | Affect each symbolic link, rather than the referent |
| `--no-preserve-root`       | Do not treat '/' specially |
| `-P`, `--physical`         | With -R, do not follow any symbolic links |
| `--preserve-root`          | Fail to operate recursively on '/' (the default) |
| `--quiet`                  | Same as --silent |
| `-R`, `--recursive`        | Change files and directories recursively |
| `--reference`              | Use RFILE's mode instead of specifying MODE values |
| `-f`, `--silent`           | Suppress most error messages |
| `-v`, `--verbose`          | Output a diagnostic for every file processed |

- **Behavior.** chmodx changes the permission bits of each FILE to MODE, which is either an octal
  number (755, 0644, 4755) or a comma-separated list of symbolic changes. Each change names who (u,
  g, o or a for user, group, others or all), an operator (+ adds, - removes, = sets exactly) and
  permissions: r, w, x, X (execute only on directories and on files already executable by someone),
  s (setuid/setgid) and t (sticky), or u, g or o to copy another class's bits. A symbolic change
  without a who honors the umask. --reference=RFILE copies RFILE's mode instead, and then every
  operand is a FILE. -R changes directory trees; -H (the default) follows symbolic links given on
  the command line, -L follows every symbolic link and -P none, and when several are given the most
  cautious wins (-P, then -H, then -L). A symbolic link given as a FILE is followed unless -h is
  given, and one met while recursing is not, unless --dereference is given (which changes its
  referent without descending into it) or -L; when both -h and --dereference are given, -h wins. A
  link that is not followed has its own mode changed where the system gives links one (macOS, the
  BSDs); on Linux it is reported as unchanged. -c reports each change, -v every file, and -f
  suppresses most error messages. A mode may also be written as an option, as in 'chmodx -w FILE';
  several such options combine as if joined with commas, and then, as in GNU, a file left with bits
  the mode would have cleared but for the umask is reported ('new permissions are ..., not ...')
  and counts as a failure. Unlike GNU chmod, which leaves the safeguard off unless --preserve-root
  is given, chmodx refuses to work recursively on '/' unless --no-preserve-root is given; when both
  are given, --preserve-root wins. On Windows only the owner's write bit maps onto the file system
  (the read-only attribute), so the other bits are accepted but have no effect.
- **Verified by.** `core/chmod/chmod_test.go`, `cmd/chmodx/chmodx_test.go`.

### FR-101: chownx

`chownx` must change file owner and group, as GNU `chown` does.

```text
chownx [OPTION]... [OWNER][:[GROUP]] FILE...
```

| Option                     | Meaning |
|----------------------------|---------|
| `-c`, `--changes`          | Like verbose but report only when a change is made |
| `--dereference`            | Affect the referent of each symbolic link (this is the default), rather than the symbolic link itself |
| `-L`, `--dereference-all`  | With -R, traverse every symbolic link to a directory encountered |
| `-H`, `--dereference-args` | With -R, follow symbolic links to directories given on the command line |
| `--from`                   | Change the ownership of each file only if its current owner and/or group match those specified here; either may be omitted (CURRENT_OWNER:CURRENT_GROUP) |
| `-h`, `--no-dereference`   | Affect symbolic links instead of any referenced file |
| `--no-preserve-root`       | Do not treat '/' specially |
| `-P`, `--physical`         | With -R, do not traverse any symbolic links (default) |
| `--preserve-root`          | Fail to operate recursively on '/' (the default) |
| `--quiet`                  | Same as --silent |
| `-R`, `--recursive`        | Operate on files and directories recursively |
| `--reference`              | Use RFILE's ownership rather than specifying values |
| `-f`, `--silent`           | Suppress most error messages |
| `-v`, `--verbose`          | Output a diagnostic for every file processed |

- **Behavior.** chownx changes the owner and/or group of each FILE. OWNER and GROUP are names or
  numeric IDs: 'OWNER' changes the owner only, 'OWNER:GROUP' both, 'OWNER:' the owner and the
  group to OWNER's login group, and ':GROUP' the group only; an empty spec changes nothing but
  still checks that each FILE exists. --reference=RFILE takes the owner and group from RFILE
  instead, and then every operand is a FILE. --from=CURRENT_OWNER:CURRENT_GROUP changes only the
  files currently owned as stated (either part may be omitted). A symbolic link given as a FILE is
  followed, and its referent changed, unless -h is given; when both -h and --dereference are
  given, -h wins. -R changes directory trees, following no symbolic link by default (-P); -H
  follows the ones given on the command line and -L every one, and when several are given the most
  cautious wins (-P, then -H, then -L). -c reports each change, -v every file, and -f suppresses
  most error messages. Unlike GNU chown, which leaves the safeguard off unless --preserve-root is
  given, chownx refuses to work recursively on '/' unless --no-preserve-root is given; when both
  are given, --preserve-root wins. On Windows, which has no Unix owners, every change fails with
  'Operation not supported'.
- **Verified by.** `core/chown/chown_test.go`, `cmd/chownx/chownx_test.go`.

### FR-102: cksumx

`cksumx` must print or check checksums and byte counts, as GNU `cksum` does.

```text
cksumx [OPTION]... [FILE]...
```

| Option              | Meaning |
|---------------------|---------|
| `-a`, `--algorithm` | Select the digest TYPE to use: sysv, bsd, crc, crc32b, md5, sha1, sha224, sha256, sha384, sha512, blake2b |
| `--base64`          | Emit base64-encoded digests, not hexadecimal |
| `-c`, `--check`     | Read checksums from the FILEs and check them |
| `--ignore-missing`  | Don't fail or report status for missing files |
| `-l`, `--length`    | Digest length in bits; must not exceed the maximum for the blake2 algorithm and must be a multiple of 8 |
| `--quiet`           | Don't print OK for each successfully verified file |
| `--raw`             | Emit a raw binary digest, not hexadecimal |
| `--status`          | Don't output anything, status code shows success |
| `--strict`          | Exit non-zero for improperly formatted checksum lines |
| `--tag`             | Create a BSD-style checksum (the default) |
| `--untagged`        | Create a reversed style checksum, without digest type |
| `-w`, `--warn`      | Warn about improperly formatted checksum lines |
| `-z`, `--zero`      | End each output line with NUL, not newline, and disable file name escaping |

- **Behavior.** cksumx prints a checksum and the byte count of each FILE. With no FILE, or when
  FILE is -, it reads standard input (and, with no FILE at all, prints no name). By default it
  computes the POSIX CRC (the 'crc' algorithm), as plain cksum always has. -a ALGORITHM picks
  another: sysv and bsd (the legacy sum checksums, with a block count rather than a byte count),
  crc32b (the CRC-32 of zlib and gzip), or the message digests md5, sha1, sha224, sha256, sha384,
  sha512 and blake2b. Digests are printed as BSD-style tagged lines ('SHA256 (FILE) = DIGEST');
  --untagged prints them in the md5sumx style instead, and when both --tag and --untagged are
  given, --tag wins. --base64 prints digests in base64, --raw writes the bare binary checksum of a
  single input, -l BITS shortens a blake2b digest, and -z ends lines with NUL. -c checks the files
  listed in tagged or untagged digest lines, detecting the algorithm from each tagged line when -a
  is not given, with --ignore-missing, --quiet, --status, --strict and -w as in md5sumx. The
  legacy algorithms cannot be checked.
- **Verified by.** `core/cksum/cksum_test.go`, `cmd/cksumx/cksumx_test.go`.

### FR-103: commx

`commx` must compare two sorted files line by line, as GNU `comm` does.

```text
commx [OPTION]... FILE1 FILE2
```

| Option                    | Meaning |
|---------------------------|---------|
| `--check-order`           | Check that the input is correctly sorted, even if all input lines are pairable |
| `--nocheck-order`         | Do not check that the input is correctly sorted |
| `--output-delimiter`      | Separate columns with STR |
| `-3`, `--suppress-common` | Suppress column 3 (lines that appear in both files) |
| `-1`, `--suppress-file1`  | Suppress column 1 (lines unique to FILE1) |
| `-2`, `--suppress-file2`  | Suppress column 2 (lines unique to FILE2) |
| `--total`                 | Output a summary |
| `-z`, `--zero-terminated` | Line delimiter is NUL, not newline |

- **Behavior.** commx compares two sorted files, FILE1 and FILE2, line by line. With no options it
  prints three columns: lines only in FILE1, lines only in FILE2, and lines in both, each column
  indented by one more TAB than the one before. -1, -2 and -3 suppress the matching column, so -12
  prints the common lines and -23 the lines only in FILE1. Either FILE may be - for standard
  input. Both files must be sorted: by default commx warns once per file when it finds a line out
  of order after a line that appears in only one file, and then exits with status 1; --check-order
  fails at the first line out of order, and --nocheck-order never checks. --output-delimiter=STR
  separates the columns with STR (an empty STR writes a NUL byte), --total adds a final line with
  the three counts and the word 'total', and -z makes NUL the line delimiter. Lines are compared
  byte by byte, as in the C locale, whatever LANG says, so sort the inputs with sortx (or LC_ALL=C
  sort). The framework requires a long name for every option, so -1, -2 and -3 also answer to
  gnux's own --suppress-file1, --suppress-file2 and --suppress-common. When both --check-order and
  --nocheck-order are given, --check-order wins.
- **Verified by.** `core/comm/comm_test.go`, `cmd/commx/commx_test.go`.

### FR-53: cpx

`cpx` must copy files and directories, as GNU `cp` does.

```text
cpx [OPTION]... SOURCE... DIRECTORY
```

| Option                          | Meaning                                                                  |
|---------------------------------|--------------------------------------------------------------------------|
| `-a`, `--archive`               | Same as `-dR --preserve=all`                                             |
| `--backup[=CONTROL]`, `-b`, `-S` | Back up each existing destination (FR-167)                              |
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
| `--remove-destination`        | Remove each existing destination before opening it (contrast with `--force`) |
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
  same base name. `--remove-destination` removes an existing destination before writing it, so
  a destination symbolic link is replaced rather than written through and a file shared by hard
  links is not changed under its other names; with `-v` it reports `removed 'DEST'`. `-l` with
  `-s`, and `-t` with `-T`, are usage errors. Backups follow FR-167.
- **Deviations.** Option order is not visible, so `-n` overrides `-i`, and an explicit `-P` or
  `-d` overrides `-H`, which overrides `-L`. `-d`, `-H` and `-p` also answer to the long names
  `--no-dereference-links`, `--dereference-args` and `--preserve-default`. On Windows only the
  read-only attribute and the modification time are carried over. `-b` also answers to the
  long name `--backup-default`. Not implemented: `--reflink`, `--sparse`, `--attributes-only`,
  `--copy-contents`, `--keep-directory-symlink`, `-Z` and `--context`,
  `--debug`.
- **Verified by.** `core/cp/cp_test.go`, `cmd/cpx/cpx_test.go`.

### FR-104: csplitx

`csplitx` must split a file into sections determined by context lines, as GNU `csplit` does.

```text
csplitx [OPTION]... FILE PATTERN...
```

| Option                      | Meaning |
|-----------------------------|---------|
| `-n`, `--digits`            | Use specified number of digits instead of 2 |
| `-z`, `--elide-empty-files` | Suppress generation of empty output files |
| `-k`, `--keep-files`        | Do not remove output files on errors |
| `-f`, `--prefix`            | Use PREFIX instead of 'xx' |
| `-s`, `--quiet`             | Do not print counts of output file sizes |
| `-q`, `--silent`            | Same as --quiet |
| `-b`, `--suffix-format`     | Use sprintf FORMAT instead of %02d |
| `--suppress-matched`        | Suppress the lines matching PATTERN |

- **Behavior.** csplitx copies FILE (- for standard input) into files xx00, xx01, ..., starting a
  new file at each PATTERN, and prints the size in bytes of each file it makes. A PATTERN is a
  line number N (the next file starts at line N), /REGEXP/ (the next file starts at the next line
  matching the POSIX basic regular expression REGEXP), or %REGEXP% (the lines up to that match are
  skipped, not written). A regular expression may be followed by an offset, +N or -N, that moves
  the split that many lines after or before the matching line. A pattern followed by {N} is
  repeated N more times, and by {*} as many times as it matches; a line number repeated advances
  by its own value each time. The lines after the last split go to a final file. -f PREFIX
  replaces xx, -n N uses N digits, and -b FORMAT names the files with a printf format holding one
  integer conversion (%02d.txt, %03x). -s (or -q) does not print the sizes, -z removes empty files
  (and reuses their number), and --suppress-matched leaves the matching lines out. If a pattern
  cannot be satisfied (a line number past the end, a regular expression that never matches),
  csplitx reports it and removes every file it made, so a failed run leaves nothing half-done
  behind; -k keeps them. Lines are compared as bytes.
- **Verified by.** `core/csplit/csplit_test.go`, `cmd/csplitx/csplitx_test.go`.

### FR-90: cutx

`cutx` must print selected parts of each line, as GNU `cut` does.

```text
cutx OPTION... [FILE]...
```

| Option                      | Meaning                                                                      |
|-----------------------------|------------------------------------------------------------------------------|
| `-b`, `--bytes=LIST`        | Select only these bytes                                                      |
| `-c`, `--characters=LIST`   | Select only these characters                                                 |
| `-d`, `--delimiter=DELIM`   | Use DELIM instead of TAB for field delimiter                                 |
| `-f`, `--fields=LIST`       | Select only these fields; lines with no delimiter are printed whole          |
| `-n`                        | Ignored                                                                      |
| `--complement`              | Complement the set of selected bytes, characters or fields                   |
| `-s`, `--only-delimited`    | Do not print lines not containing delimiters                                 |
| `--output-delimiter=STRING` | Use STRING as the output delimiter                                           |
| `-z`, `--zero-terminated`   | Line delimiter is NUL, not newline                                           |

- **Behavior.** Exactly one of `-b`, `-c`, `-f` is required. LIST is N, N-, N-M or -M,
  separated by commas or blanks, numbered from 1; the selection is printed in input order,
  each position once. `-d ''` means NUL. The output delimiter defaults to the input delimiter
  for `-f`, and for `-b`/`-c` is written only when given, between non-overlapping ranges. List
  and option errors are usage errors in GNU's words (`fields are numbered from 1`,
  `invalid decreasing range`, `the delimiter must be a single character`,
  `only one type of list may be specified`, ...). Every output line ends with the line
  delimiter.
- **Deviations.** As in GNU, `-c` counts bytes. Giving the same kind of list twice keeps the
  last instead of refusing.
- **Verified by.** `core/cut/cut_test.go`, `cmd/cutx/cutx_test.go`.

### FR-105: datex

`datex` must print the date and time, as GNU `date` does.

```text
datex [OPTION]... [+FORMAT]
```

| Option              | Meaning |
|---------------------|---------|
| `-d`, `--date`      | Display time described by STRING, not 'now' |
| `-f`, `--file`      | Like --date; once for each line of DATEFILE |
| `-I`, `--iso-8601`  | Output date/time in ISO 8601 format. FMT='date' for date only (the default), 'hours', 'minutes', 'seconds', or 'ns' |
| `-r`, `--reference` | Display the last modification time of FILE |
| `--resolution`      | Output the available resolution of timestamps |
| `--rfc-3339`        | Output date/time in RFC 3339 format. FMT='date', 'seconds', or 'ns' |
| `-R`, `--rfc-email` | Output date and time in RFC 5322 format |
| `-s`, `--set`       | Set time described by STRING (not supported) |
| `--universal`       | Same as --utc |
| `-u`, `--utc`       | Print or set Coordinated Universal Time (UTC) |

- **Behavior.** datex prints the current date and time, or the one -d STRING describes, in the C
  locale's default format ('Mon Jan 2 15:04:05 UTC 2006') or in +FORMAT. FORMAT takes GNU's
  conversions: %Y %m %d %H %M %S and friends, %a %A %b %B %p (in English), %s (seconds since the
  epoch), %N (nanoseconds), %z %:z %::z %:::z and %Z (zones), %j %U %W %V %G %u %w (day and week
  numbers), %F %T %D %R %r %c %x %X %h %e %k %l %C %y %g %n %t %%, and the flags -, _, 0, ^ and #
  with a field width (%-d, %_H, %10s). -I[FMT] prints ISO 8601 (date, hours, minutes, seconds or
  ns; date by default), --rfc-3339=FMT RFC 3339 (date, seconds or ns) and -R RFC 5322; only one
  output format may be given. -u uses UTC, and otherwise the TZ environment variable or the local
  zone applies. -d STRING accepts the forms GNU date accepts most often: 'now', 'today',
  'yesterday', 'tomorrow', '@SECONDS', ISO 8601 and RFC 3339 dates and times, MM/DD/YYYY, 'Jan 2
  2006', '2 Jan 2006', times of day (14:30, 2:30pm, noon, midnight) with UTC, Z or a numeric
  offset, weekdays ('next friday', 'last monday') and relative items ('3 days ago', '+2 hours',
  'next month', 'fortnight'), combined as in '2026-01-31 +1 month' or 'tomorrow 09:00'. Anything
  else is reported as an invalid date, where GNU's grammar is larger still. -f FILE prints one date
  per line of FILE (- for standard input) and -r FILE the last modification time of FILE. gnux
  never changes system settings, so -s and a MMDDhhmm[[CC]YY][.ss] operand fail with 'cannot set
  date: Operation not supported'. --resolution prints the resolution of the system clock, measured
  as GNU measures it, as a time that many seconds after the epoch (0.000000001 for a nanosecond
  clock), and in any format '%-N' prints as many digits of nanoseconds as that resolution needs.
  --debug is not implemented.
- **Verified by.** `core/date/date_test.go`, `cmd/datex/datex_test.go`.

### FR-106: ddx

`ddx` must convert and copy a file, block by block, as GNU `dd` does.

```text
ddx [OPERAND]...
```

`ddx` takes no options besides `--help` and `--version`.

- **Behavior.** ddx copies its input to its output in blocks, optionally converting it, and
  reports what it copied on standard error. Operands are KEY=VALUE: if=FILE and of=FILE name the
  input and output (standard input and output by default); bs=N sets the block size for both and
  writes each block as read, while ibs=N and obs=N set them separately (512 by default) and
  reblock the output; cbs=N is the record size for conv=block and conv=unblock; skip=N (also
  iseek=N) skips N input blocks, seek=N (also oseek=N) skips N output blocks, and count=N copies
  only N input blocks. N may carry a suffix: c (1), w (2), b (512), kB, MB, GB, TB, PB, EB (powers
  of 1000) or K, M, G, T, P, E, alone or followed by iB (powers of 1024), and may be a product
  such as 2x512. conv= takes a comma-separated list: notrunc (do not truncate the output), noerror
  (continue after read errors), sync (pad short input blocks to ibs with NUL bytes, or spaces with
  block/unblock), fsync and fdatasync (flush the output before finishing), excl and nocreat (fail
  if the output exists, or does not), lcase and ucase, swab (swap each pair of bytes), block
  (newline-terminated lines to fixed cbs-sized records) and unblock (the reverse). iflag= accepts
  fullblock, count_bytes and skip_bytes; oflag= accepts seek_bytes and append; binary and text are
  accepted by both and ignored. status=none prints nothing but errors, status=noxfer omits the
  transfer line. The summary reads '1+0 records in / 1+0 records out / 512 bytes copied, 0.0001 s,
  5.1 MB/s', where N+M counts full and partial blocks. Unless conv=notrunc, a regular output file
  is truncated at the seek offset; an output that cannot seek is advanced by writing NUL bytes.
  Differences from GNU dd: status=progress prints only the final statistics; the conversions
  ascii, ebcdic, ibm, sparse and nocache and the flags direct, dsync, sync, nonblock, noatime,
  noctty, nofollow and cio are not implemented and are refused as invalid; block and unblock do
  nothing without cbs; a closed output pipe ends ddx quietly instead of by SIGPIPE; and no
  statistics are printed on SIGINFO or SIGUSR1.
- **Verified by.** `core/dd/dd_test.go`.

### FR-107: dfx

`dfx` must report file system space usage, as GNU `df` does.

```text
dfx [OPTION]... [FILE]...
```

| Option                   | Meaning |
|--------------------------|---------|
| `-a`, `--all`            | Include pseudo, duplicate, inaccessible file systems |
| `-B`, `--block-size`     | Scale sizes by SIZE before printing them; e.g., '-BM' prints sizes in units of 1,048,576 bytes |
| `-x`, `--exclude-type`   | Limit listing to file systems not of type TYPE |
| `-h`, `--human-readable` | Print sizes in powers of 1024 (e.g., 1023M) |
| `-i`, `--inodes`         | List inode information instead of block usage |
| `-k`, `--kilobytes`      | Like --block-size=1K |
| `-l`, `--local`          | Limit listing to local file systems |
| `--no-sync`              | Do not invoke sync before getting usage info (default) |
| `--output`               | Use the output format defined by FIELD_LIST, or print all fields if FIELD_LIST is omitted |
| `-P`, `--portability`    | Use the POSIX output format |
| `-T`, `--print-type`     | Print file system type |
| `-H`, `--si`             | Print sizes in powers of 1000 (e.g., 1.1G) |
| `--sync`                 | Invoke sync before getting usage info (no effect) |
| `-v`, `--sysv`           | (ignored) |
| `--total`                | Elide all entries insignificant to available space, and produce a grand total |
| `-t`, `--type`           | Limit listing to file systems of type TYPE |

- **Behavior.** dfx shows, for each mounted file system, its size, the space used and available,
  the percentage used and where it is mounted. With FILE operands it shows only the file systems
  holding them. Sizes are in 1024-byte blocks by default; -h prints them like 23G (--si/-H in
  powers of 1000), -k and -m use KiB and MiB, and -B SIZE any unit. Use% is used/(used+available),
  rounded up, so space reserved for the superuser shows as used. -i reports inodes instead of
  blocks, -T adds the file system type, -P uses the POSIX format, and --output=FIELDS chooses the
  columns (source, fstype, itotal, iused, iavail, ipcent, size, used, avail, pcent, file, target).
  -t TYPE and -x TYPE select or exclude file system types, -l leaves out network file systems, and
  --total adds a grand total. Without -a, pseudo file systems (proc, sysfs, devfs, ...), file
  systems with no blocks and duplicate mounts of the same device are left out; -a shows them all,
  with '-' for any that cannot be examined. The block size defaults to DF_BLOCK_SIZE, BLOCK_SIZE
  or BLOCKSIZE when set. GNU df only runs where coreutils is installed; dfx reports the mounted
  file systems on Linux, macOS and FreeBSD, and on Windows lists every drive (C:\, D:\, network
  drives) with the same columns, quietly skipping a card reader or DVD drive with no medium.
  --sync and --no-sync are accepted and have no effect, and -v is ignored, as in GNU.
- **Verified by.** `core/df/df_test.go`, `cmd/dfx/dfx_test.go`.

### FR-171: dircolorsx

`dircolorsx` must print commands that set LS_COLORS, as GNU `dircolors` does.

```text
dircolorsx [OPTION]... [FILE]
```

| Option                         | Meaning |
|--------------------------------|---------|
| `-b`, `--sh`, `--bourne-shell` | Output Bourne shell code to set LS_COLORS |
| `-c`, `--csh`, `--c-shell`     | Output C shell code to set LS_COLORS |
| `-p`, `--print-database`       | Output the default database |
| `--print-ls-colors`            | Output each color, escaped for display |

- **Behavior.** dircolorsx reads FILE (- is standard input) or the built-in database and prints
  LS_COLORS as `LS_COLORS='...';` and `export LS_COLORS` (-b) or `setenv LS_COLORS '...'` (-c),
  guessing the syntax from $SHELL (csh and tcsh are C shell) and failing when SHELL is unset. TERM
  and COLORTERM lines filter what follows by glob against $TERM ("none" when unset) and $COLORTERM.
  A line without a value is "invalid line;  missing second token", and an unknown keyword after a
  matching TERM line is "unrecognized keyword"; either prints nothing and exits 1. The built-in
  database is GNU coreutils 9.7's, as `dircolors -p` prints it.
- **Deviations.** None known.
- **Verified by.** `core/dircolors/dircolors_test.go`, `cmd/dircolorsx/dircolorsx_test.go`, the GNU
  conformance cases in `cmd/testdata/conformance/more-tools.txt`.

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

### FR-160: dirx

`dirx` must list directory contents in columns, as GNU `dir` does.

```text
dirx [OPTION]... [FILE]...
```

dirx takes every option of lsx (FR-60).

- **Behavior.** dirx is lsx with GNU dir's defaults: it takes every lsx option (FR-60) and
  lists exactly as lsx does, but arranges the names in columns whether or not the output is a terminal, and writes unprintable characters in names as backslash
  escapes, as -b does, and never prints `?` for one without `-q`. An option that picks a format
  or a quoting style, or the `QUOTING_STYLE` variable, replaces those defaults. Diagnostics and
  exit statuses are lsx's, prefixed with dirx.
- **Verified by.** `cmd/dirx/dirx_test.go`, `core/ls/ls_test.go`.

### FR-108: dux

`dux` must estimate file space usage, as GNU `du` does.

```text
dux [OPTION]... [FILE]...
```

| Option                     | Meaning |
|----------------------------|---------|
| `-a`, `--all`              | Write counts for all files, not just directories |
| `-A`, `--apparent-size`    | Print apparent sizes rather than device usage |
| `-B`, `--block-size`       | Scale sizes by SIZE before printing them; e.g., '-BM' prints sizes in units of 1,048,576 bytes |
| `-b`, `--bytes`            | Equivalent to '--apparent-size --block-size=1' |
| `-l`, `--count-links`      | Count sizes many times if hard linked |
| `-L`, `--dereference`      | Dereference all symbolic links |
| `-D`, `--dereference-args` | Dereference only symlinks that are listed on the command line |
| `--exclude`                | Exclude files that match PATTERN |
| `-X`, `--exclude-from`     | Exclude files that match any pattern in FILE |
| `--files0-from`            | Summarize device usage of the NUL-terminated file names specified in file F; if F is -, then read names from standard input |
| `-h`, `--human-readable`   | Print sizes in human readable format (e.g., 1K 234M 2G) |
| `--inodes`                 | List inode usage information instead of block usage |
| `-k`, `--kilobytes`        | Like --block-size=1K |
| `-d`, `--max-depth`        | Print the total for a directory (or file, with --all) only if it is N or fewer levels below the command line argument |
| `-m`, `--megabytes`        | Like --block-size=1M |
| `-P`, `--no-dereference`   | Don't follow any symbolic links (this is the default) |
| `-0`, `--null`             | End each output line with NUL, not newline |
| `-x`, `--one-file-system`  | Skip directories on different file systems |
| `-S`, `--separate-dirs`    | For directories do not include size of subdirectories |
| `--si`                     | Like -h, but use powers of 1000 not 1024 |
| `-s`, `--summarize`        | Display only a total for each argument |
| `-t`, `--threshold`        | Exclude entries smaller than SIZE if positive, or entries greater than SIZE if negative |
| `--time`                   | Show time of the last modification of any file in the directory, or any of its subdirectories; with WORD: atime, access, use, ctime or status |
| `--time-style`             | Show times using STYLE, which can be: full-iso, long-iso, iso, or +FORMAT |
| `-c`, `--total`            | Produce a grand total |

- **Behavior.** dux shows how much disk space each directory under each FILE (the current
  directory by default) takes, including everything below it, in 1024-byte blocks rounded up. -a
  lists files too, -s only the operands, and -d N only N levels down; -c adds a grand total. -h
  prints sizes like 1.5K, 23M and 4.0G (--si in powers of 1000), -k and -m use KiB and MiB, -b
  bytes, and -B SIZE any unit ('-BM' prints the unit too). --apparent-size counts the bytes files
  hold rather than the blocks allocated to them, which differs for sparse and compressed files.
  --inodes counts files instead of space. -S leaves subdirectories out of a directory's total, -x
  stays on one file system, -l counts a hard-linked file every time it is met (by default it is
  counted once), -L follows symbolic links and -H only those given as operands. -t SIZE hides
  entries smaller than SIZE (or, negative, larger). --exclude=PATTERN and -X FILE skip matching
  paths. --time adds the latest modification (or --time=atime, ctime) time in each subtree,
  formatted by --time-style. --files0-from=F reads NUL-separated operands from F, and -0 ends
  output lines with NUL. Where GNU du reads one directory at a time and prints entries in whatever
  order the file system returns them, dux reads directories in parallel, which is much faster on
  large trees and network drives, and then adds everything up in name order, so the output, and
  which path a hard-linked file is counted under, is the same every run. The block size defaults
  to DU_BLOCK_SIZE, BLOCK_SIZE or BLOCKSIZE when set. Names are quoted only when output goes to a
  terminal.
- **Verified by.** `core/du/du_test.go`, `cmd/dux/dux_test.go`.

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

### FR-109: expandx

`expandx` must convert tabs to spaces, as GNU `expand` does.

```text
expandx [OPTION]... [FILE]...
```

| Option            | Meaning |
|-------------------|---------|
| `-i`, `--initial` | Do not convert tabs after non blanks |
| `-t`, `--tabs`    | Have tabs N characters apart, not 8, or use comma separated list of explicit tab positions |

- **Behavior.** expandx writes each FILE to standard output with every tab replaced by the spaces
  that reach the next tab stop. With no FILE, or when FILE is -, it reads standard input. Tab
  stops are every 8 columns unless -t says otherwise: -t N puts them every N columns, and -t LIST
  (numbers separated by commas or blanks, in ascending order) puts them at those columns; the last
  number may be written /N for a stop at every multiple of N after it, or +N for a stop every N
  columns after it. Past the last listed stop a tab becomes one space. Several -t options add to
  one list, and the obsolete -N form (expandx -4) is a list of its own, in the order given; GNU's
  comma form -4,8 is not accepted, and the digits of two -N options in a row run together (-4 -8
  reads as -48), so write -t 4,8 instead. -i converts only the tabs before the first non-blank
  character of each line. A backspace moves back one column. Like GNU expand in the C locale,
  expandx counts bytes rather than characters.
- **Verified by.** `core/expand/expand_test.go`, `cmd/expandx/expandx_test.go`.

### FR-110: exprx

`exprx` must evaluate an expression and print its value, as GNU `expr` does.

```text
exprx EXPRESSION
```

`exprx` takes no options besides `--help` and `--version`.

- **Behavior.** exprx evaluates EXPRESSION, given as separate arguments, and prints its value.
  From the loosest binding: A | B is A if it is neither empty nor 0, otherwise B (or 0); A & B is
  A if neither is empty nor 0, otherwise 0; A = B (also ==), A != B, and the orderings compare as
  integers when both sides are integers and as strings otherwise, giving 1 or 0; + and - add and
  subtract; * / and % multiply, divide (truncating toward zero) and take the remainder; STRING :
  REGEX matches. Integers are an optional minus sign and decimal digits, of any size. A match is
  anchored at the start of STRING: when REGEX has a \( \) group the value is the text the first
  group matched (or empty), otherwise the number of bytes matched (or 0). REGEX is a POSIX basic
  regular expression with GNU's \+, \? and \| extensions and character classes such as
  [[:digit:]]. The keywords match STRING REGEX, substr STRING POS LENGTH, index STRING CHARS and
  length STRING do the same as their names, and + TOKEN takes TOKEN as a string even if it is a
  keyword or an operator. Parentheses group. Operators must be separate arguments, and shell
  characters such as * | & ( ) must be quoted. A leading -- is skipped. The exit status is 0 when
  the value is neither empty nor 0, 1 when it is, 2 for an invalid expression (syntax error,
  non-integer argument, division by zero, a bad regular expression) and 3 for other errors.
  Differences from GNU expr: strings are measured and sliced in bytes and compared byte by byte
  (the C locale), and back references (\1) and the [= =] and [. .] bracket forms are not supported
  in a REGEX.
- **Verified by.** `core/expr/expr_test.go`.

### FR-157: factorx

`factorx` must print the prime factors of each number, as GNU `factor` does.

```text
factorx [OPTION] [NUMBER]...
```

| Option              | Meaning |
|---------------------|---------|
| `-h`, `--exponents` | Print repeated factors in form p^e unless e is 1 |

- **Behavior.** factorx prints each NUMBER, a colon and its prime factors in ascending order,
  a repeated factor as many times as it divides the number; 0 and 1 print just the colon. With
  no NUMBER it reads numbers separated by spaces, tabs and newlines from standard input and
  answers each line as it is read. A NUMBER may have leading spaces and one '+', and must
  otherwise be decimal digits; anything else is reported as 'is not a valid positive integer',
  the rest are still factored, and the exit status is 1. A NUMBER has no size limit: below
  2^64 it is factored with trial division, Miller-Rabin and Pollard's rho, above with
  arbitrary-precision arithmetic.
- **Verified by.** `core/factor/factor_test.go`, `cmd/factorx/factorx_test.go`,
  `cmd/robust_test.go` (it answers each line).

### FR-57: falsex

`falsex` must do nothing, unsuccessfully, as GNU `false` does.

```text
falsex [ignored command line arguments]
```

- **Behavior.** Exits with status 1. Reads no input, writes no output and ignores every
  argument. It has no options of its own.
- **Deviations.** None. `falsex` has no engine package: the adapter is the whole tool.
- **Verified by.** `cmd/falsex/falsex_test.go`.

### FR-148: findx

`findx` must search for files in a directory hierarchy, as GNU `find` (findutils) does.

```text
findx [-H] [-L] [-P] [STARTING-POINT]... [EXPRESSION]
```

`findx` takes an expression, not GNU-style options, so the engine parses the whole command
line; `--help` and `--version` are honored as the sole argument.

| Group      | Supported                                                                    |
|------------|------------------------------------------------------------------------------|
| Symlinks   | `-P` (default), `-H`, `-L`: the last one given wins                           |
| Operators  | `( )`, `!` / `-not`, `-a` / `-and` (also implied), `-o` / `-or`, `,`          |
| Options    | `-maxdepth`, `-mindepth`, `-depth` / `-d`, `-xdev` / `-mount`, `-daystart`, `-regextype`, `-follow`, `-warn`, `-nowarn`; `-noleaf`, `-ignore_readdir_race`, `-noignore_readdir_race` and `-O0` to `-O3` are accepted and do nothing |
| Tests      | `-name`, `-iname`, `-path` / `-wholename`, `-ipath` / `-iwholename`, `-lname`, `-ilname`, `-regex`, `-iregex`, `-type`, `-xtype`, `-size`, `-empty`, `-newer`, `-anewer`, `-cnewer`, `-newerXY`, `-mtime`, `-atime`, `-ctime`, `-mmin`, `-amin`, `-cmin`, `-perm`, `-user`, `-group`, `-uid`, `-gid`, `-nouser`, `-nogroup`, `-links`, `-inum`, `-samefile`, `-readable`, `-writable`, `-executable`, `-true`, `-false` |
| Actions    | `-print`, `-print0`, `-printf`, `-fprint`, `-fprint0`, `-fprintf`, `-ls`, `-fls`, `-delete`, `-prune`, `-quit`, `-exec` and `-execdir` (with `;` and `{} +`), `-ok` and `-okdir` |

- **Behavior.** With no starting point findx searches `.`; with no action it prints each
  matching path. Operators bind as in GNU and evaluation short-circuits. `-delete` implies
  `-depth` and refuses `.`. `-exec COMMAND ;` is true when the command exits 0; `-exec COMMAND
  {} +` runs the command on as many paths as fit and makes the exit status 1 when a batch
  fails. An unreadable directory is diagnosed and the walk continues. The exit status is 0
  when everything was processed and 1 otherwise.
- **Deviations.** Directory entries are visited in name order, where GNU uses the order the
  file system returns. Paths are joined with `/` on every platform, so `-path` and `-regex`
  patterns are portable. A command named after a gnux tool runs in-process, and anything else
  is a program on `PATH`; under `-execdir` an in-process tool is given the full path, since it
  cannot change directory. A command that cannot be run makes the exit status 1, which GNU
  leaves at 0 for the `;` form. Regular expressions are translated to RE2, so back references
  are refused. `-newerXt` accepts a fixed set of date spellings and `@SECONDS`. A command-line
  mistake is followed by the "Try" hint. Off Unix there are no owners, inodes or devices: IDs
  print 0, links 1, `-xdev` does nothing and ctime is the modification time. Not implemented,
  and refused: `-fstype`, `-context`, `-used`, `-files0-from`, `-D`, and `%F` and `%Z` in
  `-printf`.
- **Verified by.** `core/find/find_test.go`, `core/find/find_unix_test.go`,
  `core/find/internal_test.go`, `cmd/findx/findx_test.go`.

### FR-111: fmtx

`fmtx` must reformat each paragraph of text, as GNU `fmt` does.

```text
fmtx [-WIDTH] [OPTION]... [FILE]...
```

| Option                     | Meaning |
|----------------------------|---------|
| `-c`, `--crown-margin`     | Preserve indentation of first two lines |
| `-g`, `--goal`             | Goal width (default of 93% of width) |
| `-p`, `--prefix`           | Reformat only lines beginning with STRING, reattaching the prefix to reformatted lines |
| `-s`, `--split-only`       | Split long lines, but do not refill |
| `-t`, `--tagged-paragraph` | Indentation of first line different from second |
| `-u`, `--uniform-spacing`  | One space between words, two after sentences |
| `-w`, `--width`            | Maximum line width (default of 75 columns) |

- **Behavior.** fmtx fills and joins the lines of each paragraph so that they come as close as
  possible to a goal width without exceeding the maximum width, and writes the result to standard
  output. With no FILE, or when FILE is -, it reads standard input. Paragraphs are separated by
  blank lines and by changes of indentation; the indentation of each paragraph is kept. The line
  breaks are chosen as GNU fmt chooses them, minimizing raggedness over the whole paragraph and
  avoiding breaking after the first word of a sentence or before its last. -w WIDTH sets the
  maximum width (75 by default, at most 2500), also written -WIDTH; -g WIDTH sets the goal (93% of
  the maximum by default, and without -w the maximum becomes the goal plus 10). -c (crown margin)
  keeps the indentation of the first two lines and aligns the rest with the second; -t (tagged
  paragraph) is the same, except that a first line indented like the second is a paragraph of its
  own. -s only splits long lines and never joins short ones, which suits code. -u puts one space
  between words and two after sentences. -p STRING reformats only the lines that begin with STRING
  (after any indentation) and puts it back on each output line; the other lines are copied
  unchanged. A sentence ends at '.', '?' or '!' followed by the end of the line or two spaces, as
  in GNU fmt.
- **Verified by.** `core/fmt/fmt_test.go`, `cmd/fmtx/fmtx_test.go`.

### FR-112: foldx

`foldx` must wrap each input line to fit in a specified width, as GNU `fold` does.

```text
foldx [OPTION]... [FILE]...
```

| Option           | Meaning |
|------------------|---------|
| `-b`, `--bytes`  | Count bytes rather than columns |
| `-s`, `--spaces` | Break at spaces |
| `-w`, `--width`  | Use WIDTH columns instead of 80 |

- **Behavior.** foldx writes each FILE to standard output, breaking any line longer than 80
  columns into several. With no FILE, or when FILE is -, it reads standard input. -w WIDTH sets
  the width; the obsolete form -WIDTH (foldx -72) is accepted too, and whichever is given last
  wins. Columns are counted as a terminal would: a tab advances to the next multiple of 8, a
  backspace moves back one column and a carriage return goes back to column 0. -b counts bytes
  instead, so every byte takes one position. -s breaks a long line after the last blank that fits,
  rather than exactly at the width, so words are not split unless one is longer than the width.
  Like GNU fold in the C locale, foldx counts bytes rather than characters, so a multibyte UTF-8
  character takes more than one column.
- **Verified by.** `core/fold/fold_test.go`, `cmd/foldx/foldx_test.go`.

### FR-149: grepx

`grepx` must print the lines that match patterns, as GNU `grep` does.

```text
grepx [OPTION]... PATTERNS [FILE]...
```

| Option                              | Meaning                                                            |
|-------------------------------------|--------------------------------------------------------------------|
| `-G`, `-E`, `-F`                    | PATTERNS are basic (default), extended regular expressions, or fixed strings |
| `-e PATTERNS`, `-f FILE`            | Take the patterns from the option or from FILE; both repeat         |
| `-i`, `-y`, `--no-ignore-case`      | Ignore, or do not ignore, case                                      |
| `-v`, `-w`, `-x`                    | Select non-matching lines; match whole words; match whole lines     |
| `-c`, `-l`, `-L`, `-q`, `-o`        | Print a count, the matching files, the non-matching files, nothing, or only the matched parts |
| `-m NUM`                            | Stop after NUM selected lines                                       |
| `-b`, `-H`, `-h`, `-n`, `-T`, `-Z`, `--label=LABEL` | Control the prefix of each output line              |
| `-A NUM`, `-B NUM`, `-C NUM`, `-NUM`, `--group-separator=SEP`, `--no-group-separator` | Context lines |
| `-r`, `-R`, `-d ACTION`, `-D ACTION` | Recurse (without or with following symbolic links); handle directories and devices |
| `--include=GLOB`, `--exclude=GLOB`, `--exclude-from=FILE`, `--exclude-dir=GLOB` | Select the files a recursive search reads |
| `-a`, `-I`, `--binary-files=TYPE`, `-U` | Treat binary files as text, skip them, or choose; `-U` does nothing |
| `-s`                                | Suppress messages about missing or unreadable files                 |
| `-z`                                | Lines end in NUL, not newline                                       |
| `--color[=WHEN]`, `--colour[=WHEN]` | Highlight matches, names and separators with GNU's default colors   |
| `--line-buffered`                   | Flush the output after every line                                   |

- **Behavior.** PATTERNS is the first operand unless `-e` or `-f` gave patterns, and may hold
  several patterns separated by newlines. With no FILE, standard input is searched, or `.`
  under `-r`. A file with a NUL byte is binary: a match in it is reported as `FILE: binary file
  matches` on standard error. The exit status is 0 when a line was selected, 1 when none was
  and 2 on an error, including a command-line error, unless `-q` found a match.
- **Deviations.** Conflicting options take a fixed precedence instead of last-one-wins: `-F`
  over `-E` over `-G`; `-H` over `-h`; `--no-ignore-case` over `-i`; `-A`/`-B` over `-C` over
  `-NUM`; `-I` over `-a` over `--binary-files`; `-R` over `-r` over `-d`; `-l` over `-L`.
  Patterns are translated to RE2: back references and `-P` are refused with status 2, `\<` and
  `\>` behave as `\b`, and a repetition count above 1000 is too big. Input is treated as
  UTF-8. Directory entries are searched in name order. A missing pattern is a usage error in
  gnux's form, not GNU's usage line. `GREP_COLOR` is not read, and an input file that is also
  the output is not detected. Not implemented: `-P`, `-u`.
- **Verified by.** `core/grep/grep_test.go`, `core/grep/grep_unix_test.go`,
  `cmd/grepx/grepx_test.go`.

### FR-161: groupsx

`groupsx` must print the groups a user is in, as GNU `groups` does.

```text
groupsx [OPTION]... [USERNAME]...
```

groupsx has no options of its own.

- **Behavior.** With no USERNAME, groupsx prints the names of the process's groups: the real
  group, the effective group when it differs, then the supplementary groups, each once, as
  idx -Gn does. For each USERNAME it prints 'USERNAME : ' and that user's groups from the account
  database. An unknown user is reported as 'no such user' and the rest are still printed; a group
  with no name is reported and printed as its number; either makes the exit status 1.
- **Verified by.** `core/groups/groups_test.go`, `cmd/groupsx/groupsx_test.go`.

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

### FR-172: hostidx

`hostidx` must print the numeric identifier of the current host, as GNU `hostid` does.

```text
hostidx [OPTION]
```

- **Behavior.** hostidx prints the host identifier as eight hexadecimal digits: on Linux the four
  bytes of /etc/hostid, or else glibc's derivation from the first IPv4 address of the host name (the
  address in memory order, its halves swapped); on macOS and the BSDs the kern.hostid sysctl. An
  operand is "extra operand".
- **Deviations.** Windows has no gethostid; the identifier is derived from the host's address as
  glibc does.
- **Verified by.** `core/hostid/hostid_test.go`, the GNU conformance cases in
  `cmd/testdata/conformance/more-tools.txt`.

### FR-113: hostnamex

`hostnamex` must print the name of the current host system, as GNU `hostname` does.

```text
hostnamex [NAME]
```

`hostnamex` takes no options besides `--help` and `--version`.

- **Behavior.** hostnamex prints the name of the current host, as the operating system reports it.
  GNU coreutils hostname also sets the name when given NAME (with sethostname(2), as root); gnux
  never changes system settings, so hostnamex refuses NAME with 'cannot set name to NAME:
  Operation not supported' and exits 1. Use the system's own tool (hostnamectl, scutil,
  Rename-Computer) to set it. Note that the hostname program shipped by many Linux distributions
  is not the coreutils one and has options such as -f and -I; hostnamex has none of them.
- **Verified by.** `core/hostname/hostname_test.go`, `cmd/hostnamex/hostnamex_test.go`.

### FR-114: idx

`idx` must print real and effective user and group IDs, as GNU `id` does.

```text
idx [OPTION]... [USER]...
```

| Option            | Meaning |
|-------------------|---------|
| `-a`, `--all`     | Ignore, for compatibility with other versions |
| `-Z`, `--context` | Print only the security context of the process (not supported) |
| `-g`, `--group`   | Print only the effective group ID |
| `-G`, `--groups`  | Print all group IDs |
| `-n`, `--name`    | Print a name instead of a number, for -u,-g,-G |
| `-r`, `--real`    | Print the real ID instead of the effective ID, with -u,-g,-G |
| `-u`, `--user`    | Print only the effective user ID |
| `-z`, `--zero`    | Delimit entries with NUL characters, not whitespace; not permitted in default format |

- **Behavior.** idx prints the user and group IDs of the current process, or of each USER (a name
  or a numeric ID), in GNU's default format: 'uid=1000(alice) gid=1000(alice)
  groups=1000(alice),27(sudo)', with euid= and egid= added when they differ from the real IDs. -u,
  -g and -G print only the effective user ID, the effective group ID or every group ID; -n prints
  names instead of numbers and -r the real IDs instead of the effective ones, with one of those
  three; -z ends each entry with NUL. -a is accepted and ignored, as in GNU. -Z (SELinux context)
  is not supported and fails, as GNU id does on a kernel without SELinux. An unknown USER is
  reported ('no such user') and the remaining USERs are still printed. On Windows the IDs are
  security identifiers (SIDs) and the names come from the account database.
- **Verified by.** `core/id/id_test.go`, `cmd/idx/idx_test.go`.

### FR-115: installx

`installx` must copy files into place and set their mode, owner and group, as GNU `install` does.

```text
installx [OPTION]... SOURCE... DEST
```

| Option                        | Meaning |
|-------------------------------|---------|
| `--backup[=CONTROL]`, `-b`, `-S` | Back up each existing destination (FR-167) |
| `-C`, `--compare`             | Compare content of source and destination files, and if no change to content, ownership, and permissions, do not modify the destination at all |
| `-c`, `--copy`                | Ignored; for compatibility with old Unix versions of install |
| `-D`, `--create-leading`      | Create all leading components of DEST except the last, or all components of --target-directory, then copy SOURCE to DEST |
| `-d`, `--directory`           | Treat all arguments as directory names; create all components of the specified directories |
| `-g`, `--group`               | Set group ownership, instead of process' current group, to GROUP |
| `-m`, `--mode`                | Set permission mode (as in chmod) to MODE, instead of rwxr-xr-x |
| `-T`, `--no-target-directory` | Treat DEST as a normal file |
| `-o`, `--owner`               | Set ownership (super-user only) to OWNER |
| `-p`, `--preserve-timestamps` | Apply access/modification times of SOURCE files to corresponding destination files |
| `-s`, `--strip`               | Strip symbol tables (not supported: warns and installs unstripped) |
| `-t`, `--target-directory`    | Copy all SOURCE arguments into DIRECTORY |
| `-v`, `--verbose`             | Print the name of each created file or directory |

- **Behavior.** installx copies SOURCE to DEST, or several SOURCE files into the existing
  directory DEST (or into -t DIRECTORY), and then sets each copy's permission mode (755 unless -m
  gives another, octal or symbolic) and, with -o and -g, its owner and group. An existing DEST is
  removed and replaced, not rewritten, so a running program keeps its copy. With -d every operand
  is a directory to create, with its missing parents; the mode applies to the named directories,
  and parents get 755 minus the umask. -D creates the leading directories of DEST (or the -t
  directory) first. -T treats DEST as a file even if it is a directory. -C leaves a destination
  alone when it already has the same content, mode, owner and group. -p gives each copy its
  source's modification time (the access time is set to the same value). -v reports each file and
  directory. -s is accepted, but stripping symbol tables needs a strip program, so it only warns
  that files are installed unstripped. -c is accepted and ignored, as in GNU. -D also answers to
  the gnux long name --create-leading. A symbolic -m starts from no permissions (u=rwx,go=rx is
  755), as GNU install does. With a backup (FR-167) an existing DEST is renamed to its backup
  name instead of being removed.
- **Verified by.** `core/install/install_test.go`, `cmd/installx/installx_test.go`.

### FR-116: joinx

`joinx` must join lines of two files on a common field, as GNU `join` does.

```text
joinx [OPTION]... FILE1 FILE2
```

| Option                    | Meaning |
|---------------------------|---------|
| `--check-order`           | Check that the input is correctly sorted, even if all input lines are pairable |
| `-e`, `--empty`           | Replace missing input fields with EMPTY |
| `-1`, `--field1`          | Join on this FIELD of file 1 |
| `-2`, `--field2`          | Join on this FIELD of file 2 |
| `--header`                | Treat the first line in each file as field headers, print them without trying to pair them |
| `-i`, `--ignore-case`     | Ignore differences in case when comparing fields |
| `-j`, `--join-field`      | Equivalent to '-1 FIELD -2 FIELD' |
| `--nocheck-order`         | Do not check that the input is correctly sorted |
| `-v`, `--only-unpaired`   | Like -a FILENUM, but suppress joined output lines |
| `-o`, `--output-format`   | Obey FORMAT while constructing output line |
| `-t`, `--separator`       | Use CHAR as input and output field separator |
| `-a`, `--unpaired`        | Also print unpairable lines from file FILENUM, where FILENUM is 1 or 2 |
| `-z`, `--zero-terminated` | Line delimiter is NUL, not newline |

- **Behavior.** joinx prints, for each pair of input lines with identical join fields, one line
  made of the join field followed by the remaining fields of both lines. By default the join field
  is the first, fields are separated by blanks (leading blanks are ignored) and output fields by a
  single space. Either FILE may be - for standard input, but not both. Both files must be sorted
  on the join field, as sortx (or LC_ALL=C sort) sorts them; lines are compared byte by byte, as
  in the C locale, whatever LANG says. -1 FIELD and -2 FIELD pick the join field of each file and
  -j FIELD both. -a FILENUM also prints the unpairable lines of that file, and -v FILENUM prints
  only them. -o FORMAT lists the output fields as FILENUM.FIELD or 0 (the join field), separated
  by commas or blanks; -o auto takes the number of fields from the first line of each file. -e
  EMPTY fills missing fields. -t CHAR separates fields on input and output with CHAR, and -t ''
  makes the whole line one field. -i ignores case when comparing join fields. --header pairs the
  first lines of the files as headers without checking their order. By default joinx warns once
  per file when it finds a line out of order after an unpairable line and then exits with status
  1; --check-order fails at the first line out of order, and --nocheck-order never checks. -z
  makes NUL the line delimiter. When both --check-order and --nocheck-order are given,
  --check-order wins. GNU's obsolescent -j1 FIELD and -j2 FIELD forms are not accepted; use -1 and
  -2.
- **Verified by.** `core/join/join_test.go`, `cmd/joinx/joinx_test.go`.

### FR-117: killx

`killx` must send a signal to processes, or list signals, as GNU `kill` does.

```text
killx [-s SIGNAL | -SIGNAL] PID...
```

| Option           | Meaning |
|------------------|---------|
| `-l`, `--list`   | List signal names, or convert signal names to/from numbers |
| `-n`, `--number` | Same as -s, with a NUMBER |
| `-s`, `--signal` | Specify the name or number of the signal to be sent |
| `-L`, `--table`  | Print a table of signal information (also -t) |

- **Behavior.** killx sends a signal, TERM by default, to each process PID. The signal is given as
  -s SIGNAL, -n NUMBER, or -SIGNAL directly (-9, -KILL, -SIGKILL), by name in any case with or
  without the SIG prefix, or by number; signal 0 sends nothing and only checks that the process
  exists. A PID that cannot be signaled is reported and the remaining PIDs are still tried. -l
  prints the signal names, or converts each operand between a name and a number (an exit status
  above 128 is taken as 128 plus a signal, so 'killx -l 137' prints KILL); -L (or -t) prints a
  table of numbers, names and descriptions. Unlike GNU kill, killx reads options wherever they
  appear before '--', so 'killx 1234 -9' sends KILL to 1234 rather than signaling process group 9;
  a negative PID (a process group, or -1 for every process) must follow '--'. Giving two signals
  is an error. On Windows, which has no signals, only KILL and TERM can be sent, both by
  terminating the process, and 0 checks that it exists; any other signal fails with 'Operation not
  supported'.
- **Verified by.** `core/kill/kill_test.go`, `cmd/killx/killx_test.go`.

### FR-118: linkx

`linkx` must call the link function to create a hard link to a file, as GNU `link` does.

```text
linkx FILE1 FILE2
```

`linkx` takes no options besides `--help` and `--version`.

- **Behavior.** linkx creates FILE2 as a hard link to the existing FILE1: a second name for the
  same file, sharing its contents and metadata. It is the bare link(2) system call, with no
  options: it never overwrites FILE2, never makes a symbolic link, and never places the link
  inside a directory. Exactly two operands are required. lnx does the same with options; linkx is
  for scripts that want exactly one system call and its error.
- **Verified by.** `core/link/link_test.go`, `cmd/linkx/linkx_test.go`.

### FR-59: lnx

`lnx` must make hard and symbolic links, as GNU `ln` does.

```text
lnx [OPTION]... TARGET... [LINK_NAME|DIRECTORY]
```

| Option                             | Meaning                                                                  |
|------------------------------------|--------------------------------------------------------------------------|
| `--backup[=CONTROL]`, `-b`, `-S`   | Back up each existing destination (FR-167)                               |
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
  `-T`, are usage errors. Backups follow FR-167: a backup replaces the destination as `-f`
  would, and only a hard link is then checked against its own name.
- **Deviations.** Option order is not visible, so `-i` overrides `-f` and `-L` overrides `-P`.
  On Windows, creating symbolic links needs Developer Mode or an elevated prompt. `-b` also
  answers to the long name `--backup-default`. Not implemented: `-d`, `-F` and `--directory`.
- **Verified by.** `core/ln/ln_test.go`, `cmd/lnx/lnx_test.go`.

### FR-162: lognamex

`lognamex` must print the user's login name, as GNU `logname` does.

```text
lognamex [OPTION]...
```

lognamex has no options of its own.

- **Behavior.** lognamex prints the name the user logged in as, unchanged by su or sudo, and
  takes no operands. With no login name it reports 'no login name' and exits 1. Without cgo it
  cannot call getlogin, so it does what glibc does: on Linux the login user the kernel records
  (/proc/self/loginuid), then on every Unix the login record of the terminal on standard input.
  On macOS and the BSDs a process with no terminal therefore has no login name, where GNU
  logname would read the kernel session. On Windows it prints the session's user without the
  domain.
- **Verified by.** `core/logname/logname_test.go`, `cmd/lognamex/lognamex_test.go`.

### FR-60: lsx

`lsx` must list directory contents, as GNU `ls` does.

```text
lsx [OPTION]... [FILE]...
```

| Option                          | Meaning                                                                 |
|---------------------------------|-------------------------------------------------------------------------|
| `-a`, `--all`                   | Do not ignore entries starting with `.`                                 |
| `-A`, `--almost-all`            | Do not list implied `.` and `..`                                        |
| `--author`                      | With `-l`, print the author of each file                                |
| `-b`, `--escape`                | Print C-style escapes for nongraphic characters                         |
| `-B`, `--ignore-backups`        | Do not list implied entries ending with `~`                             |
| `--block-size=SIZE`             | With `-l` and `-s`, scale sizes by SIZE (`K`, `1M`, `KB`, `human-readable`, `si`) |
| `-c`                            | Use the status change time, for sorting and for display                 |
| `-C`                            | List entries by columns                                                 |
| `--color[=WHEN]`                | Color the output: `always` (when WHEN is omitted), `auto` or `never`    |
| `-d`, `--directory`             | List directories themselves, not their contents                         |
| `-D`, `--dired`                 | Generate output for Emacs' dired mode (implies `-l`)                    |
| `-F`, `--classify`              | Append an indicator (one of `*/=@` or a vertical bar) to entries        |
| `--file-type`                   | Likewise, except do not append `*`                                      |
| `--format=WORD`                 | across, commas, horizontal, long, single-column, verbose, vertical      |
| `--full-time`                   | Like `-l --time-style=full-iso`                                         |
| `-g`                            | Like `-l`, but do not list owner                                        |
| `--group-directories-first`     | Group directories before files                                          |
| `-G`, `--no-group`              | In a long listing, do not print group names                             |
| `-h`, `--human-readable`        | With `-l` and `-s`, print sizes like 1K 234M 2G                         |
| `--si`                          | Likewise, but use powers of 1000, not 1024                              |
| `-H`, `--dereference-command-line` | Follow symbolic links listed on the command line                     |
| `--hide=PATTERN`                | Do not list implied entries matching PATTERN (overridden by `-a`, `-A`) |
| `--hyperlink[=WHEN]`            | Hyperlink file names: `always` (when WHEN is omitted), `auto` or `never` |
| `-i`, `--inode`                 | Print the index number of each file                                     |
| `-I`, `--ignore=PATTERN`        | Do not list implied entries matching PATTERN                            |
| `--indicator-style=WORD`        | none, slash (`-p`), file-type (`--file-type`), classify (`-F`)          |
| `-k`, `--kibibytes`             | Count `-s` blocks and totals in 1024-byte units                         |
| `-l`                            | Use a long listing format                                               |
| `-L`, `--dereference`           | Show information for the file a symbolic link references                |
| `-m`                            | Fill width with a comma-separated list of entries                       |
| `-n`, `--numeric-uid-gid`       | Like `-l`, but list numeric user and group IDs                          |
| `-N`, `--literal`               | Print entry names without quoting                                       |
| `-o`                            | Like `-l`, but do not list group information                            |
| `-p`                            | Append `/` to directories                                               |
| `-q`, `--hide-control-chars`    | Print `?` instead of nongraphic characters                              |
| `-Q`, `--quote-name`            | Enclose entry names in double quotes                                    |
| `--quoting-style=WORD`          | literal, shell, shell-always, shell-escape, shell-escape-always, c, c-maybe, escape, locale, clocale |
| `-r`, `--reverse`               | Reverse order while sorting                                             |
| `-R`, `--recursive`             | List subdirectories recursively                                         |
| `-s`, `--size`                  | Print the allocated size of each file, in blocks                        |
| `--show-control-chars`          | Print nongraphic characters as they are                                 |
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
| `--zero`                        | End each output line with NUL, not newline                              |

- **Behavior.** Operands that are not directories are listed first, then each directory under a
  header. On a terminal names are arranged in columns sized to the window; when output is piped
  or redirected there is one name per line. In the long format, timestamps from the last six
  months show the time of day and older ones the year. Entries that cannot be examined are
  still listed, with `?` in place of the unknown fields. When several options of one family are
  given (the formats, the sort keys, `-a` and `-A`, the time selectors) the last one on the
  command line wins, as in GNU. The exit status is 0 on success, 1 for minor problems, and 2
  for serious trouble such as an operand that cannot be accessed or a bad command line; an
  invalid word for an option (`--sort=x`, `--format=x`) exits 1, as GNU's argmatch does.
- **Quoting.** Names are quoted as gnulib's quotearg quotes them, in the style `--quoting-style`
  names (or `-N`, `-b`, `-Q`), else the one the `QUOTING_STYLE` variable names (an invalid one
  is warned about and ignored), else shell-escape on a terminal and literal otherwise (dir and
  vdir: escape). The escape style also escapes a space, and with `-F` or `--file-type` the
  indicator characters; a directory header also escapes `:`. In the long format and in columns,
  with the shell, shell-escape and c-maybe styles, an unquoted name gets a leading space when
  some name in the listing is quoted, so that the names line up. `?` replaces a nonprintable
  character with `-q`, or on a terminal unless `--show-control-chars`, in the literal and shell
  styles; dir and vdir never do so without `-q`. A control character left in a name takes no
  column.
- **Other options.** `--zero` ends each line (and each name in `-m`) with NUL, leaves headers'
  `:` lines and the blank line between directories as they are, and implies `-1` (unless `-l`),
  `-N`, `--show-control-chars` and no color, each undone by a later option. `--block-size` sets
  both the long format's size unit and the `-s` unit; without it (or `-h`, `--si`),
  `LS_BLOCK_SIZE` and then `BLOCK_SIZE` set both, and otherwise sizes are in bytes and blocks in
  1024 bytes (512 under `POSIXLY_CORRECT`; `-k` makes it 1024 again); an invalid value exits 2
  with `invalid --block-size argument`. `--author` adds a column after the group holding the
  owner. `--hyperlink` wraps each name, and each directory header, in an OSC 8 link to
  `file://HOST/PATH`, PATH absolute with symbolic links resolved and percent-encoded; a quoted
  name lined up with unquoted ones keeps its quotes outside the link. `-D` implies `-l` and
  turns `--hyperlink` off, both undone by a later option; with the long format it indents each
  line by two spaces and ends the output with `//DIRED//` and `//SUBDIRED//` (the byte offsets
  of every name and directory header) and `//DIRED-OPTIONS// --quoting-style=STYLE`; without
  the long format it does nothing, and with `--zero` it is refused (`--dired and --zero are
  incompatible`, status 2).
- **Deviations.** Names are sorted by plain byte comparison (the C locale), whatever the
  environment's locale. On Windows, which has no POSIX owner, group, inode or block count, those
  columns hold placeholders. The single-letter options `-c`, `-C`, `-g`, `-l`, `-m`, `-o`,
  `-p`, `-S`, `-t`, `-u`, `-U`, `-v`, `-x`, `-X` and `-1` also answer to long names of gnux's
  own (`--ctime`, `--columns`, `--long-no-owner`, `--long`, `--commas`, `--long-no-group`,
  `--slash`, `--sort-size`, `--sort-time`, `--atime`, `--unsorted`, `--sort-version`,
  `--across`, `--sort-extension`, `--one-per-line`). A multibyte name is read as UTF-8 in
  every locale, so in the C locale a printable non-ASCII character is shown where GNU escapes
  its bytes. Not implemented: `-f`, `-Z` and `--context`.
- **Verified by.** `core/ls/ls_test.go`, `cmd/lsx/lsx_test.go`.

### FR-119: md5sumx

`md5sumx` must print or check MD5 (RFC 1321) (128-bit) checksums, as GNU `md5sum` does.

```text
md5sumx [OPTION]... [FILE]...
```

| Option             | Meaning |
|--------------------|---------|
| `-b`, `--binary`   | Read in binary mode |
| `-c`, `--check`    | Read checksums from the FILEs and check them |
| `--ignore-missing` | Don't fail or report status for missing files |
| `--quiet`          | Don't print OK for each successfully verified file |
| `--status`         | Don't output anything, status code shows success |
| `--strict`         | Exit non-zero for improperly formatted checksum lines |
| `--tag`            | Create a BSD-style checksum |
| `-t`, `--text`     | Read in text mode (default) |
| `-w`, `--warn`     | Warn about improperly formatted checksum lines |
| `-z`, `--zero`     | End each output line with NUL, not newline, and disable file name escaping |

- **Behavior.** md5sumx prints or checks MD5 (RFC 1321) message digests (128 bits). For each FILE
  it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text mode, '*'
  with -b), and the file name. With no FILE, or when FILE is -, it reads standard input. Files are
  always read as bytes, so -b and -t change only the mark; when both are given, -t wins. --tag
  prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a
  backslash, newline or carriage return is escaped and its line starts with a backslash, unless -z
  ends lines with NUL instead. -c reads such lines (in either style) from the FILEs and checks
  each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches,
  unreadable files and improperly formatted lines; --quiet drops the OK lines, --status prints
  nothing, --ignore-missing skips listed files that do not exist, -w warns about each improperly
  formatted line and --strict fails because of them. The exit status is 0 when everything was read
  (and, with -c, matched), 1 otherwise. MD5 (RFC 1321) is broken for security purposes: use it to
  detect accidental corruption, not tampering, and prefer sha256sumx or b2sumx otherwise.
- **Verified by.** `core/md5sum/md5sum_test.go`, `cmd/md5sumx/md5sumx_test.go`.

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

### FR-173: mkfifox

`mkfifox` must create named pipes, as GNU `mkfifo` does.

```text
mkfifox [OPTION]... NAME...
```

| Option                  | Meaning |
|-------------------------|---------|
| `-m`, `--mode=MODE`     | Set file permission bits to MODE, not a=rw - umask |
| `-Z`, `--context[=CTX]` | Accepted and ignored: gnux does not support SELinux |

- **Behavior.** mkfifox creates a FIFO at each NAME, with mode a=rw less the umask, or exactly MODE
  (octal or symbolic, applied to a=rw; a mode with bits beyond the permissions is refused with "mode
  must specify only file permission bits"). A NAME that cannot be created is reported ("cannot
  create fifo") and the rest are still made. --context=CTX warns as GNU does on a kernel without
  SELinux.
- **Deviations.** Windows has no named pipes in its file system: every NAME fails with "Operation
  not supported".
- **Verified by.** `core/mkfifo/mkfifo_test.go`, `core/mkfifo/mkfifo_unix_test.go`,
  `cmd/mkfifox/mkfifox_test.go`, `core/internal/mode/mode_test.go`, the GNU conformance cases in
  `cmd/testdata/conformance/more-tools.txt`.

### FR-174: mknodx

`mknodx` must create block or character special files, or named pipes, as GNU `mknod` does.

```text
mknodx [OPTION]... NAME TYPE [MAJOR MINOR]
```

| Option                  | Meaning |
|-------------------------|---------|
| `-m`, `--mode=MODE`     | Set file permission bits to MODE, not a=rw - umask |
| `-Z`, `--context[=CTX]` | Accepted and ignored: gnux does not support SELinux |

- **Behavior.** mknodx creates NAME as a block device (b), a character device (c or u) with MAJOR
  and MINOR, or a FIFO (p), judging TYPE by its first letter. Device numbers are decimal, octal with
  a leading 0, or hexadecimal with 0x, and must fit in 32 bits. Missing and extra operands are usage
  errors in GNU's words, including its second lines "Special files require major and minor device
  numbers." and "Fifos do not have major and minor device numbers.".
- **Deviations.** Windows has no special files: every TYPE fails with "Operation not supported".
- **Verified by.** `core/mknod/mknod_test.go`, `cmd/mknodx/mknodx_test.go`, the GNU conformance
  cases in `cmd/testdata/conformance/more-tools.txt`.

### FR-120: mktempx

`mktempx` must create a temporary file or directory, safely, and print its name, as GNU `mktemp` does.

```text
mktempx [OPTION]... [TEMPLATE]
```

| Option              | Meaning |
|---------------------|---------|
| `-p`, `--base-dir`  | Use DIR as the directory, like --tmpdir=DIR |
| `-d`, `--directory` | Create a directory, not a file |
| `-u`, `--dry-run`   | Do not create anything; merely print a name (unsafe) |
| `-t`, `--in-tmpdir` | Interpret TEMPLATE as a single file name component, relative to $TMPDIR, else DIR (-p), else the temporary directory (deprecated) |
| `-q`, `--quiet`     | Suppress diagnostics about file/dir-creation failure |
| `--suffix`          | Append SUFF to TEMPLATE; SUFF must not contain a slash |
| `--tmpdir`          | Interpret TEMPLATE relative to DIR; if DIR is not given, use $TMPDIR if set, else the temporary directory |

- **Behavior.** mktempx creates a new, empty file (or a directory with -d) whose name is TEMPLATE
  with its run of trailing X's replaced by random letters and digits, and prints the name.
  TEMPLATE must contain at least 3 consecutive X's at the end of the name; whatever follows the
  last X is kept as a suffix, and --suffix adds one explicitly. With no TEMPLATE, tmp.XXXXXXXXXX
  is used and the file is created in the temporary directory. A relative TEMPLATE is resolved
  against the current directory unless -p DIR or --tmpdir[=DIR] is given; then it goes under DIR,
  else $TMPDIR, else the system temporary directory (/tmp on Unix, the user's temporary folder on
  Windows), and must not be absolute. The file is created with mode 600 and the directory with
  mode 700, and creation is atomic: an existing name is never reused. -u only prints a name that
  did not exist at the time, which is unsafe. -q suppresses the diagnostic when creation fails. -t
  is the deprecated form: TEMPLATE is a single name component under $TMPDIR, else -p DIR, else the
  temporary directory. -p and -t also answer to the gnux long names --base-dir and --in-tmpdir.
- **Verified by.** `core/mktemp/mktemp_test.go`, `cmd/mktempx/mktempx_test.go`.

### FR-62: mvx

`mvx` must move and rename files, as GNU `mv` does.

```text
mvx [OPTION]... SOURCE... DIRECTORY
```

| Option                             | Meaning                                                                 |
|------------------------------------|-------------------------------------------------------------------------|
| `--backup[=CONTROL]`, `-b`, `-S`   | Back up each existing destination (FR-167)                              |
| `-f`, `--force`                    | Do not prompt before overwriting                                        |
| `-i`, `--interactive`              | Prompt before overwrite                                                 |
| `-n`, `--no-clobber`               | Do not overwrite an existing file                                       |
| `--no-copy`                        | Do not copy if renaming fails                                           |
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
  unless `-f` is given. `--no-copy` refuses a move across filesystems with `cannot move 'SOURCE'
  to 'DEST': Invalid cross-device link` instead of copying, and leaves the source in place.
  Backups follow FR-167.
- **Deviations.** Option order is not visible, so `-n` overrides `-i`, which overrides `-f`.
  `-b` also answers to the long name `--backup-default`. Not implemented: `-Z` and
  `--context`, `--exchange`, `--debug`.
- **Verified by.** `core/mv/mv_test.go`, `cmd/mvx/mvx_test.go`.

### FR-121: nicex

`nicex` must run a program with modified scheduling priority, as GNU `nice` does.

```text
nicex [OPTION] [COMMAND [ARG]...]
```

| Option               | Meaning |
|----------------------|---------|
| `-n`, `--adjustment` | Add integer N to the niceness (default 10) |

- **Behavior.** nicex runs COMMAND with its niceness adjusted by 10, or by -n ADJUSTMENT, so that
  it gets less (or, with a negative adjustment, more) CPU time than other processes. Niceness
  ranges from -20 (most favorable) to 19 (least); the result is clamped to that range. With no
  COMMAND it prints the current niceness. The obsolete forms -ADJUSTMENT and --ADJUSTMENT
  (negative) are accepted as in GNU: 'nicex -5 make' and 'nicex --5 make'. Options must come
  before COMMAND. Lowering the niceness needs privilege; without it nicex warns 'cannot set
  niceness' and runs the command anyway, as GNU nice does. The exit status is the command's, or
  125 when nicex itself failed, 126 when COMMAND was found but could not be run and 127 when it
  was not found. nicex starts COMMAND as a child and sets its niceness as it starts, where GNU
  nice changes its own and then becomes COMMAND. On Windows, which has no niceness, the adjustment
  picks the nearest priority class instead (idle, below normal, normal, above normal or high;
  never realtime), so 'nicex make' really does lower the build's priority there too, and nicex
  with no COMMAND prints the niceness its own class corresponds to.
- **Verified by.** `cmd/nicex/nicex_test.go`.

### FR-122: nlx

`nlx` must number lines of files, as GNU `nl` does.

```text
nlx [OPTION]... [FILE]...
```

| Option                         | Meaning |
|--------------------------------|---------|
| `-b`, `--body-numbering`       | Use STYLE for numbering body lines (a, t, n or pBRE; default t) |
| `-f`, `--footer-numbering`     | Use STYLE for numbering footer lines (default n) |
| `-h`, `--header-numbering`     | Use STYLE for numbering header lines (default n) |
| `-l`, `--join-blank-lines`     | Group of NUMBER empty lines counted as one (default 1) |
| `-i`, `--line-increment`       | Line number increment at each line (default 1) |
| `-p`, `--no-renumber`          | Do not reset line numbers for each section |
| `-n`, `--number-format`        | Insert line numbers according to FORMAT (ln, rn or rz; default rn) |
| `-s`, `--number-separator`     | Add STRING after (possible) line number (default TAB) |
| `-w`, `--number-width`         | Use NUMBER columns for line numbers (default 6) |
| `-d`, `--section-delimiter`    | Use CC for logical page delimiters (default \:) |
| `-v`, `--starting-line-number` | First line number for each section (default 1) |

- **Behavior.** nlx writes each FILE to standard output with line numbers added. With no FILE, or
  when FILE is -, it reads standard input; several FILEs are numbered as one stream. By default
  non-empty lines are numbered from 1, right-aligned in a 6-column field followed by a tab, and
  empty lines get the same width of blanks so the text stays aligned. -b STYLE chooses which lines
  are numbered: a (all), t (non-empty, the default), n (none) or pBRE (those matching the POSIX
  basic regular expression BRE). -n FORMAT is ln (left-aligned), rn (right-aligned) or rz
  (zero-padded); -w sets the width, -s the separator, -v the first number and -i the increment
  (both may be zero or negative). -l N with -b a numbers only every Nth of a run of empty lines.
  Input may be split into logical pages of header, body and footer sections by lines holding only
  '\:\:\:', '\:\:' or '\:' (change the two characters with -d; an empty -d turns sections off);
  each delimiter prints as an empty line, -h and -f choose the numbering of headers and footers
  (none by default), and numbering restarts at every header unless -p. Every output line ends with
  a newline, a last line without one included.
- **Verified by.** `core/nl/nl_test.go`, `cmd/nlx/nlx_test.go`.

### FR-123: nohupx

`nohupx` must run a command immune to hangups, with output to a non-terminal, as GNU `nohup` does.

```text
nohupx COMMAND [ARG]...
```

`nohupx` takes no options besides `--help` and `--version`.

- **Behavior.** nohupx runs COMMAND so that it keeps running after you log out or close the
  terminal. On Unix it ignores the hangup signal, as GNU nohup does. Any of the standard streams
  that is a terminal is replaced: standard input by an unusable /dev/null, standard output by the
  file nohup.out, appended to and created readable only by you, in the current directory or, if
  that fails, in $HOME, and standard error by standard output; nohupx says which on standard
  error. Streams that are already redirected are left alone. The exit status is COMMAND's, or 125
  when nohupx itself failed, 126 when COMMAND was found but could not be run and 127 when it was
  not found. GNU nohup does nothing useful on Windows; nohupx starts COMMAND detached from the
  console and in a process group of its own, so closing the console window or pressing Ctrl-C in
  it no longer ends COMMAND. nohupx runs COMMAND as a child and waits for it, where GNU nohup
  becomes COMMAND; to get your prompt back, start it in the background with your shell.
- **Verified by.** `core/nohup/nohup_test.go`, `cmd/nohupx/nohupx_test.go`.

### FR-124: nprocx

`nprocx` must print the number of processing units available, as GNU `nproc` does.

```text
nprocx [OPTION]...
```

| Option     | Meaning |
|------------|---------|
| `--all`    | Print the number of installed processors |
| `--ignore` | If possible, exclude N processing units |

- **Behavior.** nprocx prints the number of processing units available to the current process. As
  in GNU nproc, the OMP_NUM_THREADS environment variable (its first value, when it is a
  comma-separated list) replaces that count when it is a positive number, and OMP_THREAD_LIMIT
  caps the result. --all prints the number of installed processors instead and ignores both
  variables. --ignore=N subtracts N, but the result is never less than 1. gnux has no cgo, so both
  counts come from Go's runtime.NumCPU: the processors this process may run on. A machine with
  processors taken offline, or a process restricted to fewer of them, prints the same number with
  and without --all.
- **Verified by.** `core/nproc/nproc_test.go`, `cmd/nprocx/nprocx_test.go`.

### FR-175: numfmtx

`numfmtx` must convert numbers to and from human-readable strings, as GNU `numfmt` does.

```text
numfmtx [OPTION]... [NUMBER]...
```

| Option                    | Meaning |
|---------------------------|---------|
| `--debug`                 | Print warnings about invalid input |
| `-d`, `--delimiter=X`     | Use X instead of whitespace for field delimiter |
| `--field=FIELDS`          | Replace the numbers in these input fields (default 1) |
| `--format=FORMAT`         | Use printf style floating-point FORMAT |
| `--from=UNIT`             | Auto-scale input numbers to UNITs: none, auto, si, iec, iec-i |
| `--from-unit=N`           | Specify the input unit size |
| `--grouping`              | Use locale-defined grouping of digits (none in the C locale) |
| `--header[=N]`            | Print (without converting) the first N header lines |
| `--invalid=MODE`          | Failure mode for invalid numbers: abort, fail, warn, ignore |
| `--padding=N`             | Pad the output to N characters |
| `--round=METHOD`          | up, down, from-zero (default), towards-zero, nearest |
| `--suffix=SUFFIX`         | Add SUFFIX to output numbers, and accept it on input |
| `--to=UNIT`               | Auto-scale output numbers to UNITs: none, si, iec, iec-i |
| `--to-unit=N`             | The output unit size |
| `-z`, `--zero-terminated` | Line delimiter is NUL, not newline |

- **Behavior.** numfmtx is a port of GNU numfmt: the same number syntax, scaling, rounding
  (simple_round), field splitting (a field keeps its leading blanks and fields are rejoined with one
  space), automatic padding, --format parsing (including its quirks with %%), suffix handling and
  diagnostics. An invalid number stops the run with status 2 (--invalid=abort), or under fail, warn
  and ignore is printed unchanged and the run goes on, ending with 2, 0 and 0. A missing final
  newline stays missing. It is a line filter: each line is answered as it arrives.
- **Deviations.** Arithmetic has the 64-bit mantissa of x86-64's long double, so above 18 digits it
  behaves as GNU on x86-64, not on arm64. --grouping and ' in --format add nothing (the C locale).
- **Verified by.** `core/numfmt/numfmt_test.go`, `cmd/numfmtx/numfmtx_test.go`,
  `cmd/robust_test.go`, the GNU conformance cases in `cmd/testdata/conformance/more-tools.txt`.

### FR-125: odx

`odx` must dump files in octal and other formats, as GNU `od` does.

```text
odx [OPTION]... [FILE]...
```

| Option                             | Meaning |
|------------------------------------|---------|
| `-A`, `--address-radix`            | Output format for file offsets; RADIX is one of [doxn], for Decimal, Octal, Hex or None |
| `-c`, `--characters`               | Same as -t c, select printable characters or backslash escapes |
| `-s`, `--decimal-2-bytes`          | Same as -t d2, select decimal 2-byte units |
| `-i`, `--decimal-ints`             | Same as -t dI, select decimal ints |
| `-l`, `--decimal-longs`            | Same as -t dL, select decimal longs |
| `--endian`                         | Swap input bytes according the specified ORDER: little or big |
| `-f`, `--floats`                   | Same as -t fF, select floats |
| `-t`, `--format`                   | Select output format or formats; TYPE is a, c, d, f, o, u or x with an optional size, and z |
| `-x`, `--hex-2-bytes`              | Same as -t x2, select hexadecimal 2-byte units |
| `-a`, `--named-characters`         | Same as -t a, select named characters, ignoring high-order bit |
| `-o`, `--octal-2-bytes`            | Same as -t o2, select octal 2-byte units |
| `-b`, `--octal-bytes`              | Same as -t o1, select octal bytes |
| `-v`, `--output-duplicates`        | Do not use * to mark line suppression |
| `-N`, `--read-bytes`               | Limit dump to BYTES input bytes |
| `-j`, `--skip-bytes`               | Skip BYTES input bytes first |
| `-S`, `--strings`                  | Show only NUL terminated strings of at least BYTES printable characters |
| `-d`, `--unsigned-decimal-2-bytes` | Same as -t u2, select unsigned decimal 2-byte units |
| `-w`, `--width`                    | Output BYTES bytes per output line |

- **Behavior.** odx writes an unambiguous representation of its input, by default as two-byte
  octal words, sixteen bytes per line, each line starting with the input offset in octal. The
  FILEs are read one after another as one input; with no FILE, or when FILE is -, it reads
  standard input. -t TYPE selects the format and may be given more than once, each format getting
  a line of its own per block, in the order given: a (named characters), c (printable characters
  or backslash escapes), d, o, u and x (signed decimal, octal, unsigned decimal and hexadecimal)
  with an optional size in bytes (1, 2, 4, 8, or C, S, I, L), and f (floating point, size 4 or 8,
  or F, D, L); a trailing z adds the printable characters of the line between > and <. The
  traditional letters -a, -b, -c, -d, -f, -i, -l, -o, -s and -x are shorthands for -t a, o1, c,
  u2, fF, dI, dL, o2, d2 and x2. -A RADIX picks the offset base (d, o, x, or n for none), -j BYTES
  skips input, -N BYTES limits it, -w BYTES sets the bytes per line, and --endian chooses the byte
  order of multi-byte values. Unless -v is given, a run of lines identical to the one before is
  shown as a single '*'. -S N prints the NUL-terminated strings of at least N printable characters
  instead of a dump. BYTES may be decimal, octal with a leading 0, or hexadecimal with 0x,
  followed by a multiplier: b 512, k or K 1024, kB 1000, M, MB, G and so on. Differences from GNU
  od: characters are interpreted in the C locale, so -c shows every byte above 127 in octal; long
  double (fL) is read as an eight-byte double; -S and -w need a value (GNU's bare -S means -S3 and
  bare -w means -w32), because an optional value could not also be attached as in -w8; the
  traditional offset operand ([+]OFFSET[.][b]) and --traditional are not implemented.
- **Verified by.** `core/od/od_test.go`, `cmd/odx/odx_test.go`.

### FR-126: pastex

`pastex` must merge lines of files, as GNU `paste` does.

```text
pastex [OPTION]... [FILE]...
```

| Option                    | Meaning |
|---------------------------|---------|
| `-d`, `--delimiters`      | Reuse characters from LIST instead of TABs |
| `-s`, `--serial`          | Paste one file at a time instead of in parallel |
| `-z`, `--zero-terminated` | Line delimiter is NUL, not newline |

- **Behavior.** pastex writes lines made of the corresponding lines of each FILE, separated by
  TABs, to standard output. With no FILE, or when FILE is -, it reads standard input; every -
  reads the same standard input, so 'pastex - -' joins pairs of lines. A file that runs out of
  lines contributes empty fields while the others continue. -s pastes one file at a time instead,
  turning each file into a single line. -d LIST uses the characters of LIST as delimiters instead
  of TAB, cycling through them and starting again from the first on every output line (and, with
  -s, for every file). LIST understands the escapes \n, \t, \\, \b, \f, \r, \v and \0, the empty
  delimiter. -z makes NUL the line delimiter. Delimiters are single bytes: a multibyte character
  in LIST is used one byte at a time.
- **Verified by.** `core/paste/paste_test.go`, `cmd/pastex/pastex_test.go`.

### FR-176: pathchkx

`pathchkx` must check whether file names are valid or portable, as GNU `pathchk` does.

```text
pathchkx [OPTION]... NAME...
```

| Option          | Meaning |
|-----------------|---------|
| `-p`            | Check for most POSIX systems |
| `-P`            | Check for empty names and leading "-" |
| `--portability` | Check for all POSIX systems (-p -P) |

- **Behavior.** pathchkx checks each NAME and reports those that could not be created: by default
  against the file system (a component that is not a searchable directory, or a name or component
  longer than pathconf allows); with -p against _POSIX_PATH_MAX (256) and _POSIX_NAME_MAX (14) and
  the portable character set; with -P also an empty name and a component starting with "-". Every
  NAME is checked; the messages are GNU's ("limit 14 exceeded by length 16 of file name component
  ...", "non-portable character ...").
- **Deviations.** -p and -P also answer to the gnux long names --basic and --extra. On Windows the
  limits are NTFS's.
- **Verified by.** `core/pathchk/pathchk_test.go`, `cmd/pathchkx/pathchkx_test.go`, the GNU
  conformance cases in `cmd/testdata/conformance/more-tools.txt`.

### FR-177: pinkyx

`pinkyx` must show who is logged in, or what is known about users, as GNU `pinky` does.

```text
pinkyx [OPTION]... [USER]...
```

| Option     | Meaning |
|------------|---------|
| `-l`       | Produce long format output for the specified USERs |
| `-b`       | Omit the home directory and shell in long format |
| `-h`       | Omit the project file in long format |
| `-p`       | Omit the plan file in long format |
| `-s`       | Do short format output (the default) |
| `-f`       | Omit the line of column headings |
| `-w`       | Omit the full name |
| `-i`       | Omit the full name and remote host |
| `-q`       | Omit the full name, remote host and idle time |
| `--lookup` | Canonicalize host names via DNS |

- **Behavior.** pinkyx prints one line per user session from the login records, as whox reads them
  (FR-147), only for the USERs given if any, in GNU's columns: login, full name (GECOS, & expanded,
  cut to 19), the message mark (* when the terminal is not group-writable), terminal, idle time,
  login time (%b %e %H:%M) and host. -l prints each USER's login and real name, home directory and
  shell, and ~/.project and ~/.plan; an unknown user is "???". The last of -s and -l wins.
- **Deviations.** The long option names of its letters (--long, --omit-heading, ...) are gnux's own.
  The shell is read from /etc/passwd, blank for users kept elsewhere and on Windows.
- **Verified by.** `core/pinky/pinky_test.go`, `cmd/pinkyx/pinkyx_test.go`, the GNU conformance
  cases in `cmd/testdata/conformance/more-tools.txt`.

### FR-127: printenvx

`printenvx` must print all or part of the environment, as GNU `printenv` does.

```text
printenvx [OPTION]... [VARIABLE]...
```

| Option         | Meaning |
|----------------|---------|
| `-0`, `--null` | End each output line with NUL, not newline |

- **Behavior.** printenvx prints the value of each environment VARIABLE, one per line. With no
  VARIABLE it prints the whole environment as NAME=value lines. A VARIABLE that is not set prints
  nothing, not even an empty line, and makes the exit status 1, with no message; one that is set
  to the empty string prints an empty line. Names are matched exactly, including their case, on
  every platform. -0 ends each item with NUL instead of a newline, so values containing newlines
  survive.
- **Verified by.** `core/printenv/printenv_test.go`, `cmd/printenvx/printenvx_test.go`.

### FR-94: printfx

`printfx` must format and print data, as GNU `printf` does.

```text
printfx FORMAT [ARGUMENT]...
```

- **Behavior.** It has no options of its own; a leading `--` is skipped. FORMAT is printed with
  backslash escapes interpreted (`\"`, `\\`, `\a`, `\b`, `\c`, `\e`, `\f`, `\n`, `\r`, `\t`,
  `\v`, `\NNN`, `\xHH`, `\uHHHH`, `\UHHHHHHHH`) and each conversion (`%d %i %o %u %x %X %f %F %e
  %E %g %G %a %A %c %s %b %q %%`, with the flags `- + space # 0`, a width and a precision,
  either of which may be `*`) replaced by the next ARGUMENT. FORMAT is reused while it consumes
  arguments and some remain; a missing argument reads as an empty string or zero. A numeric
  argument may be decimal, `0x` hexadecimal, `0` octal (integers) or a quote followed by a
  character (its code). An argument that is not a number is diagnosed (`expected a numeric
  value`, `value not completely converted`, `Numerical result out of range`), printed as far as
  it converted, and makes the exit status 1 without stopping the output. An invalid conversion
  or escape is diagnosed and stops at once with status 1. When FORMAT consumes no argument and
  some were given, `warning: ignoring excess arguments, starting with 'X'` is printed and the
  status is unaffected. No FORMAT is the usage error `missing operand`. `\c`, in FORMAT or in a
  `%b` argument, ends all output.
- **Deviations.** Floating-point values are held in double precision rather than long double.
  `\c` ends output with the status the run already had rather than always 0. `%b` and `%q`
  accept a width and precision.
- **Verified by.** `core/printf/printf_test.go`, `cmd/printfx/printfx_test.go`.

### FR-178: prx

`prx` must paginate or columnate files for printing, as GNU `pr` does.

```text
prx [OPTION]... [FILE]...
```

| Option                               | Meaning |
|--------------------------------------|---------|
| `+FIRST[:LAST]`, `--pages`           | Begin and stop printing with those pages |
| `-COLUMN`, `--columns`               | Output COLUMN columns, printed down unless -a |
| `-a`, `--across`                     | Print columns across rather than down |
| `-c`, `--show-control-chars`         | Hat notation (^G) and octal backslash notation |
| `-d`, `--double-space`               | Double space the output |
| `-D`, `--date-format=FORMAT`         | Use FORMAT for the header date |
| `-e`, `--expand-tabs[=CHAR[WIDTH]]`  | Expand input CHARs (TABs) to tab WIDTH (8) |
| `-F`, `-f`, `--form-feed`            | Use form feeds to separate pages |
| `-h`, `--header=HEADER`              | Use HEADER instead of the file name |
| `-i`, `--output-tabs[=CHAR[WIDTH]]`  | Replace spaces with CHARs (TABs) to tab WIDTH (8) |
| `-J`, `--join-lines`                 | Merge full lines |
| `-l`, `--length=PAGE_LENGTH`         | Set the page length (66) |
| `-m`, `--merge`                      | Print all files in parallel, one per column |
| `-n`, `--number-lines[=SEP[DIGITS]]` | Number lines |
| `-N`, `--first-line-number=NUMBER`   | Start counting with NUMBER |
| `-o`, `--indent=MARGIN`              | Offset each line with MARGIN spaces |
| `-r`, `--no-file-warnings`           | Omit warnings when a file cannot be opened |
| `-s`, `--separator[=CHAR]`           | Separate columns by a single character |
| `-S`, `--sep-string[=STRING]`        | Separate columns by STRING |
| `-t`, `--omit-header`                | Omit page headers and trailers |
| `-T`, `--omit-pagination`            | Also eliminate form feeds set in input files |
| `-v`, `--show-nonprinting`           | Octal backslash notation |
| `-w`, `--width=PAGE_WIDTH`           | Page width (72) for multiple text-column output only |
| `-W`, `--page-width=PAGE_WIDTH`      | Page width (72), always truncating lines |

- **Behavior.** prx is a port of GNU pr's page layout: headers and footers, columns down with the
  last page balanced, across, and side by side (-m), truncation, separators, tabification, line
  numbering, form feeds in the input, page ranges, and the reinterpretation of -s and -w next to -S,
  -W and the column options. It reads its own command line as GNU's getopt does, digit options and
  attached optional values included. A file that cannot be opened is reported (unless -r) and makes
  the status 1; a read error ends the run.
- **Deviations.** The header's date is in the C locale's words.
- **Verified by.** `core/pr/pr_test.go`, `cmd/prx/prx_test.go`, the GNU conformance cases in
  `cmd/testdata/conformance/more-tools.txt`.

### FR-179: ptxx

`ptxx` must produce a permuted index of the words in a text, as GNU `ptx` does.

```text
ptxx [OPTION]... [INPUT]...
ptxx -G [OPTION]... [INPUT [OUTPUT]]
```

| Option                           | Meaning |
|----------------------------------|---------|
| `-A`, `--auto-reference`         | Output automatically generated references |
| `-G`, `--traditional`            | Behave more like System V ptx |
| `-F`, `--flag-truncation=STRING` | Use STRING for flagging line truncations |
| `-M`, `--macro-name=STRING`      | Macro name to use instead of xx |
| `-O`, `--format=roff`            | Generate output as roff directives |
| `-R`, `--right-side-refs`        | Put references at right, not counted in -w |
| `-S`, `--sentence-regexp=REGEXP` | For end of lines or end of sentences |
| `-T`, `--format=tex`             | Generate output as TeX directives |
| `-W`, `--word-regexp=REGEXP`     | Use REGEXP to match each keyword |
| `-b`, `--break-file=FILE`        | Word break characters in this FILE |
| `-f`, `--ignore-case`            | Fold lower case to upper case for sorting |
| `-g`, `--gap-size=NUMBER`        | Gap size in columns between output fields |
| `-i`, `--ignore-file=FILE`       | Read ignore word list from FILE |
| `-o`, `--only-file=FILE`         | Read only word list from FILE |
| `-r`, `--references`             | First field of each line is a reference |
| `-t`, `--typeset-mode`           | Not implemented, as in GNU |
| `-w`, `--width=NUMBER`           | Output width in columns, reference excluded |

- **Behavior.** ptxx is a port of GNU ptx: words and contexts are found, sorted by keyword, and laid
  out with the same field widths, truncation marks and wrap-around head and tail fields, in the
  dumb, roff and TeX formats. -F, -S and -W take GNU ptx's backslash escapes. -O, -T and --format:
  the last one wins.
- **Deviations.** Regular expressions are in the Emacs syntax GNU ptx uses, run by RE2 in
  leftmost-longest mode; back references are refused.
- **Verified by.** `core/ptx/ptx_test.go`, `cmd/ptxx/ptxx_test.go`, the GNU conformance cases in
  `cmd/testdata/conformance/more-tools.txt`.

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

### FR-128: readlinkx

`readlinkx` must print resolved symbolic links or canonical file names, as GNU `readlink` does.

```text
readlinkx [OPTION]... FILE...
```

| Option                          | Meaning |
|---------------------------------|---------|
| `-f`, `--canonicalize`          | Canonicalize by following every symlink in every component recursively; all but the last component must exist |
| `-e`, `--canonicalize-existing` | Canonicalize by following every symlink in every component recursively; all components must exist |
| `-m`, `--canonicalize-missing`  | Canonicalize by following every symlink in every component recursively, without requirements on existence |
| `-n`, `--no-newline`            | Do not output the trailing delimiter |
| `-q`, `--quiet`                 | Suppress most error messages (on by default) |
| `-s`, `--silent`                | Suppress most error messages (on by default) |
| `-v`, `--verbose`               | Report error messages |
| `-z`, `--zero`                  | End each output line with NUL, not newline |

- **Behavior.** readlinkx prints the value of each FILE, which must be a symbolic link: the target
  exactly as it was stored, not resolved. With -f, -e or -m it prints the canonical name instead:
  absolute, with every symbolic link and every '.' and '..' component resolved. -f requires every
  component but the last to exist, -e requires all of them, and -m requires none. As in GNU
  readlink, failures are silent by default (-q and -s say so explicitly) and only the exit status
  reports them; -v prints a diagnostic for each. -n drops the trailing newline, and is ignored
  with a warning when there is more than one FILE; -z ends each name with NUL. GNU readlink lets
  the last of -f, -e and -m win; readlinkx cannot see option order, so it picks the most demanding
  (-e, then -f, then -m), and -v wins over -q and -s.
- **Verified by.** `core/readlink/readlink_test.go`, `cmd/readlinkx/readlinkx_test.go`.

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

### FR-150: sedx

`sedx` must filter and transform text with a script, as GNU `sed` does.

```text
sedx [OPTION]... {script-only-if-no-other-script} [input-file]...
```

| Option                            | Meaning                                                              |
|-----------------------------------|----------------------------------------------------------------------|
| `-n`, `--quiet`, `--silent`       | Suppress automatic printing of the pattern space                     |
| `-e SCRIPT`, `-f SCRIPT-FILE`     | Add commands to the script; both repeat and keep their order          |
| `-E`, `-r`, `--regexp-extended`   | Use extended regular expressions                                     |
| `-i[SUFFIX]`, `--in-place[=SUFFIX]` | Edit files in place, keeping a backup when SUFFIX is given          |
| `--follow-symlinks`               | Follow symbolic links when editing in place                          |
| `-s`, `--separate`                | Treat files separately rather than as one stream                     |
| `-z`, `--null-data`               | Lines end in NUL, not newline                                        |
| `-u`, `--unbuffered`              | Flush the output after every line                                    |
| `-l N`, `--line-length=N`         | Line-wrap length for the `l` command                                 |
| `--posix`                         | Print nothing when `n` or `N` finds no next line                     |
| `--sandbox`                       | Refuse the `r`, `R`, `w` and `W` commands                            |
| `-b`, `--binary`                  | Accepted; has no effect                                              |

- **Behavior.** The script language is GNU's: addresses (`N`, `$`, `/regex/` with `I` and `M`,
  `first~step`, `addr1,addr2`, `addr1,+N`, `addr1,~N`, `0,/regex/`, `!`), blocks, and the
  commands `s y d D p P n N g G h H x a i c = l q Q r R w W b t T : z F #`. The `s` command
  takes the flags `g`, `p`, a number, `i`, `m` and `w FILE`, and its replacement takes `&`,
  `\1` to `\9`, `\n` and the case conversions `\L \U \l \u \E`. The whole script is parsed
  before any input is read. In-place editing writes a temporary file beside the original and
  renames it, keeping the permission bits. The exit status is 0 on success, 1 for a script,
  regular expression or option error, 2 when an input file could not be read (the others are
  still processed), 4 for an I/O error, and the status given to `q` or `Q`.
- **Deviations.** The `e` command and the `e` flag of `s` are refused: sedx does not run
  commands. Regular expressions are translated to RE2: back references inside a pattern are
  refused at parse time, `\<` and `\>` behave as `\b`, `\w`, `\s` and `\b` are ASCII, and a
  repetition count above 1000 is too big. Patterns must be valid UTF-8. `--posix` changes only
  `n` and `N` at end of input. `-u` flushes per line but still reads in blocks. In-place
  editing does not keep the owner, group or extended attributes. Not implemented: `--debug`.
- **Verified by.** `core/sed/sed_test.go`, `cmd/sedx/sedx_test.go`.

### FR-93: seqx

`seqx` must print a sequence of numbers, as GNU `seq` does.

```text
seqx [OPTION]... LAST
seqx [OPTION]... FIRST LAST
seqx [OPTION]... FIRST INCREMENT LAST
```

| Option                  | Meaning                                                              |
|-------------------------|----------------------------------------------------------------------|
| `-f`, `--format=FORMAT` | Use a printf-style floating-point FORMAT (`%e`, `%f`, `%g` or `%a`)  |
| `-s`, `--separator=STRING` | Use STRING to separate numbers (default: newline)                 |
| `-w`, `--equal-width`   | Equalize width by padding with leading zeros                         |

- **Behavior.** FIRST and INCREMENT default to 1; a negative INCREMENT counts down, and nothing
  is printed when FIRST is already past LAST. Options come first; an argument that starts with
  `-` followed by a digit or `.` is a number, so negative operands need no `--`. Without `-f`,
  numbers have as many decimal places as FIRST and INCREMENT (after any exponent), and the
  arithmetic is exact (`seqx 0.1 0.1 0.3` ends at `0.3`); integers that fit in 64 bits take a
  fast path. LAST may be `inf`, and the sequence then ends quietly when its reader goes away.
  `-w` cannot be combined with `-f`. Errors are usage errors in GNU's words: `missing operand`,
  `extra operand 'X'`, `invalid floating point argument: 'X'`, `invalid 'not-a-number'
  argument: 'X'`, `invalid Zero increment value: 'X'`, and for `-f`, `format 'F' has no %
  directive`, `has too many % directives`, `has unknown %C directive` or `ends in %`.
- **Deviations.** `-f` values are formatted from double precision rather than long double. An
  infinite FIRST or INCREMENT is refused as an invalid floating point argument.
- **Verified by.** `core/seq/seq_test.go`, `cmd/seqx/seqx_test.go`.

### FR-129: sha1sumx

`sha1sumx` must print or check SHA-1 (FIPS 180-4) (160-bit) checksums, as GNU `sha1sum` does.

```text
sha1sumx [OPTION]... [FILE]...
```

| Option             | Meaning |
|--------------------|---------|
| `-b`, `--binary`   | Read in binary mode |
| `-c`, `--check`    | Read checksums from the FILEs and check them |
| `--ignore-missing` | Don't fail or report status for missing files |
| `--quiet`          | Don't print OK for each successfully verified file |
| `--status`         | Don't output anything, status code shows success |
| `--strict`         | Exit non-zero for improperly formatted checksum lines |
| `--tag`            | Create a BSD-style checksum |
| `-t`, `--text`     | Read in text mode (default) |
| `-w`, `--warn`     | Warn about improperly formatted checksum lines |
| `-z`, `--zero`     | End each output line with NUL, not newline, and disable file name escaping |

- **Behavior.** sha1sumx prints or checks SHA-1 (FIPS 180-4) message digests (160 bits). For each
  FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text mode,
  '*' with -b), and the file name. With no FILE, or when FILE is -, it reads standard input. Files
  are always read as bytes, so -b and -t change only the mark; when both are given, -t wins. --tag
  prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a
  backslash, newline or carriage return is escaped and its line starts with a backslash, unless -z
  ends lines with NUL instead. -c reads such lines (in either style) from the FILEs and checks
  each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches,
  unreadable files and improperly formatted lines; --quiet drops the OK lines, --status prints
  nothing, --ignore-missing skips listed files that do not exist, -w warns about each improperly
  formatted line and --strict fails because of them. The exit status is 0 when everything was read
  (and, with -c, matched), 1 otherwise. SHA-1 (FIPS 180-4) is broken for security purposes: use it
  to detect accidental corruption, not tampering, and prefer sha256sumx or b2sumx otherwise.
- **Verified by.** `core/sha1sum/sha1sum_test.go`, `cmd/sha1sumx/sha1sumx_test.go`.

### FR-130: sha224sumx

`sha224sumx` must print or check SHA-224 (FIPS 180-4) (224-bit) checksums, as GNU `sha224sum` does.

```text
sha224sumx [OPTION]... [FILE]...
```

| Option             | Meaning |
|--------------------|---------|
| `-b`, `--binary`   | Read in binary mode |
| `-c`, `--check`    | Read checksums from the FILEs and check them |
| `--ignore-missing` | Don't fail or report status for missing files |
| `--quiet`          | Don't print OK for each successfully verified file |
| `--status`         | Don't output anything, status code shows success |
| `--strict`         | Exit non-zero for improperly formatted checksum lines |
| `--tag`            | Create a BSD-style checksum |
| `-t`, `--text`     | Read in text mode (default) |
| `-w`, `--warn`     | Warn about improperly formatted checksum lines |
| `-z`, `--zero`     | End each output line with NUL, not newline, and disable file name escaping |

- **Behavior.** sha224sumx prints or checks SHA-224 (FIPS 180-4) message digests (224 bits). For
  each FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text
  mode, '*' with -b), and the file name. With no FILE, or when FILE is -, it reads standard input.
  Files are always read as bytes, so -b and -t change only the mark; when both are given, -t wins.
  --tag prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a
  backslash, newline or carriage return is escaped and its line starts with a backslash, unless -z
  ends lines with NUL instead. -c reads such lines (in either style) from the FILEs and checks
  each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches,
  unreadable files and improperly formatted lines; --quiet drops the OK lines, --status prints
  nothing, --ignore-missing skips listed files that do not exist, -w warns about each improperly
  formatted line and --strict fails because of them. The exit status is 0 when everything was read
  (and, with -c, matched), 1 otherwise.
- **Verified by.** `core/sha224sum/sha224sum_test.go`, `cmd/sha224sumx/sha224sumx_test.go`.

### FR-131: sha256sumx

`sha256sumx` must print or check SHA-256 (FIPS 180-4) (256-bit) checksums, as GNU `sha256sum` does.

```text
sha256sumx [OPTION]... [FILE]...
```

| Option             | Meaning |
|--------------------|---------|
| `-b`, `--binary`   | Read in binary mode |
| `-c`, `--check`    | Read checksums from the FILEs and check them |
| `--ignore-missing` | Don't fail or report status for missing files |
| `--quiet`          | Don't print OK for each successfully verified file |
| `--status`         | Don't output anything, status code shows success |
| `--strict`         | Exit non-zero for improperly formatted checksum lines |
| `--tag`            | Create a BSD-style checksum |
| `-t`, `--text`     | Read in text mode (default) |
| `-w`, `--warn`     | Warn about improperly formatted checksum lines |
| `-z`, `--zero`     | End each output line with NUL, not newline, and disable file name escaping |

- **Behavior.** sha256sumx prints or checks SHA-256 (FIPS 180-4) message digests (256 bits). For
  each FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text
  mode, '*' with -b), and the file name. With no FILE, or when FILE is -, it reads standard input.
  Files are always read as bytes, so -b and -t change only the mark; when both are given, -t wins.
  --tag prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a
  backslash, newline or carriage return is escaped and its line starts with a backslash, unless -z
  ends lines with NUL instead. -c reads such lines (in either style) from the FILEs and checks
  each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches,
  unreadable files and improperly formatted lines; --quiet drops the OK lines, --status prints
  nothing, --ignore-missing skips listed files that do not exist, -w warns about each improperly
  formatted line and --strict fails because of them. The exit status is 0 when everything was read
  (and, with -c, matched), 1 otherwise.
- **Verified by.** `core/sha256sum/sha256sum_test.go`, `cmd/sha256sumx/sha256sumx_test.go`.

### FR-132: sha384sumx

`sha384sumx` must print or check SHA-384 (FIPS 180-4) (384-bit) checksums, as GNU `sha384sum` does.

```text
sha384sumx [OPTION]... [FILE]...
```

| Option             | Meaning |
|--------------------|---------|
| `-b`, `--binary`   | Read in binary mode |
| `-c`, `--check`    | Read checksums from the FILEs and check them |
| `--ignore-missing` | Don't fail or report status for missing files |
| `--quiet`          | Don't print OK for each successfully verified file |
| `--status`         | Don't output anything, status code shows success |
| `--strict`         | Exit non-zero for improperly formatted checksum lines |
| `--tag`            | Create a BSD-style checksum |
| `-t`, `--text`     | Read in text mode (default) |
| `-w`, `--warn`     | Warn about improperly formatted checksum lines |
| `-z`, `--zero`     | End each output line with NUL, not newline, and disable file name escaping |

- **Behavior.** sha384sumx prints or checks SHA-384 (FIPS 180-4) message digests (384 bits). For
  each FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text
  mode, '*' with -b), and the file name. With no FILE, or when FILE is -, it reads standard input.
  Files are always read as bytes, so -b and -t change only the mark; when both are given, -t wins.
  --tag prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a
  backslash, newline or carriage return is escaped and its line starts with a backslash, unless -z
  ends lines with NUL instead. -c reads such lines (in either style) from the FILEs and checks
  each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches,
  unreadable files and improperly formatted lines; --quiet drops the OK lines, --status prints
  nothing, --ignore-missing skips listed files that do not exist, -w warns about each improperly
  formatted line and --strict fails because of them. The exit status is 0 when everything was read
  (and, with -c, matched), 1 otherwise.
- **Verified by.** `core/sha384sum/sha384sum_test.go`, `cmd/sha384sumx/sha384sumx_test.go`.

### FR-133: sha512sumx

`sha512sumx` must print or check SHA-512 (FIPS 180-4) (512-bit) checksums, as GNU `sha512sum` does.

```text
sha512sumx [OPTION]... [FILE]...
```

| Option             | Meaning |
|--------------------|---------|
| `-b`, `--binary`   | Read in binary mode |
| `-c`, `--check`    | Read checksums from the FILEs and check them |
| `--ignore-missing` | Don't fail or report status for missing files |
| `--quiet`          | Don't print OK for each successfully verified file |
| `--status`         | Don't output anything, status code shows success |
| `--strict`         | Exit non-zero for improperly formatted checksum lines |
| `--tag`            | Create a BSD-style checksum |
| `-t`, `--text`     | Read in text mode (default) |
| `-w`, `--warn`     | Warn about improperly formatted checksum lines |
| `-z`, `--zero`     | End each output line with NUL, not newline, and disable file name escaping |

- **Behavior.** sha512sumx prints or checks SHA-512 (FIPS 180-4) message digests (512 bits). For
  each FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text
  mode, '*' with -b), and the file name. With no FILE, or when FILE is -, it reads standard input.
  Files are always read as bytes, so -b and -t change only the mark; when both are given, -t wins.
  --tag prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a
  backslash, newline or carriage return is escaped and its line starts with a backslash, unless -z
  ends lines with NUL instead. -c reads such lines (in either style) from the FILEs and checks
  each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches,
  unreadable files and improperly formatted lines; --quiet drops the OK lines, --status prints
  nothing, --ignore-missing skips listed files that do not exist, -w warns about each improperly
  formatted line and --strict fails because of them. The exit status is 0 when everything was read
  (and, with -c, matched), 1 otherwise.
- **Verified by.** `core/sha512sum/sha512sum_test.go`, `cmd/sha512sumx/sha512sumx_test.go`.

### FR-180: shredx

`shredx` must overwrite a file to hide its contents, and optionally delete it, as GNU `shred` does.

```text
shredx [OPTION]... FILE...
```

| Option                 | Meaning |
|------------------------|---------|
| `-f`, `--force`        | Change permissions to allow writing if necessary |
| `-n`, `--iterations=N` | Overwrite N times instead of 3 |
| `--random-source=FILE` | Get random bytes from FILE |
| `-s`, `--size=N`       | Shred this many bytes |
| `-u`                   | Deallocate and remove the file after overwriting (--remove=wipesync) |
| `--remove[=HOW]`       | unlink, wipe or wipesync |
| `-v`, `--verbose`      | Show progress |
| `-x`, `--exact`        | Do not round file sizes up to the next full block |
| `-z`, `--zero`         | Add a final overwrite with zeros |

- **Behavior.** shredx overwrites each FILE with GNU's choice of passes (random passes spread among
  fixed bit patterns, genpattern), rounding a regular file up to a whole block and overwriting a
  small file first at its own size, flushing each pass, and with -u truncates it, renames it to ever
  shorter names of zeros (skipping names that exist) and removes it, reporting each step with -v
  exactly as GNU does. A file that cannot be opened ("failed to open for writing") is reported and
  the rest are shredded. - shreds standard output.
- **Deviations.** The order of the fixed patterns is drawn from Go's generator (or --random-source)
  rather than GNU's ISAAC. A bad block ends the file's shredding instead of being skipped.
- **Verified by.** `core/shred/shred_test.go`, `cmd/shredx/shredx_test.go`, the GNU conformance
  cases in `cmd/testdata/conformance/more-tools.txt`.

### FR-134: shufx

`shufx` must generate random permutations, as GNU `shuf` does.

```text
shufx [OPTION]... [FILE]
```

| Option                    | Meaning |
|---------------------------|---------|
| `-e`, `--echo`            | Treat each ARG as an input line |
| `-n`, `--head-count`      | Output at most COUNT lines |
| `-i`, `--input-range`     | Treat each number LO-HI as an input line |
| `-o`, `--output`          | Write result to FILE instead of standard output |
| `--random-source`         | Get random bytes from FILE |
| `-r`, `--repeat`          | Output lines can be repeated |
| `-z`, `--zero-terminated` | Line delimiter is NUL, not newline |

- **Behavior.** shufx writes a random permutation of its input lines to standard output. The input
  is the lines of FILE, or of standard input when FILE is absent or -; with -e, each ARG operand
  is an input line instead (shufx -e [OPTION]... [ARG]...); with -i LO-HI, the numbers LO through
  HI are (shufx -i LO-HI [OPTION]...). -n COUNT writes at most COUNT lines (given more than once,
  the smallest COUNT wins, as in GNU shuf). -r draws with replacement, so lines may repeat, and
  with no -n it writes forever until its reader goes away. -o FILE writes to FILE instead of
  standard output; it is opened only after the input has been read, so FILE may also be the input.
  --random-source=FILE takes the random bytes from FILE, so the same FILE always gives the same
  order; without it the order comes from a generator seeded by the operating system's random
  source. -z uses NUL rather than newline as the line delimiter. Differences from GNU shuf: the
  permutation drawn from a given --random-source differs from the one GNU shuf draws from the same
  bytes, though it is just as reproducible.
- **Verified by.** `core/shuf/shuf_test.go`, `cmd/shufx/shufx_test.go`.

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

### FR-88: sortx

`sortx` must sort, merge or check the lines of text files, as GNU `sort` does.

```text
sortx [OPTION]... [FILE]...
```

| Option                          | Meaning                                                                        |
|---------------------------------|--------------------------------------------------------------------------------|
| `-b`, `--ignore-leading-blanks` | Ignore leading blanks                                                          |
| `-d`, `--dictionary-order`      | Consider only blanks and alphanumeric characters                               |
| `-f`, `--ignore-case`           | Fold lower case to upper case characters                                       |
| `-g`, `--general-numeric-sort`  | Compare according to general numerical value                                   |
| `-i`, `--ignore-nonprinting`    | Consider only printable characters                                             |
| `-M`, `--month-sort`            | Compare (unknown) < JAN < ... < DEC                                            |
| `-h`, `--human-numeric-sort`    | Compare human readable numbers (2K, 1G)                                        |
| `-n`, `--numeric-sort`          | Compare according to string numerical value                                    |
| `-R`, `--random-sort`           | Shuffle, but group identical keys                                              |
| `--random-source=FILE`          | Get random bytes from FILE                                                     |
| `-r`, `--reverse`               | Reverse the result of comparisons                                              |
| `--sort=WORD`                   | general-numeric, human-numeric, month, numeric, random or version              |
| `-V`, `--version-sort`          | Natural sort of (version) numbers within text                                  |
| `-c`, `--check[=WHEN]`          | Check for sorted input and report the first disorder; WHEN: diagnose-first, quiet, silent |
| `-C`                            | Check for sorted input without reporting                                       |
| `--files0-from=F`               | Read input from the files named, NUL-terminated, in file F; `-` is standard input |
| `-k`, `--key=KEYDEF`            | Sort via a key, `F[.C][OPTS][,F[.C][OPTS]]`; may be repeated                   |
| `-m`, `--merge`                 | Merge already sorted files; do not sort                                        |
| `-o`, `--output=FILE`           | Write the result to FILE instead of standard output                            |
| `-s`, `--stable`                | Disable the last-resort comparison                                             |
| `-t`, `--field-separator=SEP`   | Use SEP instead of the non-blank to blank transition                           |
| `-u`, `--unique`                | Output only the first of an equal run                                          |
| `-z`, `--zero-terminated`       | Line delimiter is NUL, not newline                                             |
| `-S`, `-T`, `--parallel`, `--batch-size`, `--compress-program` | Accepted and ignored                          |

- **Behavior.** With no key the whole line is compared byte by byte. A key with no ordering
  letters inherits the global options; one with letters uses only its own. Without `-t`, a field
  is a run of blanks followed by non-blanks, so it keeps its leading blanks unless `b` is given.
  When the keys compare equal, the whole lines are compared, unless `-s` or `-u`. `-u` keeps the
  first line of each equal run. Ordering options that contradict each other (`-n` with `-g`,
  `-d` with `-n`, ...) are refused with `options '-gn' are incompatible`. With `-c`, the first
  line out of order is reported as `sortx: FILE:LINE: disorder: TEXT` and the status is 1; `-C`
  is silent. Every input is checked to be readable before `-o` opens its file, and the file is
  emptied only after all input is read, so `-o` may name an input. Unlike most tools, trouble
  with any input or the output stops the run with status 2, as in GNU `sort`; a command-line
  mistake is also status 2. `-R` orders by the MD5 of 16 random bytes followed by the key, as GNU
  `sort` does, so equal keys stay together; `--random-source=FILE` takes those bytes from FILE,
  and the same FILE gives the same order as GNU `sort`. A FILE shorter than 16 bytes is reported
  as `'FILE': end of file`. Without `-R` the source is never read.
- **Deviations.** Comparison always uses the C locale (byte order, ASCII case folding and month
  names, `.` as the decimal point). `-S`, `-T`, `--parallel`, `--batch-size` and
  `--compress-program` are accepted and ignored; sorting is done in memory. When `-c` and `-C`
  are both given, `-c` wins in either order. Not implemented: `--debug`.
- **Verified by.** `core/sort/sort_test.go`, `cmd/sortx/sortx_test.go`.

### FR-135: splitx

`splitx` must split a file into pieces, as GNU `split` does.

```text
splitx [OPTION]... [FILE [PREFIX]]
```

| Option                      | Meaning |
|-----------------------------|---------|
| `--additional-suffix`       | Append an additional SUFFIX to file names |
| `-b`, `--bytes`             | Put SIZE bytes per output file |
| `-e`, `--elide-empty-files` | Do not generate empty output files with '-n' |
| `-x`, `--hex-suffixes`      | Use hex suffixes starting at 0, not alphabetic; --hex-suffixes=FROM starts at FROM |
| `-C`, `--line-bytes`        | Put at most SIZE bytes of records per output file |
| `-l`, `--lines`             | Put NUMBER lines/records per output file |
| `-n`, `--number`            | Generate CHUNKS output files: N, K/N, l/N, l/K/N, r/N or r/K/N |
| `-d`, `--numeric-suffixes`  | Use numeric suffixes starting at 0, not alphabetic; --numeric-suffixes=FROM starts at FROM |
| `-t`, `--separator`         | Use SEP instead of newline as the record separator; '\0' (zero) specifies the NUL character |
| `-a`, `--suffix-length`     | Generate suffixes of length N (default 2) |
| `-u`, `--unbuffered`        | Immediately copy input to output with '-n r/...' (always the case here) |
| `--verbose`                 | Print a diagnostic just before each output file is opened |

- **Behavior.** splitx copies FILE (standard input when FILE is - or absent) into files named
  PREFIX (x by default) plus a suffix: xaa, xab, and so on, 1000 lines per file. -l N puts N lines
  in each file (the obsolete -N form too), -b SIZE puts SIZE bytes, and -C SIZE as many whole
  lines as fit in SIZE bytes, splitting only a line longer than that. SIZE takes the suffixes K,
  M, G, ... (powers of 1024) and KB, MB, ... (powers of 1000). -n CHUNKS makes a fixed number of
  files instead: N files of nearly equal size (the first ones a byte longer when the size does not
  divide), l/N without splitting lines (each line goes to the chunk it starts in), or r/N dealing
  lines out in turn; K/N, l/K/N and r/K/N write only the Kth chunk to standard output. To size the
  chunks of a pipe, splitx first copies it to a temporary file, as GNU does. Suffixes are two
  letters, and when they run out they grow (xyz is followed by xzaaa) so the names still sort in
  order; -a N fixes their length, -d and -x make them decimal or hexadecimal,
  --numeric-suffixes=FROM and --hex-suffixes=FROM start them at FROM, and --additional-suffix
  appends a fixed string such as .csv. -t SEP splits lines on SEP instead of newline ('\0' for
  NUL), -e skips empty files in the -n modes, and --verbose reports each file as it is created.
  splitx refuses to overwrite FILE with one of its pieces. Only one way of splitting may be given.
  GNU's --filter, which pipes each piece through a shell command, is not implemented; -u is
  accepted and has no effect, since output is never held back. The digits of two obsolete -N
  options in a row run together (-5 -3 reads as -53).
- **Verified by.** `core/split/split_test.go`, `cmd/splitx/splitx_test.go`.

### FR-136: statx

`statx` must display file or file system status, as GNU `stat` does.

```text
statx [OPTION]... FILE...
```

| Option                | Meaning |
|-----------------------|---------|
| `--cached`            | Specify how to use cached attributes (always, never or default); accepted, no effect |
| `-L`, `--dereference` | Follow links |
| `-f`, `--file-system` | Display file system status instead of file status |
| `-c`, `--format`      | Use the specified FORMAT instead of the default; output a newline after each use of FORMAT |
| `--printf`            | Like --format, but interpret backslash escapes, and do not output a mandatory trailing newline |
| `-t`, `--terse`       | Print the information in terse form |

- **Behavior.** statx prints everything the system records about each FILE: its type, size,
  blocks, device, inode, links, permissions, owner and group, and its access, modification, change
  and birth times. A symbolic link is described itself unless -L follows it. -f describes the file
  system holding FILE instead: its type, block sizes, and total, free and available blocks and
  inodes. -t prints the same fields on one line, for scripts. -c FORMAT prints only what FORMAT
  asks for, followed by a newline; --printf FORMAT does the same but interprets backslash escapes
  (\n, \t, \x41, \101) and adds no newline; when both are given, the last wins. FORMAT takes GNU's
  directives: %n (name), %N (quoted name, with its target for a link), %s (size), %F (file type),
  %a and %A (permissions in octal and as rwx), %u %U %g %G (owner and group), %h (links), %i
  (inode), %b %B %o (blocks, block unit, I/O size), %d %D %Hd %Ld (device), %r %R %t %T %Hr %Lr
  (device type of a special file), %m (mount point), %w %x %y %z (birth, access, modification and
  change times) and %W %X %Y %Z (the same as seconds since the epoch, which take a precision:
  %.3Y), with printf flags and widths (%-10s, %04a). With -f: %n %i %l %t %T %s %S %b %f %a %c %d.
  %N quotes the name, and a link's target, in the style `QUOTING_STYLE` names (as `lsx
  --quoting-style` takes it; an invalid one is warned about and ignored), else shell-escape-always;
  the default formats name files as they are. The device is shown as major,minor, as GNU
  stat 9 shows it. Missing data is reported honestly rather than invented: the birth time prints
  '-' where the system does not record it (Linux without statx), and on Windows, which has no
  numeric owners, %u and %g print '?'. SELinux contexts (%C) are not supported. --cached is
  accepted for compatibility and has no effect.
- **Verified by.** `core/stat/stat_test.go`, `cmd/statx/statx_test.go`.

### FR-181: stdbufx

`stdbufx` must run a command with modified buffering of its standard streams, as GNU `stdbuf` does.

```text
stdbufx OPTION... COMMAND [ARG]...
```

| Option                | Meaning |
|-----------------------|---------|
| `-i`, `--input=MODE`  | Adjust standard input stream buffering |
| `-o`, `--output=MODE` | Adjust standard output stream buffering |
| `-e`, `--error=MODE`  | Adjust standard error stream buffering |

- **Behavior.** stdbufx checks its MODEs as GNU does (L, 0, or a size with K, M, ... suffixes; -iL
  is "line buffering stdin is meaningless"; a mode is required), then runs COMMAND. A COMMAND named
  after a gnux tool runs in-process as it is. Any other COMMAND gets GNU coreutils' libstdbuf.so
  preloaded with the _STDBUF_ variables, found beside gnux or where coreutils packages install it;
  without it stdbufx fails with "failed to find 'libstdbuf.so'", as GNU does. Statuses: 125 for its
  own failures, 126 and 127 when COMMAND cannot be run or found, else COMMAND's.
- **Deviations.** A gnux tool is not given the buffer sizes asked for: it already writes each line
  as soon as its input runs dry. Windows has no library preloading.
- **Verified by.** `core/stdbuf/stdbuf_test.go`, `cmd/stdbufx/stdbufx_test.go`, the GNU conformance
  cases in `cmd/testdata/conformance/more-tools.txt`.

### FR-182: sttyx

`sttyx` must print or change terminal line settings, as GNU `stty` does.

```text
sttyx [-F DEVICE | --file=DEVICE] [SETTING]...
sttyx [-F DEVICE | --file=DEVICE] [-a|--all]
sttyx [-F DEVICE | --file=DEVICE] [-g|--save]
```

| Option                | Meaning |
|-----------------------|---------|
| `-a`, `--all`         | Print all current settings in human-readable form |
| `-g`, `--save`        | Print all current settings in a stty-readable form |
| `-F`, `--file=DEVICE` | Open and use DEVICE instead of standard input |

- **Behavior.** sttyx is a port of GNU stty over termios: the mode and control-character tables as
  GNU builds them for the platform, combination settings, speeds, window size, line discipline
  (Linux), -g output and its reading back, the listing of changed and all settings with GNU's
  wrapping, and the check that every setting was taken. Its command line is read as GNU's is, so a
  setting such as -echo is not an option.
- **Deviations.** On macOS and the BSDs the kernel-set PENDIN flag is left out of that check. When
  the Linux kernel ignores a setting, the failure is worded as "unable to perform all requested
  operations" where glibc's tcsetattr gives "Invalid argument". Windows has no termios.
- **Verified by.** `core/stty/stty_test.go`, `cmd/sttyx/sttyx_test.go`, the GNU conformance cases in
  `cmd/testdata/conformance/more-tools.txt`.

### FR-158: sumx

`sumx` must print the checksum and block count of each file, as GNU `sum` does.

```text
sumx [OPTION]... [FILE]...
```

| Option         | Meaning |
|----------------|---------|
| `-r`           | Use the BSD sum algorithm (the default), use 1K blocks |
| `-s`, `--sysv` | Use the System V sum algorithm, use 512-byte blocks |

- **Behavior.** sumx prints a 16-bit checksum of each FILE and its size in blocks, followed by
  the name when FILE operands are given, as GNU sum 9 does. With no FILE, or when FILE is -, it
  reads standard input. By default it computes the BSD checksum (five digits, 1024-byte
  blocks); -s computes the System V checksum (512-byte blocks). They are the checksums of
  cksumx -a bsd and -a sysv. -s wins over -r in any order. A FILE that cannot be read is
  reported and the rest are still summed.
- **Verified by.** `core/sum/sum_test.go`, `cmd/sumx/sumx_test.go`, `cmd/robust_test.go`
  (bounded memory).

### FR-183: syncx

`syncx` must flush cached writes to persistent storage, as GNU `sync` does.

```text
syncx [OPTION] [FILE]...
```

| Option                | Meaning |
|-----------------------|---------|
| `-d`, `--data`        | Sync only file data, no unneeded metadata |
| `-f`, `--file-system` | Sync the file systems that contain the files |

- **Behavior.** syncx flushes every file system, or each FILE (fsync), its data only (-d, fdatasync)
  or its whole file system (-f, syncfs). -d needs a FILE and -d with -f is refused, without the
  --help hint, as in GNU. A FILE is opened read-only, or write-only when reading is refused; one
  that cannot be opened ("error opening") or flushed is reported and the rest are flushed.
- **Deviations.** Off Linux, -d flushes as fsync and -f flushes every file system. On Windows a run
  with no FILE flushes nothing.
- **Verified by.** `core/sync/sync_test.go`, `cmd/syncx/syncx_test.go`, the GNU conformance cases in
  `cmd/testdata/conformance/more-tools.txt`.

### FR-137: tacx

`tacx` must concatenate and print files in reverse, as GNU `tac` does.

```text
tacx [OPTION]... [FILE]...
```

| Option              | Meaning |
|---------------------|---------|
| `-b`, `--before`    | Attach the separator before instead of after |
| `-r`, `--regex`     | Interpret the separator as a regular expression |
| `-s`, `--separator` | Use STRING as the separator instead of newline |

- **Behavior.** tacx writes each FILE to standard output last line first. With no FILE, or when
  FILE is -, it reads standard input. Each FILE is reversed on its own, and the files are written
  in the order given. A record ends with its separator, a newline by default, so a last line
  without one is joined to the line before it in the output, as in GNU. -s STRING uses STRING as
  the separator instead, and -b attaches the separator to the start of the record that follows it
  rather than to the end of the one before. -r treats the separator as a regular expression; gnux
  uses Go's RE2 syntax (close to POSIX extended regular expressions) rather than the GNU regex
  dialect, and reports an invalid expression in Go's words. Each input is read whole into memory
  before it is written.
- **Verified by.** `core/tac/tac_test.go`, `cmd/tacx/tacx_test.go`.

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
| `--max-unchanged-stats=N`   | With `--follow=name`, reopen a FILE unchanged for N polls (default 5)          |
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
  long name `--follow-retry`. `--follow=name` checks the name on every poll, where GNU waits for
  `--max-unchanged-stats` polls without growth, so a rotation is noticed at once; N is validated
  (`invalid maximum number of unchanged stats between opens`) but changes nothing.
- **Verified by.** `core/tail/tail_test.go`, `cmd/tailx/tailx_test.go`.

### FR-92: teex

`teex` must copy standard input to standard output and to each FILE, as GNU `tee` does.

```text
teex [OPTION]... [FILE]...
```

| Option                       | Meaning                                                                    |
|------------------------------|----------------------------------------------------------------------------|
| `-a`, `--append`             | Append to the FILEs, do not overwrite                                      |
| `-i`, `--ignore-interrupts`  | Ignore interrupt signals (accepted, no effect; see Deviations)             |
| `-p`                         | Same as `--output-error=warn-nopipe`                                       |
| `--output-error[=MODE]`      | On a write error: warn, warn-nopipe (the default MODE), exit, exit-nopipe  |

- **Behavior.** Each chunk read is written at once to standard output and then to every FILE,
  in order. FILEs are created with mode 0666 masked by the umask, truncated unless `-a`. A FILE
  named `-` is a file of that name (GNU 9). A FILE that cannot be opened is reported and the
  rest still receive the data; the status is then 1. Without `--output-error` a failing output
  is reported and dropped and a closed pipe stops `teex` quietly; `warn` reports every error and
  continues, the `-nopipe` modes drop a closed pipe silently, `exit` modes stop at the first
  reported error. Reading stops once no output is left. An invalid MODE is a usage error.
- **Deviations.** `-i` is accepted but has no effect, because gnux runs tools in-process (also
  inside `gnux shell` and the playground). When `-p` and `--output-error=MODE` are both given,
  MODE wins in either order (GNU: the last one). A closed pipe in the default mode ends with
  status 0 rather than death by SIGPIPE.
- **Verified by.** `core/tee/tee_test.go`, `cmd/teex/teex_test.go`.

### FR-138: testx

`testx` must check file types and compare values, as GNU `test` does.

```text
testx EXPRESSION
```

`testx` takes no options besides `--help` and `--version`.

- **Behavior.** testx evaluates EXPRESSION and exits with status 0 when it is true and 1 when it
  is false; it prints nothing. With no EXPRESSION it is false. Strings: -n STRING (not empty), -z
  STRING (empty), STRING alone (not empty), S1 = S2 (also ==), S1 != S2, and S1 < S2 or S1 > S2,
  comparing bytes (the C locale). Integers: N1 -eq, -ne, -lt, -le, -gt or -ge N2, where an integer
  is an optional sign and decimal digits of any length, with surrounding blanks allowed. Files: -e
  (exists), -f (regular file), -d (directory), -h or -L (symbolic link, not followed), -b and -c
  (block and character devices), -p (named pipe), -S (socket), -s (size above zero), -r, -w and -x
  (readable, writable, executable by you), -u, -g and -k (set-user-ID, set-group-ID and sticky
  bits), -O and -G (owned by your effective user or group), -t FD (descriptor FD is a terminal),
  and F1 -nt, -ot or -ef F2 (newer, older, or the same file). Expressions combine with ! (not), -a
  (and, binding tighter), -o (or) and parentheses, which must be separate arguments (quote them
  from the shell). As POSIX requires, one to four arguments are read by their count first, so
  testx -n = -n compares two strings. A malformed expression is diagnosed with GNU's wording ('a':
  unary operator expected, invalid integer 'x', missing argument after '-f', extra argument 'c')
  and exits with status 2. -l STRING may stand for either integer of a comparison and means the
  length of STRING. The same tool answers to `[ EXPRESSION ]` with the closing `]` required: as
  `gnux [ ... ]`, as a program named `[`, and in the gnux shell; without the `]` it reports
  missing ']' and exits 2. `[` is not a tool of its own: it has no x-name, no reference page and
  no `--list` entry. Differences from GNU test: a lone --help or --version prints help or the
  version where GNU test would treat it as a non-empty string, and off Unix -O and -G are always
  false while -r, -w and -x are read from the permission bits.
- **Verified by.** `core/test/test_test.go`, `cmd/main_test.go` (`TestBracket`).

### FR-139: timeoutx

`timeoutx` must run a command with a time limit, as GNU `timeout` does.

```text
timeoutx [OPTION] DURATION COMMAND [ARG]...
```

| Option               | Meaning |
|----------------------|---------|
| `--foreground`       | When not running timeout directly from a shell prompt, allow COMMAND to read from the TTY and get TTY signals; in this mode, children of COMMAND will not be timed out |
| `-k`, `--kill-after` | Also send a KILL signal if COMMAND is still running this long after the initial signal was sent (DURATION) |
| `--preserve-status`  | Exit with the same status as COMMAND, even when the command times out |
| `-s`, `--signal`     | Specify the SIGNAL to be sent on timeout; SIGNAL may be a name like 'HUP' or a number |
| `-v`, `--verbose`    | Diagnose to stderr any signal sent upon timeout |

- **Behavior.** timeoutx runs COMMAND and, if it is still running after DURATION, sends it the
  TERM signal, or the signal -s names. DURATION is a number, possibly decimal or hexadecimal, with
  an optional unit: s (seconds, the default), m (minutes), h (hours) or d (days); 0 means no
  limit. -k DURATION also sends KILL if COMMAND is still running that long after the first signal.
  By default the signal goes to COMMAND's whole process group, so the processes it started stop
  too; --foreground signals COMMAND alone and leaves it in the terminal's process group, so it can
  read from the terminal. A signal sent to timeoutx is passed on to COMMAND, and -v reports each
  signal sent. The exit status is 124 when COMMAND timed out (unless --preserve-status, which
  keeps COMMAND's own status), 125 when timeoutx itself failed, 126 when COMMAND was found but
  could not be run, 127 when it was not found, and otherwise COMMAND's status (128+N when signal N
  ended it, 137 after KILL). Options must come before DURATION. On Windows, which has no signals,
  the limit still works: TERM and KILL both end COMMAND, so timeoutx is a portable way to bound a
  step in a script on every platform.
- **Verified by.** `core/timeout/timeout_test.go`, `cmd/timeoutx/timeoutx_test.go`.

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

### FR-140: truncatex

`truncatex` must shrink or extend the size of each FILE to the specified size, as GNU `truncate` does.

```text
truncatex OPTION... FILE...
```

| Option              | Meaning |
|---------------------|---------|
| `-o`, `--io-blocks` | Treat SIZE as number of IO blocks instead of bytes |
| `-c`, `--no-create` | Do not create any files |
| `-r`, `--reference` | Base size on RFILE |
| `-s`, `--size`      | Set or adjust the file size by SIZE bytes |

- **Behavior.** truncatex sets the size of each FILE. A FILE that does not exist is created,
  unless -c is given. A file larger than the size loses the data past it; a file shorter than the
  size is extended with zero bytes (on most file systems as a sparse hole that takes no disk
  space). -s SIZE gives the size: a number with an optional suffix (K, M, G, T, P, E for powers of
  1024, also written KiB, MiB; KB, MB for powers of 1000), optionally preceded by + (extend by), -
  (reduce by), < (at most), > (at least), / (round down to a multiple of) or % (round up to a
  multiple of). -r FILE takes the size from another file; with -s as well, the relative SIZE
  adjusts the reference size. -o counts SIZE in I/O blocks rather than bytes. One of -s and -r is
  required. Off Unix the I/O block size used by -o is 512 bytes, because the platform does not
  report one.
- **Verified by.** `core/truncate/truncate_test.go`, `cmd/truncatex/truncatex_test.go`.

### FR-91: trx

`trx` must translate, squeeze and delete bytes from standard input, as GNU `tr` does.

```text
trx [OPTION]... STRING1 [STRING2]
```

| Option                     | Meaning                                                                    |
|----------------------------|----------------------------------------------------------------------------|
| `-c`, `-C`, `--complement` | Use the complement of SET1                                                 |
| `-d`, `--delete`           | Delete the bytes in SET1, do not translate                                 |
| `-s`, `--squeeze-repeats`  | Squeeze each run of a repeated byte from the last SET given into one       |
| `-t`, `--truncate-set1`    | First truncate SET1 to the length of SET2                                  |

- **Behavior.** Reads only standard input and writes standard output. Translating needs two
  SETs; `-d` alone takes one, `-d -s` two, `-s` one or two. SET2 is padded with its last byte
  to the length of SET1 unless `-t`; a byte repeated in SET1 takes its last mapping. SETs
  accept `\NNN`, `\\`, `\a` `\b` `\f` `\n` `\r` `\t` `\v`, ranges, `[:CLASS:]` (the twelve POSIX
  classes), `[=C=]`, `[C*N]` and, in SET2 only, `[C*]`. When translating, SET2 may hold only
  `[:upper:]`/`[:lower:]`, aligned with a case class in SET1. Wrong operand counts are usage
  errors with GNU's second line ("Two strings must be given when translating."); an invalid
  SET (reversed range, unknown class, misaligned case class, empty SET2) is a diagnostic with
  status 1. A trailing backslash and an over-range octal escape draw GNU's warnings.
- **Deviations.** None known in the options listed. Like GNU `tr`, it is byte-oriented: a
  multibyte UTF-8 character is several bytes and the classes are the C locale's (ASCII).
- **Verified by.** `core/tr/tr_test.go`, `cmd/trx/trx_test.go`.

### FR-159: tsortx

`tsortx` must perform a topological sort, as GNU `tsort` does.

```text
tsortx [OPTION] [FILE]
```

tsortx has no options of its own.

- **Behavior.** tsortx reads pairs of strings separated by spaces, tabs and newlines from FILE
  (standard input when there is none, or for -); a pair 'A B' puts A before B, and a pair of one
  string twice names it. It prints every string once, after every string paired before it, in
  GNU's order (its algorithm: the strings nothing precedes in byte order, then each as it is
  freed). A loop is reported as 'input contains a loop:' and its strings on standard error, one
  of its pairs is dropped and sorting carries on, and the exit status is 1. An odd number of
  strings is an error and prints nothing. More than one FILE is a usage error.
- **Verified by.** `core/tsort/tsort_test.go`, `cmd/tsortx/tsortx_test.go`.

### FR-141: ttyx

`ttyx` must print the file name of the terminal connected to standard input, as GNU `tty` does.

```text
ttyx [OPTION]...
```

| Option           | Meaning |
|------------------|---------|
| `--quiet`        | Same as --silent |
| `-s`, `--silent` | Print nothing, only return an exit status |

- **Behavior.** ttyx prints the file name of the terminal on standard input, such as /dev/pts/3,
  or 'not a tty' when standard input is a pipe, a file or nothing, and then exits with status 1.
  -s prints nothing, so only the exit status tells; GNU deprecates it in favor of 'test -t 0',
  which gnux has as testx -t 0. The name is found without the C library: through /proc on Linux,
  and on macOS and the BSDs by finding the device in /dev with the same device number, as
  ttyname(3) does. Windows has no terminal file names, so a console prints as CON (its device
  name) instead of failing. The exit status is 0 when standard input is a terminal, 1 when it is
  not, 2 for a command-line mistake and 3 when the name could not be written.
- **Verified by.** `core/tty/tty_test.go`, `cmd/ttyx/ttyx_test.go`.

### FR-142: unamex

`unamex` must print system information, as GNU `uname` does.

```text
unamex [OPTION]...
```

| Option                      | Meaning |
|-----------------------------|---------|
| `-a`, `--all`               | Print all information, in the following order, except omit -p and -i if unknown |
| `-i`, `--hardware-platform` | Print the hardware platform (non-portable) |
| `-s`, `--kernel-name`       | Print the kernel name |
| `-r`, `--kernel-release`    | Print the kernel release |
| `-v`, `--kernel-version`    | Print the kernel version |
| `-m`, `--machine`           | Print the machine hardware name |
| `-n`, `--nodename`          | Print the network node hostname |
| `-o`, `--operating-system`  | Print the operating system |
| `-p`, `--processor`         | Print the processor type (non-portable) |

- **Behavior.** unamex prints information about the machine and operating system. With no option
  it prints the kernel name, as -s does. The selected fields are printed on one line, separated by
  spaces, in a fixed order whatever the order of the options: kernel name (-s), network node host
  name (-n), kernel release (-r), kernel version (-v), machine hardware name (-m), processor type
  (-p), hardware platform (-i) and operating system (-o). -a prints them all, leaving out -p and
  -i when they are unknown, as GNU does. The values come from uname(2) on Linux, sysctl on macOS
  and FreeBSD, and the registry on Windows, so unamex prints the same fields on every platform: on
  Windows the kernel name is 'Windows_NT' and the operating system 'MS/Windows'; in the browser
  playground the fields describe the WebAssembly runtime.
- **Verified by.** `core/uname/uname_test.go`, `cmd/unamex/unamex_test.go`.

### FR-143: unexpandx

`unexpandx` must convert spaces to tabs, as GNU `unexpand` does.

```text
unexpandx [OPTION]... [FILE]...
```

| Option         | Meaning |
|----------------|---------|
| `-a`, `--all`  | Convert all blanks, instead of just initial blanks |
| `--first-only` | Convert only leading sequences of blanks (overrides -a) |
| `-t`, `--tabs` | Have tabs N characters apart instead of 8 (enables -a), or use comma separated list of explicit tab positions |

- **Behavior.** unexpandx writes each FILE to standard output with runs of blanks replaced by tabs
  where they reach a tab stop. With no FILE, or when FILE is -, it reads standard input. By
  default only the blanks at the start of each line are converted; -a converts every run of two or
  more blanks that ends at a tab stop, and --first-only goes back to the leading blanks only. Tab
  stops are every 8 columns unless -t says otherwise, with the same syntax as expandx: -t N, or -t
  LIST with an optional final /N or +N. -t implies -a, as in GNU. Several -t options add to one
  list, and the obsolete -N form (unexpandx -4) is a list of its own, in the order given, without
  implying -a; GNU's comma form -4,8 is not accepted, and the digits of two -N options in a row
  run together (-4 -8 reads as -48), so write -t 4,8 instead. When both -a and --first-only are
  given, --first-only wins.
- **Verified by.** `core/unexpand/unexpand_test.go`, `cmd/unexpandx/unexpandx_test.go`.

### FR-89: uniqx

`uniqx` must report or omit repeated adjacent lines, as GNU `uniq` does.

```text
uniqx [OPTION]... [INPUT [OUTPUT]]
```

| Option                         | Meaning                                                                      |
|--------------------------------|------------------------------------------------------------------------------|
| `-c`, `--count`                | Prefix lines by the number of occurrences                                    |
| `-d`, `--repeated`             | Only print duplicate lines, one for each group                               |
| `-D`, `--all-repeated[=METHOD]`| Print all duplicate lines; METHOD: none (default), prepend, separate        |
| `-f`, `--skip-fields=N`        | Avoid comparing the first N fields                                           |
| `--group[=METHOD]`             | Show all lines, groups delimited; METHOD: separate (default), prepend, append, both |
| `-i`, `--ignore-case`          | Ignore differences in case when comparing                                    |
| `-s`, `--skip-chars=N`         | Avoid comparing the first N characters                                       |
| `-u`, `--unique`               | Only print unique lines                                                      |
| `-z`, `--zero-terminated`      | Line delimiter is NUL, not newline                                           |
| `-w`, `--check-chars=N`        | Compare no more than N characters in lines                                   |

- **Behavior.** Only adjacent lines are compared. The comparison skips N fields (blanks then
  non-blanks), then N characters, then compares at most N characters. OUTPUT, when given, is
  created or truncated; `-` for INPUT or OUTPUT is the standard stream. Every output line ends
  with the line delimiter. `--group` cannot be combined with `-c`, `-d`, `-D` or `-u`, and
  `-D` cannot be combined with `-c`; both are usage errors in GNU's words. METHOD arguments may
  be abbreviated. With `-D` the comparison anchor moves to each line, as in GNU.
- **Deviations.** Characters for `-s` and `-w` are bytes (GNU's C locale) and `-i` folds ASCII
  letters only. Not implemented: the obsolete `-N` and `+N` forms of `-f` and `-s`.
- **Verified by.** `core/uniq/uniq_test.go`, `cmd/uniqx/uniqx_test.go`.

### FR-144: unlinkx

`unlinkx` must call the unlink function to remove the specified file, as GNU `unlink` does.

```text
unlinkx FILE
```

`unlinkx` takes no options besides `--help` and `--version`.

- **Behavior.** unlinkx removes the single name FILE with the unlink(2) system call. It takes
  exactly one operand and no options: it never recurses, never prompts, never ignores a missing
  file, and refuses a directory, even an empty one, with 'Is a directory' (use rmdirx for that). A
  symbolic link is removed, not its target. Removing the last name of a file frees its data once
  no process has it open.
- **Verified by.** `core/unlink/unlink_test.go`, `cmd/unlinkx/unlinkx_test.go`.

### FR-145: uptimex

`uptimex` must tell how long the system has been running, as GNU `uptime` does.

```text
uptimex [OPTION]... [FILE]
```

`uptimex` takes no options besides `--help` and `--version`.

- **Behavior.** uptimex prints, on one line, the current time, how long the system has been up,
  how many users are logged in, and the load averages: the average number of processes running or
  waiting to run over the last 1, 5 and 15 minutes (on Linux, processes waiting on disk count
  too). A load average above the number of CPUs (nprocx) means work is queuing. With FILE, the
  users and the boot time are read from that utmp file instead of the current records. The boot
  time comes from /proc on Linux, kern.boottime on macOS and FreeBSD, and the tick count on
  Windows, where the users are the signed-in sessions; Windows keeps no load average, so that part
  is left out there, as GNU uptime leaves it out wherever the system cannot report it. If the boot
  time cannot be found the time up is shown as '????' and the exit status is 1.
- **Verified by.** `core/uptime/uptime_test.go`, `cmd/uptimex/uptimex_test.go`.

### FR-163: usersx

`usersx` must print the user names of the users logged in, as GNU `users` does.

```text
usersx [OPTION]... [FILE]
```

usersx has no options of its own.

- **Behavior.** usersx prints the login name of each session, sorted and separated by spaces on
  one line, a user once per session, and nothing when nobody is logged in. It reads whox's
  records (FR-147): the utmp file without the sessions whose process has ended, or utmpx on
  macOS, or the signed-in sessions on Windows. FILE reads another file in utmp format, leaving
  nothing out; it is not supported on Windows. More than one FILE is a usage error.
- **Verified by.** `core/users/users_test.go`, `cmd/usersx/usersx_test.go`.

### FR-164: vdirx

`vdirx` must list directory contents in the long format, as GNU `vdir` does.

```text
vdirx [OPTION]... [FILE]...
```

vdirx takes every option of lsx (FR-60).

- **Behavior.** vdirx is lsx with GNU vdir's defaults: it takes every lsx option (FR-60) and
  lists exactly as lsx does, but uses the long format, and writes unprintable characters in names as backslash
  escapes, as -b does, and never prints `?` for one without `-q`. An option that picks a format
  or a quoting style, or the `QUOTING_STYLE` variable, replaces those defaults. Diagnostics and
  exit statuses are lsx's, prefixed with vdirx.
- **Verified by.** `cmd/vdirx/vdirx_test.go`, `core/ls/ls_test.go`.

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

### FR-146: whoamix

`whoamix` must print the user name associated with the current effective user ID, as GNU `whoami` does.

```text
whoamix [OPTION]...
```

`whoamix` takes no options besides `--help` and `--version`.

- **Behavior.** whoamix prints the name of the user the process runs as: the effective user, so
  inside sudo it prints root. It takes no operands. On Windows, where the account name is
  DOMAIN\user, only the user part is printed, as GNU whoami does under Cygwin and MSYS. If the
  user ID has no name, whoamix reports 'cannot find name for user ID N' and exits 1.
- **Verified by.** `core/whoami/whoami_test.go`, `cmd/whoamix/whoamix_test.go`.

### FR-147: whox

`whox` must show who is logged on, as GNU `who` does.

```text
whox [OPTION]... [ FILE | ARG1 ARG2 ]
```

| Option             | Meaning |
|--------------------|---------|
| `-a`, `--all`      | Same as -b -d --login -p -r -t -T -u |
| `-b`, `--boot`     | Time of last system boot |
| `-q`, `--count`    | All login names and number of users logged on |
| `-d`, `--dead`     | Print dead processes |
| `-H`, `--heading`  | Print line of column headings |
| `-l`, `--login`    | Print system login processes |
| `--lookup`         | Attempt to canonicalize hostnames via DNS |
| `-T`, `--mesg`     | Add user's message status as +, - or ? |
| `--message`        | Same as -T |
| `-m`, `--my-line`  | Only hostname and user associated with stdin |
| `-p`, `--process`  | Print active processes spawned by init |
| `-r`, `--runlevel` | Print current runlevel |
| `-s`, `--short`    | Print only name, line, and time (default) |
| `-t`, `--time`     | Print last system clock change |
| `-u`, `--users`    | List users logged in |
| `-w`, `--writable` | Same as -T |

- **Behavior.** whox lists the users logged in: name, terminal, login time and, for a remote
  session, the host in parentheses. -u adds how long each terminal has been idle ('.' under a
  minute, 'old' over a day) and the session's process ID, -T (or -w) whether the user accepts
  messages (+, - or ?), and -H a line of headings. -q prints just the names and their count. -b
  shows the last boot, -r the run level, -l login processes, -p processes started by init, -d dead
  processes and -t clock changes; -a shows all of them. -m (or any two operands, as in 'whox am
  i') shows only the terminal on standard input. --lookup turns host addresses into names. With
  FILE, the records are read from that utmp file (on Linux, /var/log/wtmp lists past logins too)
  instead of the current ones, and in the current ones, logins whose process has ended are left
  out. Times are in the C locale's 'Oct 12 09:30' format. The records are read without the C
  library, in each system's own format: Linux utmp, macOS utmpx and FreeBSD utx.active. On
  Windows, which keeps sessions rather than a utmp file, whox lists the signed-in console and
  Remote Desktop sessions, so the same command answers 'who is on this machine' everywhere; there
  FILE is not supported.
- **Verified by.** `core/who/who_test.go`, `cmd/whox/whox_test.go`.

### FR-151: xargsx

`xargsx` must build and run command lines from standard input, as GNU `xargs` (findutils)
does.

```text
xargsx [OPTION]... [COMMAND [INITIAL-ARGUMENT]...]
```

| Option                                   | Meaning                                                        |
|------------------------------------------|----------------------------------------------------------------|
| `-0`, `--null`                           | Items end in NUL; quotes and backslashes are literal            |
| `-d CHARACTER`, `--delimiter=CHARACTER`  | Items end in CHARACTER; quotes and backslashes are literal      |
| `-a FILE`, `--arg-file=FILE`             | Read items from FILE, not standard input                        |
| `-E END`, `-e[END]`, `--eof[=END]`       | Stop at an item equal to END                                    |
| `-n MAX-ARGS`, `--max-args=MAX-ARGS`     | At most MAX-ARGS items per command line                         |
| `-L MAX-LINES`, `-l[MAX-LINES]`, `--max-lines[=MAX-LINES]` | At most MAX-LINES input lines per command line |
| `-I R`, `-i[R]`, `--replace[=R]`         | One command per input line, with R replaced by the line         |
| `-s MAX-CHARS`, `--max-chars=MAX-CHARS`  | Limit the length of a command line                              |
| `-x`, `--exit`                           | Exit if the size limit is exceeded                              |
| `-r`, `--no-run-if-empty`                | Do not run COMMAND when there are no items                      |
| `-t`, `--verbose`                        | Print each command line on standard error before running it     |
| `-p`, `--interactive`                    | Ask before running each command line                            |
| `-P MAX-PROCS`, `--max-procs=MAX-PROCS`  | Run up to MAX-PROCS commands at once                            |
| `--show-limits`                          | Show the limits on command-line length                          |

- **Behavior.** COMMAND defaults to `echo`. Without `-n`, `-L` or `-I`, each command line is
  filled up to the size limit. Without `-r` the command runs once even when there is no input.
  The exit status is 0 on success, 123 when any command exited with a status from 1 to 125,
  124 when one exited with status 255 (xargsx stops), 125 when a signal ended one, 126 when the
  command cannot be run, 127 when it is not found, and 1 for other errors.
- **Deviations.** A command named after a gnux tool, by either name, runs in-process;
  anything else is a program found on `PATH`. Conflicting options take a fixed precedence
  instead of last-one-wins, each with GNU's warning for the ignored one: `-I`/`-i` over
  `-L`/`-l` over `-n`, and `-0` over `-d`. `-P 0` runs at most 128 commands at once. The
  default size limit is a fixed figure, not one asked of the system. A NUL in the input
  outside `-0` mode is dropped with GNU's warning. Not implemented, and refused: `-o` /
  `--open-tty`, `--process-slot-var`, and changing `-P` with signals while running.
- **Verified by.** `core/xargs/xargs_test.go`, `core/xargs/xargs_unix_test.go`,
  `cmd/xargsx/xargsx_test.go`.

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

- **Verified by.** `TestDispatch` ("subcommand") in `cmd/main_test.go`;
  `TestSubcommandMatchesArgv0` requires the same output and status as the own-name form.

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

### FR-95: shell completion

`gnux completion SHELL` must print a tab-completion script for bash, zsh, fish and PowerShell.

- **Rationale.** Completion makes the tool and option names discoverable where they are typed,
  and package managers install it alongside the binary.
- **Acceptance.** Each of the four shells gets a non-empty script; `gnux --help` lists the
  `completion` command; the release archives carry the scripts in `completions/` and the
  Homebrew cask installs them.
- **Verified by.** `TestCompletion` in `cmd/completion_test.go`; the archive contents by
  `make release/snapshot`.

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
  a terminal. With an operand it runs that file. With `-c STRING` it runs the string. Further
  operands are the script's arguments (FR-153). An unknown option is a usage error with status
  2. A script that cannot be opened is reported and exits 127. A command that an `if`, a loop,
  a `case` or a `$(` leaves unfinished is continued on the next line, prompted for with `> `.
- **Verified by.** `TestShellCommand` in `cmd/shell_test.go`.

### FR-24: the language

The shell must support pipelines, `&&`, `||`, `;`, `!`, input and output redirection (`<`, `>`,
`>>`), `NAME=value` variables with `$NAME`, `${NAME}`, `${NAME:-default}` and `$?`, single and
double quotes, backslash escapes, globbing (`*`, `?`, `[...]`), `~` expansion and `#` comments,
together with control flow (FR-152), script arguments (FR-153) and command substitution
(FR-155).

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

The shell must provide `cd`, `exit`, `help`, `set`, `shift`, `break`, `continue` and `[` as
builtins.

- **Acceptance.** `cd` with no argument goes to the home directory; `exit` takes an optional
  status, masked to 0 to 255; `help` prints the language summary. `set` accepts `-e`, `-u`,
  `+e`, `+u` and `-- ARG...` and refuses anything else with status 2. `shift [N]` drops the
  first N arguments and fails when there are fewer. `break [N]` and `continue [N]` act on the
  Nth enclosing loop and fail outside one. `[ EXPRESSION ]` runs the `test` tool and requires
  its closing `]`.
- **Verified by.** `TestCd` and `TestSetAndHelp` in `core/shell/shell_test.go`;
  `TestControlFlow` and `TestParameters` in `core/shell/control_test.go`.

### FR-27: exit status

The shell's exit status must be that of the last command; 2 for a syntax error; 127 for an
unknown command.

- **Acceptance.** A syntax error aborts a non-interactive script. An interactive session
  reports it and carries on.
- **Verified by.** `TestSyntaxErrors` and `TestDiagnostics` in `core/shell/shell_test.go`;
  `TestShellCommand` in `cmd/shell_test.go`.

### FR-28: errexit

With `-e`, the shell must stop at the first failing command whose status is not tested.

- **Acceptance.** A status is tested when the command stands before `&&` or `||`, follows `!`,
  or is part of the condition of an `if` or a `while`. `set -e` and `set +e` turn the option on
  and off from inside the script. An assignment whose value is a failed `$(...)` is a failing
  command.
- **Verified by.** `TestErrExit` in `core/shell/shell_test.go`; `TestErrExitAndConditions` in
  `core/shell/control_test.go`.

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
  the environment. An unquoted `$(...)` is split into lines, never into words. Nothing spans
  lines except an `if`, a loop, a `case` or a `$(...)`: not a quoted string, not a line ending
  in `|`, `&&` or `||`, not a trailing backslash. There are no functions, arithmetic, `until`,
  here-documents, backquotes, `( )` or `{ }` grouping, or background jobs. External programs
  are not looked up: they are reached through `envx PROGRAM ARG...`.
- **Verified by.** `TestScripts` and `TestGlob` in `core/shell/shell_test.go`;
  `TestSubstitution` and `TestControlSyntaxErrors` in `core/shell/control_test.go`; review of
  the shell's reference page for the documentation half.

### FR-79: pipelines

The stages of a pipeline must run concurrently, with every redirection opened before any stage
starts, and a stage must not be able to change the shell's own state.

- **Rationale.** `lsx | wcx -l > count` must count `count`, as a POSIX shell guarantees.
  `yesx | headx` must end when `headx` does.
- **Acceptance.** Expansion, assignments and redirections happen in order before the stages
  run. A stage whose reader has gone sees a closed-pipe error and stops quietly. `cd` and
  `exit` inside a stage do not affect the shell. The pipeline's status is that of its last
  stage, inverted when the pipeline starts with `!`. A pipeline holds at most one compound
  command (FR-152), and what that one sets does not outlast the pipeline.
- **Verified by.** `TestScripts` and `TestRedirection` in `core/shell/shell_test.go`;
  `TestControlFlow` in `core/shell/control_test.go`;
  `TestRunRepeatsUntilThePipeCloses` in `core/yes/yes_test.go`.

### FR-152: control flow

The shell must support `if`/`then`/`elif`/`else`/`fi`, `for NAME in WORD...; do ...; done`,
`while ...; do ...; done` and `case WORD in PATTERN | PATTERN) ... ;; esac`.

- **Rationale.** A build script branches on what it finds and repeats a step per file or per
  platform. Without these, the first such script goes back to the system's shell, and to the
  differences between systems that BR-5 exists to remove.
- **Acceptance.** The condition of an `if` or a `while` is any list of commands, and its last
  status decides. A `for` without `in` runs over the script's arguments; its items are expanded
  once, and a variable among them is not re-split. A `case` runs the first arm one of whose
  patterns matches; a pattern uses `*`, `?` and `[...]`, `*` also matches `/`, and a quoted
  pattern or one from a variable matches only itself. The status of a compound command is that
  of the last command it ran, or 0 when it ran none. A compound command may be redirected as a
  whole and may be one stage of a pipeline. Each may be written on one line or over several.
  The reserved words mean something only as the first word of a command.
- **Verified by.** `TestControlFlow`, `TestCompoundRedirection` and `TestControlSyntaxErrors`
  in `core/shell/control_test.go`; `TestShellCommand` ("control flow with the real tools") in
  `cmd/shell_test.go`.

### FR-153: script arguments

A script must be able to read the arguments it was started with.

- **Acceptance.** The operands after `SCRIPT` are `$1`, `$2`, ... (`${10}` and beyond in
  braces), `$#` is their count and `$0` the script's name; after `-c STRING` the first operand
  is `$0`. `"$@"` and `$@` are the arguments as one word each, and nothing at all when there
  are none; `$*` is one word, the arguments joined by spaces. Options are read only before
  `SCRIPT`. A library caller sets them with `Shell.SetArgs`.
- **Verified by.** `TestParameters` in `core/shell/control_test.go`; `TestShellCommand`
  ("script arguments") in `cmd/shell_test.go`.

### FR-154: nounset

With `-u`, expanding a variable that is not set must be an error that stops the script.

- **Rationale.** A misspelled or missing variable in `rmx -rf $OUT/` must not quietly become
  `rmx -rf /`.
- **Acceptance.** The shell writes `gnux: NAME: unbound variable`, does not run the command,
  and stops with status 1; an interactive session loses that command only. This holds wherever
  a variable is read: an argument, an assignment, a redirection target, a `for` list, a `case`.
  `${NAME:-default}`, `$?`, `$#`, `$0`, `$@` and `$*` are never errors. An environment variable
  that is set but empty counts as unset. `set -u` and `set +u` turn the option on and off.
- **Verified by.** `TestNoUnset` in `core/shell/control_test.go`; `TestShellCommand`
  ("-u refuses an unset variable") in `cmd/shell_test.go`.

### FR-155: command substitution

`$(COMMANDS)` must expand to what the commands write to standard output.

- **Acceptance.** Trailing newlines are removed. Inside double quotes, and as the value of an
  assignment, the result is one word. Unquoted, it is one word per non-empty line, and none
  when there is no output; it is never split at spaces and never globbed (FR-78). It nests, and
  may span lines. The commands run as a subshell: variables, options and arguments they change,
  and a `cd` they make, end with the substitution. A command made only of assignments has the
  status of its last substitution.
- **Verified by.** `TestSubstitution` in `core/shell/control_test.go`.

### FR-168: the sandbox

`gnux shell --sandbox DIR` (and `gnux mcp`, FR-169) must confine the process with the operating
system's own sandbox before it reads or runs anything.

| Option                  | Meaning |
|-------------------------|---------|
| `--sandbox DIR`         | Let the shell, its commands and the programs they start read and write DIR; may be repeated |
| `--sandbox-read DIR`    | With `--sandbox`, also let them read DIR; may be repeated |

- **Acceptance.** Once entered, the process (every tool, in-process, and every program a command
  starts) can read and write the `--sandbox` directories, read the `--sandbox-read` ones and the
  system's own directories (programs, libraries, configuration, time zones), write `/dev/null`,
  and reach nothing else, the home directory included; neither a symbolic link nor `..` gets
  around it, and it cannot be lifted. The sandbox is Landlock on Linux 5.13 or later (in a build
  without cgo, which can restrict every thread), Seatbelt (`sandbox-exec`, the process run again
  under a deny-by-default profile) on macOS, and an AppContainer on Windows 8 or later (the
  process run again inside a container granted the directories by an inherited access entry, its
  streams and exit status passed through, ended with its parent by a job), and Capsicum on
  FreeBSD (the directories opened as `os.Root` handles, every engine's file access sent through
  them by `core/fsys`, then capability mode entered, in which nothing can be opened by path and
  no program started, so a command runs gnux's tools only). After entering, the
  process checks that it is confined (creating a file in a directory outside the policy must be
  refused) and refuses to go on otherwise. A directory that does not exist, `--sandbox-read`
  without `--sandbox`, or a system with no usable sandbox (js, wasip1, a Linux kernel without
  Landlock, a cgo build on Linux) runs nothing: the reason is written and the status is 125.
- **Deviations.** On macOS `stat` can see the names and sizes of files outside the policy, never
  their contents. On Windows the container has no network access; on Linux and macOS the network
  is not restricted. On Windows the access entries stay on the directories, so that a later run
  with the same policy starts at once; a container of another policy is granted none of them. On
  FreeBSD the read-only directories are kept read-only by `core/fsys`, not by the kernel.
- **Verified by.** `internal/sandbox/sandbox_test.go`, which confines a copy of the test binary
  and checks what it can still read and write (run without cgo in the Test workflow on Linux,
  macOS and Windows, and in a FreeBSD virtual machine); `core/fsys/rooted_test.go`; and the whole
  suite with `-tags gnux_rooted`, which sends every file access through `os.Root`.

### FR-169: gnux mcp

`gnux mcp [--read DIR]... [--timeout DURATION] [--max-output BYTES] [DIR]...` must serve the
tools to an AI agent over the Model Context Protocol, sandboxed to DIR.

- **Acceptance.** The server speaks JSON-RPC 2.0 over standard input and output, one message
  per line, and answers `initialize` (protocol revisions 2025-06-18, 2025-03-26 and
  2024-11-05), `ping`, `tools/list` and `tools/call`; anything else gets "method not found",
  malformed input "parse error", and notifications no answer. Before answering it enters the
  sandbox (FR-168) with each DIR (the current directory when none is given) writable and each
  `--read` DIR readable; where it cannot, it serves nothing and exits 125. It offers one tool,
  `run`, taking a `command` (a gnux shell command line) and an optional `directory` (relative to
  the first DIR, or absolute, and inside a DIR; another is refused). Each call runs as a process
  of its own, which inherits the sandbox, with no standard input; it is stopped, with every
  program it started, after `--timeout` (2 minutes), and each output stream is cut off after
  `--max-output` bytes (1 MiB). The result carries the text for the model and the same as
  structured content (`stdout`, `stderr`, `status`, `timedOut`, `truncated`), and is an error
  when the status is not 0. Requests are handled concurrently.
- **Verified by.** `internal/mcp/mcp_test.go` (the protocol), `cmd/mcp_test.go` (the output
  limit, the directory check, the result text).

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
  per supported platform, named `gnux-<tag>-<os>-<arch>` (`.tar.gz`, `.zip` on Windows), each
  with the tool's completion scripts, a `checksums.txt` of their SHA-256 sums and a build
  provenance attestation.
- **Acceptance (packages).** The same release carries a `.deb`, an `.rpm` and an `.apk` for
  each Linux architecture, each installing `/usr/bin/gnux` and the completion scripts.
- **Acceptance (package managers).** The same release updates a Homebrew cask
  (`Casks/gnux.rb` in this repository), a Scoop manifest (`bucket/gnux.json`), a winget
  manifest (a pull request to `microsoft/winget-pkgs`), a strictly confined snap (`gnux` on
  the Snap Store) and an AUR package (`gnux-bin`), the last three each when its secret is configured. A
  prerelease reaches none of them but the snap's `edge` channel.
- **Acceptance (GitHub Actions).** `uses: braswelljr/gnux@v0` (`action.yml`) installs a release
  on a Linux, macOS or Windows runner, verifies it against `checksums.txt` and puts gnux and
  every tool name on `PATH`.
- **Verified by.** The Build workflow covers compilation for every platform, the Release
  workflow builds and publishes the archives and manifests (`make release/snapshot` builds them
  locally), and the Action workflow runs `action.yml` on every runner OS. The install
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
from the code. Generated output is not committed: the website build, the release and `make docs`
generate it from the code at hand.

- **Verified by.** The Docs workflow; `make docs`; `.gitignore`.

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

- **Verified by.** The website build (`web/scripts/gen.mjs`), which generates `web/content/docs`
  before every build; the Docs and Web workflows, which generate it in CI.

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
