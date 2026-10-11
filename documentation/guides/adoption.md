# Adoption

How to adopt gnux next to the GNU coreutils you already know. gnux does not replace them: it brings the same interface to the places they are missing (Windows, containers without coreutils, the browser, Go programs), so the GNU tools stay where they are and gnux is added where it helps. This guide sets the two side by side: how each of the 22 gnux tools maps onto its GNU counterpart, which flags it supports, every documented deviation (the places where gnux must choose differently because it cannot see option order, or because Windows cannot express a feature), the GNU options that are not implemented yet, and what the rest of coreutils looks like on the roadmap. If a behavior is not listed here as a difference, gnux aims to match GNU exactly, down to the wording of its diagnostics and its exit statuses.

## The short version

- Add an `x`: `rm` is `rmx`, `cp` is `cpx`, `ls` is `lsx`. The GNU names also work as aliases (`gnux rm`, or a link named `rm`), and inside `gnux shell` both names are the same command.
- The flags, their meanings, the diagnostics and the exit statuses are GNU's.
- An option gnux does not implement is **refused** with `unknown flag`, never silently ignored, so a script that relies on one fails loudly instead of doing the wrong thing.
- Where GNU lets the last of two conflicting options win, gnux picks the **most cautious** one for `rmx`, `cpx`, `mvx`, `lnx` and `pwdx`. `lsx`, `headx` and `tailx` track option order and keep GNU's last-one-wins.

## Tool by tool

| GNU | gnux | Supported options | Reference |
|-----|------|-------------------|-----------|
| `basename` | `basenamex` | `-a`, `-s SUFFIX`, `-z` | [basenamex](../markdown/basenamex/index.md) |
| `cat` | `catx` | `-A`, `-b`, `-e`, `-E`, `-n`, `-s`, `-t`, `-T`, `-u`, `-v` | [catx](../markdown/catx/index.md) |
| `cp` | `cpx` | `-a`, `-d`, `-f`, `-H`, `-i`, `-l`, `-L`, `-n`, `-p`, `-P`, `-r`/`-R`, `-s`, `-t`, `-T`, `-u[=WHEN]`, `-v`, `-x`, `--parents`, `--preserve[=LIST]`, `--no-preserve=LIST`, `--strip-trailing-slashes` | [cpx](../markdown/cpx/index.md) |
| `dirname` | `dirnamex` | `-z` | [dirnamex](../markdown/dirnamex/index.md) |
| `echo` | `echox` | `-n`, `-e`, `-E` | [echox](../markdown/echox/index.md) |
| `env` | `envx` | `-0`, `-a ARG`, `-C DIR`, `-i`/`-`, `-S STRING`, `-u NAME`, `-v` | [envx](../markdown/envx/index.md) |
| `false` | `falsex` | none (arguments ignored) | [falsex](../markdown/falsex/index.md) |
| `head` | `headx` | `-c [-]NUM`, `-n [-]NUM`, `-NUM`, `-q`, `-v`, `-z`, size suffixes | [headx](../markdown/headx/index.md) |
| `ln` | `lnx` | `-f`, `-i`, `-L`, `-n`, `-P`, `-r`, `-s`, `-t`, `-T`, `-v` | [lnx](../markdown/lnx/index.md) |
| `ls` | `lsx` | `-1`, `-a`, `-A`, `-B`, `-b`, `-c`, `-C`, `-d`, `-F`, `-g`, `-G`, `-h`, `-H`, `-i`, `-I`, `-l`, `-L`, `-m`, `-n`, `-o`, `-p`, `-q`, `-Q`, `-r`, `-R`, `-s`, `-S`, `-t`, `-T`, `-u`, `-U`, `-v`, `-w`, `-x`, `-X`, `--color`, `--file-type`, `--full-time`, `--group-directories-first`, `--hide`, `--si`, `--sort`, `--time`, `--time-style` | [lsx](../markdown/lsx/index.md) |
| `mkdir` | `mkdirx` | `-m MODE`, `-p`, `-v` | [mkdirx](../markdown/mkdirx/index.md) |
| `mv` | `mvx` | `-f`, `-i`, `-n`, `-t`, `-T`, `-u[=WHEN]`, `-v`, `--strip-trailing-slashes` | [mvx](../markdown/mvx/index.md) |
| `pwd` | `pwdx` | `-L`, `-P` | [pwdx](../markdown/pwdx/index.md) |
| `realpath` | `realpathx` | `-e`, `-L`, `-m`, `-P`, `-q`, `-s`/`--no-symlinks`, `-z`, `--relative-to`, `--relative-base` | [realpathx](../markdown/realpathx/index.md) |
| `rm` | `rmx` | `-d`, `-f`, `-i`, `-I`, `-r`/`-R`, `-v`, `--interactive[=WHEN]`, `--one-file-system`, `--preserve-root`, `--no-preserve-root` | [rmx](../markdown/rmx/index.md) |
| `rmdir` | `rmdirx` | `-p`, `-v`, `--ignore-fail-on-non-empty` | [rmdirx](../markdown/rmdirx/index.md) |
| `sleep` | `sleepx` | NUMBER with `s`, `m`, `h`, `d`; fractions, exponents, hex floats, `infinity` | [sleepx](../markdown/sleepx/index.md) |
| `tail` | `tailx` | `-c [+]NUM`, `-f[=HOW]`, `-F`, `-n [+]NUM`, `-q`, `-s N`, `-v`, `-z`, `--pid`, `--retry`, size suffixes | [tailx](../markdown/tailx/index.md) |
| `touch` | `touchx` | `-a`, `-c`, `-d STRING`, `-h`, `-m`, `-r FILE`, `-t STAMP`, `--time=WORD` | [touchx](../markdown/touchx/index.md) |
| `true` | `truex` | none (arguments ignored) | [truex](../markdown/truex/index.md) |
| `wc` | `wcx` | `-c`, `-l`, `-L`, `-m`, `-w`, `--files0-from=F`, `--total=WHEN` | [wcx](../markdown/wcx/index.md) |
| `yes` | `yesx` | STRING... (no options) | [yesx](../markdown/yesx/index.md) |

Every tool also takes `--help` and `--version`. For `echox`, `truex`, `falsex` and `yesx`, as in GNU, they are honored only as the sole argument.

## Documented deviations

These are the places where gnux deliberately differs from GNU. Each is stated in the tool's reference page and in section 2 of the [functional requirements](../../docs/requirements/functional-requirements.md).

### Conflicting options resolve toward the most cautious

GNU lets the last of conflicting options win. The command-line framework gnux uses does not report option order, so for these tools a fixed precedence applies, always toward the option that destroys least:

| Tool | Precedence (strongest first) | Example |
|------|------------------------------|---------|
| `rmx` | `-i` > `--interactive=WHEN` > `-I` > `-f` | `rmx -if x` prompts, where `rm -if x` would not. `-f` still silences "No such file" errors |
| `mvx` | `-n` > `-i` > `-f` | `mvx -fn a b` never overwrites |
| `cpx` | `-n` > `-i`; `-P`/`-d` > `-H` > `-L` | `cpx -in a b` never overwrites and never asks |
| `lnx` | `-i` > `-f`; `-L` > `-P` | `lnx -fi a b` asks before replacing |
| `pwdx` | `-L` over `-P` | `pwdx -PL` and `pwdx -LP` both print `$PWD` when it is valid |
| `realpathx` | `-e` over `-m` | |

`lsx`, `headx` and `tailx` do track order: `lsx -lC` is columns, `lsx -Cl` is long, exactly as in GNU.

The rule is in the project's [rules](../../docs/rules/README.md) as "conflicts resolve toward the most cautious option".

### Behavior differences, tool by tool

| Tool | Difference |
|------|------------|
| `catx` | `-e`, `-t` and `-u` also answer to the long names `--show-nonprinting-ends`, `--show-nonprinting-tabs` and `--unbuffered`, which GNU does not have |
| `cpx` | Option precedence as above. `-d`, `-H` and `-p` also answer to `--no-dereference-links`, `--dereference-args` and `--preserve-default`. On Windows only the read-only attribute and the modification time are carried over |
| `envx` | GNU `env` replaces itself with the command; `envx` runs it as a child process on the same streams and exits with its status (128 plus the signal number when a signal killed it), so it behaves the same on platforms that cannot replace a running process |
| `lnx` | Option precedence as above. On Windows, symbolic links need Developer Mode or an elevated prompt |
| `lsx` | Names sort by plain byte comparison (the C locale) whatever `LANG` says, so uppercase sorts before lowercase. On Windows the owner, group, inode and block columns hold placeholders. Several single-letter options also have gnux long names: `--long`, `--one-per-line`, `--sort-size`, `--sort-time`, `--sort-version`, `--sort-extension`, `--unsorted`, `--columns`, `--across`, `--commas`, `--slash`, `--ctime`, `--atime`, `--long-no-owner`, `--long-no-group` |
| `mkdirx` | On Windows only the read-only attribute is representable, so most of a `-m` mode has no effect |
| `mvx` | Option precedence as above |
| `pwdx` | `-L` wins when both are given |
| `realpathx` | `-e` wins over `-m` |
| `rmx` | Option precedence as above. `-i` and `-I` also answer to `--prompt-each` and `--prompt-once`. Off Unix, `--one-file-system` has no effect (no device numbers). Independent arguments are removed concurrently when nothing is prompted or printed |
| `tailx` | Following is done by polling at the `-s` interval (no inotify). `-F` also answers to `--follow-retry` |
| `touchx` | `-d` accepts `YYYY-MM-DD`, `YYYY-MM-DD hh:mm[:ss]`, RFC 3339, `@SECONDS` and `now`, not GNU's free-form date phrases such as `yesterday` or `next tuesday`. `-h` works only on Linux; elsewhere the link is reported rather than its target touched. `-a`, `-m` and `-t` also answer to `--access-time`, `--modification-time` and `--stamp` |
| `basenamex`, `dirnamex`, `echox`, `falsex`, `headx`, `rmdirx`, `sleepx`, `truex`, `wcx`, `yesx` | None known |

## Not implemented yet

These GNU options are refused with `unknown flag`:

| Tool | Not implemented |
|------|-----------------|
| `cpx` | `--backup`, `-b`, `--suffix`, `-S`, `--reflink`, `--sparse`, `--attributes-only`, `--remove-destination`, `--copy-contents`, `--keep-directory-symlink`, `-Z`, `--context`, `--debug` |
| `envx` | `--block-signal`, `--default-signal`, `--ignore-signal`, `--list-signal-handling` |
| `lnx` | `--backup`, `-b`, `--suffix`, `-S`, `-d`, `-F`, `--directory` |
| `lsx` | `--author`, `--block-size`, `-D`, `--dired`, `-f`, `--format`, `--hyperlink`, `--indicator-style`, `-k`, `--kibibytes`, `-N`, `--literal`, `--quoting-style`, `--show-control-chars`, `-Z`, `--context`, `--zero` |
| `mkdirx` | `-Z`, `--context` |
| `mvx` | `--backup`, `-b`, `--suffix`, `-S`, `-Z`, `--context`, `--no-copy`, `--exchange`, `--debug` |
| `realpathx` | `-E`, `--canonicalize` (the default behavior anyway) |
| `rmx` | `--preserve-root=all` |
| `tailx` | `--max-unchanged-stats` |
| `touchx` | `-f` (GNU accepts and ignores it) |
| `wcx` | `--debug` |

If a script uses one, the fix is usually simple: drop `--backup` and copy first with `cpx -n`; replace `ls -f` with `lsx -aU`; replace `--format=long` with `-l`.

## Things that look different but are not

- **Diagnostics name the x tool.** `rmx: cannot remove 'x': No such file or directory` rather than `rm: ...`, even when invoked as `rm`. Everything after the prefix is GNU's wording.
- **`--version`** prints `rmx (gnux) VERSION`, in GNU's `NAME (PACKAGE) VERSION` form.
- **`--help`** is gnux's own text: a description, usage, aliases, examples and the flag list.
- **`-h` is free** on every tool, because `--help` has no short form: `lsx -h` and `touchx -h` mean what they mean in GNU.
- **Write-protected prompts** appear only when standard input is a terminal, as in GNU: piping into `rmx` or running it in CI never prompts unless `-i` or `-I` is given.

## Adopting gnux in a shell script

1. Call the x names (`rmx` for `rm`) where the script should run the same everywhere, or leave the names as they are and run the script with `gnux shell`, where both names work. The GNU tools on the machine are left untouched.
2. Check every option against the "not implemented" table.
3. If the script uses only pipelines, `&&`, `||`, `;`, redirection, variables and globs, it can run under `gnux shell` unchanged and becomes portable to Windows. Loops, functions, `if`, `case`, command substitution and here-documents are not part of the [gnux shell language](portable-scripting.md), so those scripts stay with bash, calling the x tools.
4. Calls to other programs (`git`, `go`, `make`) go through `envx` inside `gnux shell`: `envx git status`.

## What is not in gnux yet

gnux covers 22 tools today. The rest of coreutils is on the [roadmap in the README](../../README.md#roadmap), roughly in this order:

- **File attributes:** `chmod`, `chown`, `chgrp`, `stat`, `du`, `df`, `mktemp`, `install`, `truncate`, `readlink`, `link`, `unlink`
- **Text:** `sort`, `uniq`, `cut`, `tr`, `tee`, `paste`, `join`, `nl`, `fold`, `fmt`, `expand`, `unexpand`, `tac`, `split`, `csplit`, `comm`, `shuf`, `seq`, `printf`, `od`
- **Checksums and encoding:** `md5sum`, `sha1sum`, `sha256sum`, `sha512sum`, `b2sum`, `cksum`, `base32`, `base64`
- **System:** `date`, `uname`, `whoami`, `id`, `hostname`, `nproc`, `printenv`, `tty`, `who`, `uptime`, `timeout`, `nice`, `nohup`, `kill`, `test`, `expr`, `dd`

Until then, `catx -n` covers many uses of `nl`, `headx` and `tailx` cover simple slicing, `envx` with no command covers `printenv`, and `realpathx` covers most uses of `readlink -f`.

## See also

- [Functional requirements](../../docs/requirements/functional-requirements.md), section 2: each tool's required behavior, deviations and missing options.
- [Rules](../../docs/rules/README.md): the conventions behind the deviations.
- [Troubleshooting](troubleshooting.md): what an `unknown flag` or other error means.
