## teex

Copy standard input to each FILE, and also to standard output.

Mirrors GNU `tee`. Run it as `teex`, or as `gnux teex`.

### Synopsis

```bash
teex [OPTION]... [FILE]...
```

### Description

`teex` reads standard input and writes it, unchanged and as soon as it arrives, to standard output and to every FILE. Each FILE is created if it does not exist and truncated if it does, unless `-a` appends to it instead. A FILE named - is a file called '-', as in GNU tee 9, not standard output. A FILE that cannot be opened is reported and the copy goes on to the others; the exit status is then 1. `--output-error`=MODE says what happens when writing to an output fails: 'warn' reports the error and carries on with the other outputs, 'warn-nopipe' does the same but drops an output whose reader has gone without a word, 'exit' stops at the first error and 'exit-nopipe' stops at the first error that is not a closed pipe. `--output-error` with no MODE, and `-p`, mean 'warn-nopipe'. Without the option a failing output is reported and dropped, and a closed pipe stops `teex` quietly, as SIGPIPE stops GNU tee. `teex` stops reading once no output is left. When `-p` and `--output-error`=MODE are both given, MODE wins whatever their order. `-i` is accepted but has no effect: `gnux` runs tools in-process (also inside `gnux shell` and the playground), so it does not change how the process handles interrupts.

### Permissions

Needs write permission on each existing FILE, or write and search permission on the directory of a FILE it creates. New files get mode 0666 masked by the umask. Without `-a` an existing FILE is truncated before anything is written.

### Use cases

- Saving a command's output to a log while still watching it on the terminal.
- Keeping a copy of an intermediate stage of a pipeline for debugging.
- Appending each run of a build or test to a running log with `-a`.
- Writing the same data to several files at once.
- Capturing CI output as an artifact while it streams to the job log.
- Keeping a pipeline going when a downstream reader exits early, with `-p`.

### Flags

```text
  -a, --append                              append to the given FILEs, do not overwrite
      --help                                display this help and exit
  -i, --ignore-interrupts                   ignore interrupt signals (accepted; no effect in gnux)
      --output-error MODE[="warn-nopipe"]   set behavior on write error; MODE can be: warn, warn-nopipe, exit, exit-nopipe
  -p, --output-error-warn-nopipe            operate in a more appropriate MODE with pipes
      --version                             output version information and exit
```

### Examples

Show output and save it to a file:

```bash
lsx -l | teex listing.txt
```

```powershell
lsx -l | teex listing.txt
```

Append to a log instead of overwriting it:

```bash
echox 'build started' | teex -a build.log
```

```powershell
'build started' | teex -a build.log
```

Write the same input to several files:

```bash
echox 'v1.2.3' | teex VERSION dist/VERSION
```

```powershell
'v1.2.3' | teex VERSION dist/VERSION
```

Save a copy of an intermediate pipeline stage:

```bash
catx access.log | teex raw.txt | wcx -l
```

```powershell
catx access.log | teex raw.txt | wcx -l
```

Write a file without echoing it to the terminal:

```bash
echox 'KEY=value' | teex .env > /dev/null
```

```powershell
'KEY=value' | teex .env > $null
```

Save test output while still seeing it:

```bash
go test ./... 2>&1 | teex test.log
```

```powershell
go test ./... 2>&1 | teex test.log
```

Keep going when the reader stops early, without an error:

```bash
yesx | teex -p copy.txt | headx -n 3
```

```powershell
yesx | teex -p copy.txt | headx -n 3
```

Report a closed pipe as an error and stop:

```bash
yesx | teex --output-error=exit copy.txt | headx -n 1
```

```powershell
yesx | teex --output-error=exit copy.txt | headx -n 1
```

Carry on to the other files when one cannot be written:

```bash
echox data | teex --output-error=warn a.txt missing/b.txt c.txt
```

```powershell
'data' | teex --output-error=warn a.txt missing/b.txt c.txt
```

Write to a file literally named '-':

```bash
echox data | teex -
```

```powershell
'data' | teex -
```

Log a script's steps in the portable `gnux shell`:

```bash
gnux shell -c 'lsx -1 | teex -a files.log | wcx -l'
```

```powershell
gnux shell -c 'lsx -1 | teex -a files.log | wcx -l'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
teex --help                # options, first examples, exit status
gnux help teex             # this page: the full manual
gnux help teex --OPTION    # one option in depth, with its examples
```

### See also

- [catx](../catx/index.md)
- [headx](../headx/index.md)
- [tailx](../tailx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
