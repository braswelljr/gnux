## printfx

Format and print data.

Mirrors GNU `printf`. Run it as `printfx`, or as `gnux printfx`.

### Synopsis

```bash
printfx FORMAT [ARGUMENT]...
```

### GNU reference

- [GNU coreutils manual: printf invocation](https://www.gnu.org/software/coreutils/manual/html_node/printf-invocation.html)
- [Source: src/printf.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/printf.c)

### Description

`printfx` writes the ARGUMENTs to standard output under the control of FORMAT, like C's `printf`. FORMAT is printed as written except for backslash escapes (\", \\, \a, \b, \c, \e, \f, \n, \r, \t, \v, \NNN octal, \xHH, \uHHHH, \UHHHHHHHH) and conversion specifications: %d %i %o %u %x %X for integers, %f %F %e %E %g %G %a %A for floating point, %c for a character, %s for a string, %b for a string with its backslash escapes interpreted (octal as \0NNN), %q for a string quoted for reuse as shell input, and %% for a percent sign. A conversion may carry the flags - + space # 0, a width and a precision, either of which may be * to take it from the next ARGUMENT. FORMAT is reused as often as needed to consume every ARGUMENT; a missing ARGUMENT reads as an empty string or zero. A numeric ARGUMENT may be decimal, 0x hexadecimal, 0 octal (integers), or a quote followed by a character, meaning that character's code. An ARGUMENT that is not a valid number is diagnosed, printed as far as it converted, and makes the exit status 1 without stopping the output. \c ends all output. Unlike the shell's builtin `printf`, `printfx` behaves the same in bash, PowerShell and the `gnux shell`. Differences from GNU `printf`: floating-point values are held in double precision rather than long double, and \c ends output with the status the run already had rather than always 0.

### Permissions

Needs no filesystem permissions; it only writes to standard output.

### Use cases

- Printing text without a trailing newline, or with exact control over every byte.
- Aligning columns in a report with field widths.
- Formatting numbers: fixed decimals, zero padding, hexadecimal or octal.
- Converting between number bases, or printing a character's code.
- Repeating a line template over a list of arguments.
- Quoting a value with %q so it can be pasted back into a shell safely.
- Writing files with exact content (escapes, no trailing newline) from scripts.
- Getting identical formatted output in bash, PowerShell and the `gnux shell`.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Print a line, adding the newline explicitly:

```bash
printfx 'Hello, %s\n' world
```

```powershell
printfx 'Hello, %s\n' world
```

Format a number with two decimal places:

```bash
printfx '%.2f\n' 3.14159
```

```powershell
printfx '%.2f\n' 3.14159
```

Pad a number with zeros to five digits:

```bash
printfx '%05d\n' 42
```

```powershell
printfx '%05d\n' 42
```

Align columns with field widths:

```bash
printfx '%-10s|%5d\n' apples 3 pears 12
```

```powershell
printfx '%-10s|%5d\n' apples 3 pears 12
```

Reuse the format for every argument:

```bash
printfx '- %s\n' one two three
```

```powershell
printfx '- %s\n' one two three
```

Convert a number to hexadecimal and octal:

```bash
printfx '%x %o\n' 255 255
```

```powershell
printfx '%x %o\n' 255 255
```

Convert hexadecimal to decimal:

```bash
printfx '%d\n' 0xff
```

```powershell
printfx '%d\n' 0xff
```

Print the character code of a letter:

```bash
printfx '%d\n' "'A"
```

```powershell
printfx '%d\n' "'A"
```

Take the field width from an argument:

```bash
printfx '[%*s]\n' 8 right
```

```powershell
printfx '[%*s]\n' 8 right
```

Interpret escapes in an argument with %b:

```bash
printfx '%b\n' 'tab:\there'
```

```powershell
printfx '%b\n' 'tab:\there'
```

Quote a value for reuse as shell input:

```bash
printfx '%q\n' "it's a file.txt"
```

```powershell
printfx '%q\n' "it's a file.txt"
```

Print a Unicode character by code point:

```bash
printfx 'é\n'
```

```powershell
printfx 'é\n'
```

Write a file with no trailing newline:

```bash
printfx 'v1.2.3' > VERSION
```

```powershell
printfx 'v1.2.3' | Out-File -NoNewline VERSION
```

Notice a value that is not a number (exit status 1):

```bash
printfx '%d\n' abc || echox "exit status $?"
```

```powershell
printfx '%d\n' abc; if ($LASTEXITCODE -ne 0) { "exit status $LASTEXITCODE" }
```

Format a table in the portable `gnux shell`:

```bash
gnux shell -c 'printfx "%-6s %3s\n" name qty pen 4 ink 12'
```

```powershell
gnux shell -c 'printfx "%-6s %3s\n" name qty pen 4 ink 12'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
printfx --help                # options, first examples, exit status
gnux help printfx             # this page: the full manual
gnux help printfx --OPTION    # one option in depth, with its examples
```

### See also

- [echox](../echox/index.md)
- [seqx](../seqx/index.md)
- [gnux shell](../shell/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
