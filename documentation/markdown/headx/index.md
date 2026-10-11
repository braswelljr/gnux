## headx

Print the first part of each file.

Mirrors GNU `head`. Run it as `headx`, or as `gnux headx`.

### Synopsis

```bash
headx [OPTION]... [FILE]...
```

### Description

`headx` prints the first 10 lines of each FILE to standard output. With no FILE, or when FILE is -, it reads standard input. `-n` NUM changes the number of lines and `-c` NUM switches to bytes; a leading '-' on NUM inverts the selection and prints everything except the last NUM lines or bytes. NUM accepts a multiplier suffix: b is 512, K, M, G, T, P and E are powers of 1024 (also written KiB, MiB, ...), and KB, MB, ... are powers of 1000. With more than one FILE each is preceded by a '==> name <==' header; `-q` suppresses the headers and `-v` forces them. `-z` makes NUL the line delimiter. The historical '`headx -NUM`' spelling of '`-n` NUM' is accepted. Reading stops as soon as enough has been seen, so `headx` is cheap on huge files.

### Permissions

Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

### Use cases

- Peeking at the start of a large file or log without reading all of it.
- Taking the first few results of a pipeline.
- Dropping a fixed-size trailer with `-n -NUM` or `-c -NUM`.
- Sampling the first bytes of a binary file with `-c`, for example to check a magic number.
- Reading the header row of a CSV file before processing it.
- Comparing the beginnings of several files side by side, with or without '==> name <==' headers.
- Cutting an endless stream such as `yesx` down to a fixed number of lines.
- Extracting a range of lines when combined with `tailx`.
- Handling NUL-separated file lists safely with `-z`.

### Flags

```text
  -c, --bytes [-]NUM      print the first NUM bytes of each file; with a leading '-', print all but the last NUM bytes of each file
      --help              display this help and exit
  -n, --lines [-]NUM      print the first NUM lines instead of the first 10; with a leading '-', print all but the last NUM lines of each file
  -q, --quiet             never print headers giving file names
      --silent            same as --quiet
  -v, --verbose           always print headers giving file names
      --version           output version information and exit
  -z, --zero-terminated   line delimiter is NUL, not newline
```

### Examples

Print the first 10 lines of a file:

```bash
headx access.log
```

```powershell
headx access.log
```

Print the first 3 lines of each file, with headers:

```bash
headx -n 3 a.txt b.txt
```

```powershell
headx -n 3 a.txt b.txt
```

Print only the header row of a CSV file:

```bash
headx -n 1 data.csv
```

```powershell
headx -n 1 data.csv
```

Print the first line of every CSV file without the file name headers:

```bash
headx -q -n 1 *.csv
```

```powershell
headx -q -n 1 (Get-ChildItem *.csv -Name)
```

Show the file name header even for a single file (`--silent` is the long spelling of `-q`):

```bash
headx -v -n 2 config.ini
```

```powershell
headx -v -n 2 config.ini
```

Print the first kibibyte of a file:

```bash
headx -c 1K image.png > start.bin
```

```powershell
gnux shell -c 'headx -c 1K image.png > start.bin'
```

Check a file's magic number from its first bytes:

```bash
headx -c 4 image.png | catx -v
```

```powershell
headx -c 4 image.png | catx -v
```

Copy the first megabyte (10^6 bytes) of a large file:

```bash
headx -c 1MB disk.img > sample.img
```

```powershell
gnux shell -c 'headx -c 1MB disk.img > sample.img'
```

Print everything except the last 2 lines:

```bash
headx -n -2 report.txt
```

```powershell
headx -n -2 report.txt
```

Strip the final byte, such as a trailing newline:

```bash
headx -c -1 version.txt
```

```powershell
headx -c -1 version.txt
```

Keep the first 5 lines of a pipeline, using the historical `-NUM` form:

```bash
lsx -t | headx -5
```

```powershell
lsx -t | headx -5
```

Print lines 11 to 20 of a file:

```bash
headx -n 20 app.log | tailx -n 10
```

```powershell
headx -n 20 app.log | tailx -n 10
```

Cut an endless stream down to three lines:

```bash
yesx ok | headx -n 3
```

```powershell
yesx ok | headx -n 3
```

Take the first two entries of a NUL-separated list:

```bash
find . -name '*.txt' -print0 | headx -z -n 2 | xargs -0 wcx -l
```

```powershell
headx -z -n 2 names.lst
```

Fail a script when a file cannot be read:

```bash
headx -n 1 missing.txt || echox "exit status $?"
```

```powershell
headx -n 1 missing.txt; if ($LASTEXITCODE -ne 0) { "exit status $LASTEXITCODE" }
```

Preview the start of every log in the portable `gnux shell`:

```bash
gnux shell -c 'headx -n 3 *.log'
```

```powershell
gnux shell -c 'headx -n 3 *.log'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
headx --help                # options, first examples, exit status
gnux help headx             # this page: the full manual
gnux help headx --OPTION    # one option in depth, with its examples
```

### See also

- [tailx](../tailx/index.md)
- [catx](../catx/index.md)
- [wcx](../wcx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
