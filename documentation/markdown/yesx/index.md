## yesx

Output a string repeatedly until killed.

Mirrors GNU `yes`. Run it as `yesx`, or as `gnux yesx`.

### Synopsis

```bash
yesx [STRING]...
```

### Description

`yesx` repeatedly writes a line made of every STRING joined by single spaces, or 'y' when none is given, until it is killed or its output is closed. Output is written in large batches of whole lines, so it is fast enough to use as a data source. When the reader goes away (as in `yesx | head`) `yesx` stops quietly; any other write failure is reported and exits with status 1. A first '--' is taken as the end of options and not printed; `--help` and `--version` are honored only when they are the sole argument, and every other argument is repeated as text.

### Permissions

Needs no filesystem permissions; it only writes to standard output.

### Use cases

- Answering every prompt of an interactive command automatically.
- Generating an endless or, with head, a fixed amount of filler input.
- Loading a CPU core or a pipe for a quick throughput test.
- Declining every prompt with '`yesx` n' to see what a command would ask.
- Creating a test file of an exact size with `headx -c`.
- Producing numbered or repeated sample rows for testing other tools.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Answer 'y' to every prompt of a command:

```bash
yesx | rmx -i *.tmp
```

```powershell
yesx | rmx -i (Get-ChildItem -Name *.tmp)
```

Answer 'n' instead:

```bash
yesx n | rmx -i *.tmp
```

```powershell
yesx n | rmx -i (Get-ChildItem -Name *.tmp)
```

Confirm every step of an interactive recursive removal:

```bash
yesx | rmx -ri build
```

```powershell
yesx | rmx -ri build
```

Produce five copies of a line:

```bash
yesx 'all work and no play' | headx -n 5
```

```powershell
yesx 'all work and no play' | headx -n 5
```

Join several arguments into one repeated line:

```bash
yesx a b c | headx -n 2
```

```powershell
yesx a b c | headx -n 2
```

Repeat text that starts with a dash after --:

```bash
yesx -- -n | headx -n 2
```

```powershell
yesx -- -n | headx -n 2
```

Create a file of exactly one mebibyte:

```bash
yesx 0123456789 | headx -c 1M > filler.txt && wcx -c filler.txt
```

```powershell
yesx 0123456789 | headx -c 1M > filler.txt; wcx -c filler.txt
```

Number a few repeated rows:

```bash
yesx row | headx -n 4 | catx -n
```

```powershell
yesx row | headx -n 4 | catx -n
```

Measure pipe throughput by counting 100 MiB:

```bash
yesx | headx -c 100M | wcx -c
```

```powershell
yesx | headx -c 100M | wcx -c
```

Stop cleanly inside a `gnux shell` pipeline:

```bash
gnux shell -c 'yesx hello | headx -n 3 > out.txt; wcx -l out.txt'
```

```powershell
gnux shell -c 'yesx hello | headx -n 3 > out.txt; wcx -l out.txt'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | the reader went away (a closed pipe is the normal way `yesx` ends) |
| 1 | a write error other than a closed pipe |

### Help in the terminal

```bash
yesx --help                # options, first examples, exit status
gnux help yesx             # this page: the full manual
gnux help yesx --OPTION    # one option in depth, with its examples
```

### See also

- [headx](../headx/index.md)
- [echox](../echox/index.md)
- [rmx](../rmx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
