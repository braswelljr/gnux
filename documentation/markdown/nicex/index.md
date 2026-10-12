## nicex

Run a program with modified scheduling priority.

Mirrors GNU `nice`. Run it as `nicex`, or as `gnux nicex`.

### Synopsis

```bash
nicex [OPTION] [COMMAND [ARG]...]
```

### GNU reference

- [GNU coreutils manual: nice invocation](https://www.gnu.org/software/coreutils/manual/html_node/nice-invocation.html)
- [Source: src/nice.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/nice.c)

### Description

`nicex` runs COMMAND with its niceness adjusted by 10, or by `-n` ADJUSTMENT, so that it gets less (or, with a negative adjustment, more) CPU time than other processes. Niceness ranges from -20 (most favorable) to 19 (least); the result is clamped to that range. With no COMMAND it prints the current niceness. The obsolete forms `-ADJUSTMENT` and `--ADJUSTMENT` (negative) are accepted as in GNU: '`nicex` -5 make' and '`nicex` --5 make'. Options must come before COMMAND. Lowering the niceness needs privilege; without it `nicex` warns 'cannot set niceness' and runs the command anyway, as GNU `nice` does. The exit status is the command's, or 125 when `nicex` itself failed, 126 when COMMAND was found but could not be run and 127 when it was not found. `nicex` starts COMMAND as a child and sets its niceness as it starts, where GNU `nice` changes its own and then becomes COMMAND. On Windows, which has no niceness, the adjustment picks the nearest priority class instead (idle, below normal, normal, above normal or high; never realtime), so '`nicex` make' really does lower the build's priority there too, and `nicex` with no COMMAND prints the niceness its own class corresponds to.

### Permissions

Raising the niceness (lowering the priority) needs no privilege. Lowering it below the current value needs root (or CAP_SYS_NICE); on Windows the high priority class may need an administrator. Execute permission is needed on COMMAND.

### Use cases

- Running a long build or compression job without slowing an interactive desktop.
- Running backups and batch jobs at low priority.
- Giving a latency-sensitive job more CPU, as root, with a negative adjustment.
- Checking the current niceness of a shell.
- Lowering a job's priority the same way on Linux, macOS and Windows.

### Flags

```text
  -n, --adjustment N   add integer N to the niceness (default 10)
      --help           display this help and exit
      --version        output version information and exit
```

### Examples

Run a build at lower priority:

```bash
nicex make -j8
```

```powershell
nicex make -j8
```

Run at the lowest priority:

```bash
nicex -n 19 tar -czf backup.tgz data/
```

```powershell
nicex -n 19 tar -czf backup.tgz data
```

Use the obsolete adjustment form:

```bash
nicex -5 make
```

```powershell
nicex -5 make
```

Raise a job's priority (needs privilege):

```bash
sudo nicex -n -5 ./server
```

```powershell
nicex -n -5 .\server.exe
```

Raise it with the obsolete form:

```bash
sudo nicex --5 ./server
```

```powershell
nicex --5 .\server.exe
```

Print the current niceness:

```bash
nicex
```

```powershell
nicex
```

Print the niceness a child would get:

```bash
nicex nicex
```

```powershell
nicex nicex
```

Combine with a time limit:

```bash
nicex timeoutx 1h ./batch.sh
```

```powershell
nicex timeoutx 1h .\batch.ps1
```

Pass the adjustment as a long option:

```bash
nicex --adjustment=15 gzip -9 big.log
```

```powershell
nicex --adjustment=15 gzip -9 big.log
```

Keep the command's exit status:

```bash
nicex ./tests.sh; echox "exit: $?"
```

```powershell
nicex .\tests.ps1; "exit: $LASTEXITCODE"
```

Run a low-priority job from the portable `gnux shell`:

```bash
gnux shell -c 'nicex -n 10 envx make'
```

```powershell
gnux shell -c 'nicex -n 10 envx make'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error) |
| 125 | a command-line mistake |

### Help in the terminal

```bash
nicex --help                # options, first examples, exit status
gnux help nicex             # this page: the full manual
gnux help nicex --OPTION    # one option in depth, with its examples
```

### See also

- [timeoutx](../timeoutx/index.md)
- [nohupx](../nohupx/index.md)
- [killx](../killx/index.md)
- [envx](../envx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
