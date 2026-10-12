## printenvx

Print all or part of the environment.

Mirrors GNU `printenv`. Run it as `printenvx`, or as `gnux printenvx`.

### Synopsis

```bash
printenvx [OPTION]... [VARIABLE]...
```

### GNU reference

- [GNU coreutils manual: printenv invocation](https://www.gnu.org/software/coreutils/manual/html_node/printenv-invocation.html)
- [Source: src/printenv.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/printenv.c)

### Description

`printenvx` prints the value of each environment VARIABLE, one per line. With no VARIABLE it prints the whole environment as NAME=value lines. A VARIABLE that is not set prints nothing, not even an empty line, and makes the exit status 1, with no message; one that is set to the empty string prints an empty line. Names are matched exactly, including their case, on every platform. -0 ends each item with NUL instead of a newline, so values containing newlines survive.

### Permissions

Needs no permissions. Reads only the process's own environment.

### Use cases

- Checking what value a variable has in the current environment.
- Testing in a script whether a variable is set, from the exit status.
- Dumping the environment into a CI log for debugging.
- Comparing the environment before and after a change.
- Reading a variable on Windows with the same command as on Linux.
- Feeding the environment to another tool safely with -0.

### Flags

```text
      --help      display this help and exit
  -0, --null      end each output line with NUL, not newline
      --version   output version information and exit
```

### Examples

Print the whole environment:

```bash
printenvx
```

```powershell
printenvx
```

Print one variable:

```bash
printenvx HOME
```

```powershell
printenvx USERPROFILE
```

Print several variables, one per line:

```bash
printenvx USER SHELL
```

```powershell
printenvx USERNAME COMSPEC
```

Test whether a variable is set:

```bash
printenvx CI > /dev/null && echox 'running in CI'
```

```powershell
printenvx CI > $null; if ($LASTEXITCODE -eq 0) { 'running in CI' }
```

Show the search path one directory per line:

```bash
printenvx PATH | trx ':' '\n'
```

```powershell
printenvx PATH | trx ';' '\n'
```

Find variables by name:

```bash
printenvx | sortx | headx -n 20
```

```powershell
printenvx | sortx | headx -n 20
```

Print a variable set for one command:

```bash
envx GREETING=hello printenvx GREETING
```

```powershell
envx GREETING=hello printenvx GREETING
```

Print the environment NUL-separated:

```bash
printenvx -0 | trx '\0' '\n' | wcx -l
```

```powershell
printenvx -0 | trx '\0' '\n' | wcx -l
```

Save the environment for a bug report:

```bash
printenvx | sortx > env.txt
```

```powershell
printenvx | sortx > env.txt
```

Read a variable in the portable `gnux shell`:

```bash
gnux shell -c 'printenvx HOME'
```

```powershell
gnux shell -c 'printenvx USERPROFILE'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | every VARIABLE was set (or none was named) |
| 1 | at least one VARIABLE was not set |
| 2 | a command-line mistake |

### Help in the terminal

```bash
printenvx --help                # options, first examples, exit status
gnux help printenvx             # this page: the full manual
gnux help printenvx --OPTION    # one option in depth, with its examples
```

### See also

- [envx](../envx/index.md)
- [echox](../echox/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
