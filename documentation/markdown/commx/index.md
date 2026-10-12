## commx

Compare two sorted files line by line.

Mirrors GNU `comm`. Run it as `commx`, or as `gnux commx`.

### Synopsis

```bash
commx [OPTION]... FILE1 FILE2
```

### GNU reference

- [GNU coreutils manual: comm invocation](https://www.gnu.org/software/coreutils/manual/html_node/comm-invocation.html)
- [Source: src/comm.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/comm.c)

### Description

`commx` compares two sorted files, FILE1 and FILE2, line by line. With no options it prints three columns: lines only in FILE1, lines only in FILE2, and lines in both, each column indented by one more TAB than the one before. -1, -2 and -3 suppress the matching column, so -12 prints the common lines and -23 the lines only in FILE1. Either FILE may be - for standard input. Both files must be sorted: by default `commx` warns once per file when it finds a line out of order after a line that appears in only one file, and then exits with status 1; `--check-order` fails at the first line out of order, and `--nocheck-order` never checks. `--output-delimiter`=STR separates the columns with STR (an empty STR writes a NUL byte), `--total` adds a final line with the three counts and the word 'total', and `-z` makes NUL the line delimiter. Lines are compared byte by byte, as in the C locale, whatever LANG says, so sort the inputs with `sortx` (or LC_ALL=C sort). The framework requires a long name for every option, so -1, -2 and -3 also answer to `gnux`'s own `--suppress-file1`, `--suppress-file2` and `--suppress-common`. When both `--check-order` and `--nocheck-order` are given, `--check-order` wins.

### Permissions

Needs read permission on FILE1 and FILE2 and search permission on the directories leading to them. Nothing is ever modified.

### Use cases

- Finding the lines two sorted lists have in common with -12.
- Finding what was added to or removed from a list between two versions.
- Comparing the files in two directories by comparing their sorted listings.
- Checking which required packages or users are missing from an inventory.
- Counting how many lines two lists share with `--total`.

### Flags

```text
      --check-order            check that the input is correctly sorted, even if all input lines are pairable
      --help                   display this help and exit
      --nocheck-order          do not check that the input is correctly sorted
      --output-delimiter STR   separate columns with STR
  -3, --suppress-common        suppress column 3 (lines that appear in both files)
  -1, --suppress-file1         suppress column 1 (lines unique to FILE1)
  -2, --suppress-file2         suppress column 2 (lines unique to FILE2)
      --total                  output a summary
      --version                output version information and exit
  -z, --zero-terminated        line delimiter is NUL, not newline
```

### Examples

Show the three columns for two sorted files:

```bash
commx old.txt new.txt
```

```powershell
commx old.txt new.txt
```

Print only the lines the files have in common:

```bash
commx -12 old.txt new.txt
```

```powershell
commx -12 old.txt new.txt
```

Print the lines only in the first file (removed lines):

```bash
commx -23 old.txt new.txt
```

```powershell
commx -23 old.txt new.txt
```

Print the lines only in the second file (added lines):

```bash
commx -13 old.txt new.txt
```

```powershell
commx -13 old.txt new.txt
```

Compare two unsorted lists by sorting them first:

```bash
sortx a.txt > a.sorted && sortx b.txt > b.sorted && commx a.sorted b.sorted
```

```powershell
sortx a.txt > a.sorted; sortx b.txt > b.sorted; commx a.sorted b.sorted
```

Compare standard input with a file:

```bash
lsx | commx - expected.txt
```

```powershell
lsx | commx - expected.txt
```

Separate the columns with a comma:

```bash
commx --output-delimiter=, old.txt new.txt
```

```powershell
commx --output-delimiter=, old.txt new.txt
```

Count the lines unique to each file and common to both:

```bash
commx --total -123 old.txt new.txt
```

```powershell
commx --total -123 old.txt new.txt
```

Fail at the first line out of order:

```bash
commx --check-order old.txt new.txt || echox 'not sorted'
```

```powershell
commx --check-order old.txt new.txt; if ($LASTEXITCODE -ne 0) { 'not sorted' }
```

Skip the order check:

```bash
commx --nocheck-order old.txt new.txt
```

```powershell
commx --nocheck-order old.txt new.txt
```

Compare NUL-terminated lists:

```bash
commx -z -12 a.lst b.lst
```

```powershell
commx -z -12 a.lst b.lst
```

Find common lines in the portable `gnux shell`:

```bash
gnux shell -c 'commx -12 old.txt new.txt | wcx -l'
```

```powershell
gnux shell -c 'commx -12 old.txt new.txt | wcx -l'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
commx --help                # options, first examples, exit status
gnux help commx             # this page: the full manual
gnux help commx --OPTION    # one option in depth, with its examples
```

### See also

- [sortx](../sortx/index.md)
- [uniqx](../uniqx/index.md)
- [joinx](../joinx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
