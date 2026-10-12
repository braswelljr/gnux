## expandx

Convert tabs to spaces.

Mirrors GNU `expand`. Run it as `expandx`, or as `gnux expandx`.

### Synopsis

```bash
expandx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: expand invocation](https://www.gnu.org/software/coreutils/manual/html_node/expand-invocation.html)
- [Source: src/expand.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/expand.c)

### Description

`expandx` writes each FILE to standard output with every tab replaced by the spaces that reach the next tab stop. With no FILE, or when FILE is -, it reads standard input. Tab stops are every 8 columns unless `-t` says otherwise: `-t` N puts them every N columns, and `-t` LIST (numbers separated by commas or blanks, in ascending order) puts them at those columns; the last number may be written /N for a stop at every multiple of N after it, or +N for a stop every N columns after it. Past the last listed stop a tab becomes one space. Several `-t` options add to one list, and the obsolete `-N` form (`expandx` -4) is a list of its own, in the order given; GNU's comma form -4,8 is not accepted, and the digits of two `-N` options in a row run together (-4 -8 reads as -48), so write `-t` 4,8 instead. `-i` converts only the tabs before the first non-blank character of each line. A backspace moves back one column. Like GNU `expand` in the C locale, `expandx` counts bytes rather than characters.

### Permissions

Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

### Use cases

- Converting tab-indented source to spaces for a style guide.
- Making tab-separated output line up the same way in every viewer.
- Preparing text for tools that count characters, like `cutx -c` or `foldx`.
- Converting only the indentation with `-i`, leaving tabs inside lines alone.
- Laying out columns at specific positions with `-t` LIST.

### Flags

```text
      --help           display this help and exit
  -i, --initial        do not convert tabs after non blanks
  -t, --tabs N, LIST   have tabs N characters apart, not 8, or use comma separated list of explicit tab positions
      --version        output version information and exit
```

### Examples

Convert tabs to spaces with stops every 8 columns:

```bash
expandx main.c
```

```powershell
expandx main.c
```

Use 4-column tab stops:

```bash
expandx -t 4 main.go
```

```powershell
expandx -t 4 main.go
```

Use the obsolete form:

```bash
expandx -4 main.go
```

```powershell
expandx -4 main.go
```

Put tab stops at specific columns:

```bash
expandx -t 10,20,40 table.tsv
```

```powershell
expandx -t 10,20,40 table.tsv
```

Stops at 4 and then every multiple of 8:

```bash
expandx -t 4,/8 data.txt
```

```powershell
expandx -t 4,/8 data.txt
```

Convert only leading tabs:

```bash
expandx -i -t 2 config.yml
```

```powershell
expandx -i -t 2 config.yml
```

Convert standard input:

```bash
printfx 'a\tb\n' | expandx
```

```powershell
printfx 'a\tb\n' | expandx
```

Save a converted copy:

```bash
expandx -t 4 tabs.py > spaces.py
```

```powershell
expandx -t 4 tabs.py > spaces.py
```

Line up tab-separated columns:

```bash
expandx -t 16 report.tsv | headx
```

```powershell
expandx -t 16 report.tsv | headx
```

Convert several files as one stream:

```bash
expandx a.txt b.txt > combined.txt
```

```powershell
expandx a.txt b.txt > combined.txt
```

Convert in the portable `gnux shell`:

```bash
gnux shell -c 'expandx -t 4 main.go | wcx -L'
```

```powershell
gnux shell -c 'expandx -t 4 main.go | wcx -L'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
expandx --help                # options, first examples, exit status
gnux help expandx             # this page: the full manual
gnux help expandx --OPTION    # one option in depth, with its examples
```

### See also

- [unexpandx](../unexpandx/index.md)
- [foldx](../foldx/index.md)
- [trx](../trx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
