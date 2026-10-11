## echox

Display a line of text.

Mirrors GNU `echo`. Run it as `echox`, or as `gnux echox`.

### Synopsis

```bash
echox [SHORT-OPTION]... [STRING]...
```

### Description

`echox` writes each STRING to standard output, separated by single spaces and followed by a newline. `-n` omits the trailing newline, `-e` enables interpretation of backslash escapes (\\, \a, \b, \c, \e, \f, \n, \r, \t, \v, \0NNN, \xHH) and `-E` (the default) disables it; \c stops all further output, the newline included. As in GNU echo, options are recognized only as leading arguments made up solely of those letters (`-n`, `-ne`, `-En`, ...): the first argument that is anything else, such as `-x` or --, ends option parsing and is printed literally. `--help` and `--version` are honored only when they are the sole argument.

### Permissions

Needs no filesystem permissions; it only writes to standard output.

### Use cases

- Printing messages and progress from scripts.
- Emitting text without a trailing newline with `-n`, for prompts or piping.
- Producing tabs, newlines or arbitrary bytes with `-e`, identically on every platform.
- Creating or appending to small files with redirection, such as config stubs or notes.
- Feeding a known string into another tool (`wcx`, `headx`, `catx`) to test or demonstrate it.
- Reporting the result of a previous command, such as its exit status.
- Printing text that starts with a dash or contains backslashes verbatim, which is awkward with the shell's own echo.
- Writing the same output in bash, PowerShell and the `gnux shell`, where built-in echo commands differ.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Print a line of text:

```bash
echox Hello, world
```

```powershell
echox Hello, world
```

Print without a trailing newline, as for a prompt:

```bash
echox -n 'Continue? '
```

```powershell
echox -n 'Continue? '
```

Interpret backslash escapes to print a small table:

```bash
echox -e 'name\tsize\nrmx\t2M'
```

```powershell
echox -e 'name\tsize\nrmx\t2M'
```

Combine options: interpret escapes and omit the final newline:

```bash
echox -ne 'no newline here\n'
```

```powershell
echox -ne 'no newline here\n'
```

Print backslashes literally (the default, made explicit with `-E`):

```bash
echox -E 'C:\new\table'
```

```powershell
echox -E 'C:\new\table'
```

Stop all further output, newline included, at \c:

```bash
echox -e 'first\csecond'
```

```powershell
echox -e 'first\csecond'
```

Write bytes by octal and hexadecimal code (prints ABC):

```bash
echox -e '\x41\0102C'
```

```powershell
echox -e '\x41\0102C'
```

Print text that looks like an option; anything other than option letters is literal:

```bash
echox -- -n
```

```powershell
echox -- -n
```

Create a file, then append a line to it:

```bash
echox -e 'line one\nline two' > notes.txt && echox 'line three' >> notes.txt
```

```powershell
echox -e 'line one\nline two' | Out-File notes.txt; echox 'line three' | Add-Content notes.txt
```

Count the characters in a string, newline included:

```bash
echox "home is $HOME" | wcx -c
```

```powershell
echox "home is $HOME" | wcx -c
```

Feed several lines to another tool:

```bash
echox -e 'b\na\nc' | headx -n 2
```

```powershell
echox -e 'b\na\nc' | headx -n 2
```

Report the exit status of the previous command:

```bash
falsex; echox "status: $?"
```

```powershell
falsex; echox "status: $LASTEXITCODE"
```

Show an escape sequence's bytes made visible by `catx -v`:

```bash
echox -e '\e[1mbold\e[0m' | catx -v
```

```powershell
echox -e '\e[1mbold\e[0m' | catx -v
```

Use a variable in the `gnux shell`, identically on every platform:

```bash
gnux shell -c 'NAME=gnux; echox "hello from $NAME" > greet.txt; catx greet.txt'
```

```powershell
gnux shell -c 'NAME=gnux; echox "hello from $NAME" > greet.txt; catx greet.txt'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
echox --help                # options, first examples, exit status
gnux help echox             # this page: the full manual
gnux help echox --OPTION    # one option in depth, with its examples
```

### See also

- [catx](../catx/index.md)
- [yesx](../yesx/index.md)
- [gnux shell](../shell/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
