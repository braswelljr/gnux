## exprx

Evaluate an expression and print its value.

Mirrors GNU `expr`. Run it as `exprx`, or as `gnux exprx`.

### Synopsis

```bash
exprx EXPRESSION
```

### GNU reference

- [GNU coreutils manual: expr invocation](https://www.gnu.org/software/coreutils/manual/html_node/expr-invocation.html)
- [Source: src/expr.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/expr.c)

### Description

`exprx` evaluates EXPRESSION, given as separate arguments, and prints its value. From the loosest binding: A | B is A if it is neither empty nor 0, otherwise B (or 0); A & B is A if neither is empty nor 0, otherwise 0; A = B (also ==), A != B, and the orderings compare as integers when both sides are integers and as strings otherwise, giving 1 or 0; + and - add and subtract; * / and % multiply, divide (truncating toward zero) and take the remainder; STRING : REGEX matches. Integers are an optional minus sign and decimal digits, of any size. A match is anchored at the start of STRING: when REGEX has a \( \) group the value is the text the first group matched (or empty), otherwise the number of bytes matched (or 0). REGEX is a POSIX basic regular expression with GNU's \+, \? and \| extensions and character classes such as [[:digit:]]. The keywords match STRING REGEX, substr STRING POS LENGTH, index STRING CHARS and length STRING do the same as their names, and + TOKEN takes TOKEN as a string even if it is a keyword or an operator. Parentheses group. Operators must be separate arguments, and shell characters such as * | & ( ) must be quoted. A leading -- is skipped. The exit status is 0 when the value is neither empty nor 0, 1 when it is, 2 for an invalid expression (syntax error, non-integer argument, division by zero, a bad regular expression) and 3 for other errors. Differences from GNU `expr`: strings are measured and sliced in bytes and compared byte by byte (the C locale), and back references (\1) and the [= =] and [. .] bracket forms are not supported in a REGEX.

### Permissions

Needs no filesystem permissions; it only writes to standard output.

### Use cases

- Doing integer arithmetic in a script, with no size limit.
- Incrementing a counter in the portable `gnux shell`, where there is no $((...)).
- Extracting part of a string with a regular expression group.
- Getting the length of a string or a substring of it.
- Checking that a value is numeric before using it.
- Comparing two values numerically or as strings and branching on the status.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Add two numbers:

```bash
exprx 2 + 3
```

```powershell
exprx 2 + 3
```

Multiply (quote the * from the shell):

```bash
exprx 6 '*' 7
```

```powershell
exprx 6 '*' 7
```

Integer division and remainder:

```bash
exprx 17 / 5 && exprx 17 % 5
```

```powershell
exprx 17 / 5; exprx 17 % 5
```

Increment a counter in a script:

```bash
n=$(exprx "$n" + 1)
```

```powershell
$n = exprx $n + 1
```

Work with numbers of any size:

```bash
exprx 99999999999999999999 '*' 3
```

```powershell
exprx 99999999999999999999 '*' 3
```

Compare two values (prints 1 or 0):

```bash
exprx 10 '>' 9
```

```powershell
exprx 10 '>' 9
```

Get the length of a string:

```bash
exprx length 'hello world'
```

```powershell
exprx length 'hello world'
```

Take a substring (position 1, three bytes):

```bash
exprx substr release-2024 9 4
```

```powershell
exprx substr release-2024 9 4
```

Strip an extension with a regular expression group:

```bash
exprx report.txt : '\(.*\)\.txt'
```

```powershell
exprx report.txt : '\(.*\)\.txt'
```

Check that a value is all digits:

```bash
exprx "$VALUE" : '[0-9]*$' > /dev/null && echox numeric
```

```powershell
exprx "$env:VALUE" : '[0-9]*$' | Out-Null; if ($LASTEXITCODE -eq 0) { 'numeric' }
```

Find the first vowel:

```bash
exprx index gnux aeiou
```

```powershell
exprx index gnux aeiou
```

Use a keyword as a plain string:

```bash
exprx + length
```

```powershell
exprx + length
```

See the status of an invalid expression (2):

```bash
exprx 1 / 0; echox "exit status $?"
```

```powershell
exprx 1 / 0; "exit status $LASTEXITCODE"
```

Count in the portable `gnux shell`:

```bash
gnux shell -c 'N=4; exprx $N + 1'
```

```powershell
gnux shell -c 'N=4; exprx $N + 1'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | the value is neither empty nor 0 |
| 1 | the value is empty or 0 |
| 2 | EXPRESSION is invalid, or missing |
| 3 | another error occurred, such as a failed write |

### Help in the terminal

```bash
exprx --help                # options, first examples, exit status
gnux help exprx             # this page: the full manual
gnux help exprx --OPTION    # one option in depth, with its examples
```

### See also

- [testx](../testx/index.md)
- [seqx](../seqx/index.md)
- [printfx](../printfx/index.md)
- [gnux shell](../shell/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
