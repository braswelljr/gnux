## tailx

Print the last part of each file, optionally following it as it grows.

Mirrors GNU `tail`. Run it as `tailx`, or as `gnux tailx`.

### Synopsis

```bash
tailx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: tail invocation](https://www.gnu.org/software/coreutils/manual/html_node/tail-invocation.html)
- [Source: src/tail.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/tail.c)

### Description

`tailx` prints the last 10 lines of each FILE to standard output. With no FILE, or when FILE is -, it reads standard input. `-n` NUM changes the number of lines and `-c` NUM switches to bytes; a leading '+' on NUM starts output at line or byte NUM instead of counting from the end. NUM accepts a multiplier suffix: b is 512, K, M, G, T, P and E are powers of 1024 (also written KiB, MiB, ...), and KB, MB, ... are powers of 1000. With more than one FILE each is preceded by a '==> name <==' header; `-q` suppresses the headers and `-v` forces them. `-f` keeps running and prints data as it is appended, polling every `-s` seconds; `--follow`=name tracks the file name rather than the open file, so it survives log rotation, and `--retry` keeps waiting for a file that is missing. `-F` is shorthand for `--follow`=name `--retry`. `--pid` ends following when the given process exits. On a regular file `tailx` seeks to the end and reads backward, so its cost does not depend on the file's size.

### Permissions

Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified. With `--follow`=name the file is reopened on every poll, so permissions are re-checked as it is rotated.

### Use cases

- Reading the most recent entries of a log.
- Watching a log live with `-f`, or across rotations with `-F`.
- Skipping a header with `-n` +2.
- Following a build log only for as long as the build runs, with `--pid`.
- Waiting for a log file that a service has not created yet, with `--retry`.
- Printing only new log lines from now on with `-n` 0 `-f`.
- Stripping a fixed-size prefix, such as a byte-order mark, with `-c` +NUM.
- Extracting a range of lines when combined with `headx`.
- Reading the last record of a NUL-separated list with `-z`.

### Flags

```text
  -c, --bytes [+]NUM                output the last NUM bytes; or use -c +NUM to output starting with byte NUM of each file
  -f, --follow HOW[="descriptor"]   output appended data as the file grows; HOW is 'name' or 'descriptor' (the default)
  -F, --follow-retry                same as --follow=name --retry
      --help                        display this help and exit
  -n, --lines [+]NUM                output the last NUM lines, instead of the last 10; or use -n +NUM to output starting with line NUM
      --pid PID                     with -f, terminate after process ID, PID, dies
  -q, --quiet                       never output headers giving file names
      --retry                       keep trying to open a file if it is inaccessible
      --silent                      same as --quiet
  -s, --sleep-interval N            with -f, sleep for approximately N seconds (default 1.0) between iterations (default 1)
  -v, --verbose                     always output headers giving file names
      --version                     output version information and exit
  -z, --zero-terminated             line delimiter is NUL, not newline
```

### Examples

Print the last 10 lines of a file:

```bash
tailx app.log
```

```powershell
tailx app.log
```

Print the last 50 lines:

```bash
tailx -n 50 app.log
```

```powershell
tailx -n 50 app.log
```

Print the last 3 lines, using the historical `-NUM` form:

```bash
tailx -3 app.log
```

```powershell
tailx -3 app.log
```

Print everything from the second line on (skip a header):

```bash
tailx -n +2 data.csv
```

```powershell
tailx -n +2 data.csv
```

Print the last 100 bytes of a file:

```bash
tailx -c 100 app.log
```

```powershell
tailx -c 100 app.log
```

Drop a 3-byte UTF-8 byte-order mark by starting at byte 4:

```bash
tailx -c +4 with-bom.txt > clean.txt
```

```powershell
gnux shell -c 'tailx -c +4 with-bom.txt > clean.txt'
```

Print the last line of each log, with and without file name headers:

```bash
tailx -n 1 a.log b.log && tailx -q -n 1 a.log b.log
```

```powershell
tailx -n 1 a.log b.log; tailx -q -n 1 a.log b.log
```

Force the header for a single file (`--silent` is the long spelling of `-q`):

```bash
tailx -v -n 5 app.log
```

```powershell
tailx -v -n 5 app.log
```

Print lines 11 to 20 of a file:

```bash
headx -n 20 app.log | tailx -n 10
```

```powershell
headx -n 20 app.log | tailx -n 10
```

Print the oldest entry of a directory:

```bash
lsx -t | tailx -n 1
```

```powershell
lsx -t | tailx -n 1
```

Print the last entry of a NUL-separated list:

```bash
tailx -z -n 1 names.lst | xargs -0 echox
```

```powershell
tailx -z -n 1 names.lst
```

Follow a log, printing only lines appended from now on:

```bash
tailx -n 0 -f app.log
```

```powershell
tailx -n 0 -f app.log
```

Follow a log across rotations, polling twice a second:

```bash
tailx -F -s 0.5 /var/log/app.log
```

```powershell
tailx -F -s 0.5 C:\logs\app.log
```

Follow by name and wait for a file that does not exist yet:

```bash
tailx --follow=name --retry service.log
```

```powershell
tailx --follow=name --retry service.log
```

Follow a log until a process exits:

```bash
make > build.log 2>&1 & tailx -f --pid=$! build.log
```

```powershell
$p = Start-Process make -RedirectStandardOutput build.log -PassThru; tailx -f --pid=$($p.Id) build.log
```

Report a missing file through the exit status:

```bash
tailx missing.log || echox "exit status $?"
```

```powershell
tailx missing.log; if ($LASTEXITCODE -ne 0) { "exit status $LASTEXITCODE" }
```

Show the end of every log in the portable `gnux shell`:

```bash
gnux shell -c 'catx *.log | tailx -n 5'
```

```powershell
gnux shell -c 'catx *.log | tailx -n 5'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
tailx --help                # options, first examples, exit status
gnux help tailx             # this page: the full manual
gnux help tailx --OPTION    # one option in depth, with its examples
```

### See also

- [headx](../headx/index.md)
- [catx](../catx/index.md)
- [wcx](../wcx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
