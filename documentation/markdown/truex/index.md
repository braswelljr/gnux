## truex

Do nothing, successfully.

Mirrors GNU `true`. Run it as `truex`, or as `gnux truex`.

### Synopsis

```bash
truex [ignored command line arguments]
```

### GNU reference

- [GNU coreutils manual: true invocation](https://www.gnu.org/software/coreutils/manual/html_node/true-invocation.html)
- [Source: src/true.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/true.c)

### Description

`truex` exits with status 0 and does nothing else: it reads no input, writes no output and ignores every argument it is given. The only exceptions are `--help` and `--version`, which are honored when they are the sole argument. It exists so scripts have a portable command whose result is always success.

### Permissions

Needs no permissions; it touches nothing.

### Use cases

- Writing an infinite loop: while `truex`; do ...; done.
- Forcing a command list to succeed, as in `might-fail || truex`.
- Standing in for a command that should be a no-op, such as a disabled hook.
- Keeping a '`gnux shell -e`' script going past a step whose failure is acceptable.
- Testing how a script or CI step reacts to a command that always succeeds.
- Giving the always-true branch of a conditional a portable command to run.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Exit successfully:

```bash
truex
```

```powershell
truex
```

Show the exit status:

```bash
truex; echox $?
```

```powershell
truex; echox $LASTEXITCODE
```

Ignore every argument and still succeed:

```bash
truex --any thing at all; echox $?
```

```powershell
truex --any thing at all; echox $LASTEXITCODE
```

Ignore the failure of a command:

```bash
rmx missing.txt || truex
```

```powershell
rmx missing.txt; if (-not $?) { truex }
```

Chain a command that always runs:

```bash
truex && echox 'always runs'
```

```powershell
truex; if ($LASTEXITCODE -eq 0) { echox 'always runs' }
```

Keep a `gnux shell` script with `-e` running past an allowed failure:

```bash
gnux shell -e -c 'rmx old.log || truex; echox survived'
```

```powershell
gnux shell -e -c 'rmx old.log || truex; echox survived'
```

Loop until interrupted:

```bash
while truex; do date; sleepx 1; done
```

```powershell
while ($true) { truex; if ($LASTEXITCODE -ne 0) { break }; Get-Date; sleepx 1 }
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | always, whatever the arguments |

### Help in the terminal

```bash
truex --help                # options, first examples, exit status
gnux help truex             # this page: the full manual
gnux help truex --OPTION    # one option in depth, with its examples
```

### See also

- [falsex](../falsex/index.md)
- [yesx](../yesx/index.md)
- [gnux shell](../shell/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
