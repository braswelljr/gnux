## uniqx

Report or omit repeated lines.

Mirrors GNU `uniq`. Run it as `uniqx`, or as `gnux uniqx`.

### Synopsis

```bash
uniqx [OPTION]... [INPUT [OUTPUT]]
```

### GNU reference

- [GNU coreutils manual: uniq invocation](https://www.gnu.org/software/coreutils/manual/html_node/uniq-invocation.html)
- [Source: src/uniq.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/uniq.c)

### Description

`uniqx` filters adjacent matching lines from INPUT (or standard input), writing to OUTPUT (or standard output). With no options, matching lines are merged to the first occurrence. Only adjacent lines are compared, so a file is usually sorted first. `-c` prefixes each line with its number of occurrences, `-d` prints one copy of each repeated line, `-u` only the lines that are not repeated, and `-D` every line of each repeated group, with `--all-repeated`=METHOD (none, prepend, separate) putting an empty line before or between the groups. `--group`[=METHOD] (separate, prepend, append, both) prints every line with the groups delimited by an empty line, and cannot be combined with `-c`, `-d`, `-D` or `-u`; `-D` cannot be combined with `-c` either. The comparison skips the first N fields with `-f` (a field is a run of blanks then non-blanks), then N characters with `-s`, and compares at most N characters with `-w`. `-z` makes NUL the line delimiter, and the output always ends each line with it. Deviations from GNU: characters for `-s` and `-w` are bytes, as in GNU's C locale, and `-i` folds ASCII letters only. The obsolete `-N` and +N forms of `-f` and `-s` are not accepted. METHOD arguments may be abbreviated, as in GNU.

### Permissions

Needs read permission on INPUT and search permission on the directories leading to it. When OUTPUT is given it is created or truncated, which needs write permission on it if it exists, or on its directory if it does not. INPUT itself is never modified.

### Use cases

- Removing duplicate lines from sorted output.
- Counting how often each value occurs with sort and `-c`, a frequency table.
- Listing only the values that appear more than once with `-d`.
- Finding the values that appear exactly once with `-u`.
- Grouping related lines visually with `--group`.
- Comparing lines while ignoring a leading timestamp or ID with `-f` or `-s`.
- Treating lines as equal when they share a prefix with `-w`.
- Deduplicating NUL-separated file lists safely with `-z`.

### Flags

```text
  -D, --all-repeated METHOD[="none"]   print all duplicate lines; groups can be delimited with an empty line; METHOD is none (the default), prepend or separate
  -w, --check-chars N                  compare no more than N characters in lines
  -c, --count                          prefix lines by the number of occurrences
      --group METHOD[="separate"]      show all items, separating groups with an empty line; METHOD is separate (the default), prepend, append or both
      --help                           display this help and exit
  -i, --ignore-case                    ignore differences in case when comparing
  -d, --repeated                       only print duplicate lines, one for each group
  -s, --skip-chars N                   avoid comparing the first N characters
  -f, --skip-fields N                  avoid comparing the first N fields
  -u, --unique                         only print unique lines
      --version                        output version information and exit
  -z, --zero-terminated                line delimiter is NUL, not newline
```

### Examples

Remove adjacent duplicate lines:

```bash
uniqx names.txt
```

```powershell
uniqx names.txt
```

Write the result to another file:

```bash
uniqx names.txt unique-names.txt
```

```powershell
uniqx names.txt unique-names.txt
```

Count the occurrences of each line:

```bash
uniqx -c sorted.txt
```

```powershell
uniqx -c sorted.txt
```

Print only the lines that are repeated:

```bash
uniqx -d sorted.txt
```

```powershell
uniqx -d sorted.txt
```

Print only the lines that are not repeated:

```bash
uniqx -u sorted.txt
```

```powershell
uniqx -u sorted.txt
```

Print every repeated line, the groups separated by an empty line:

```bash
uniqx --all-repeated=separate sorted.txt
```

```powershell
uniqx --all-repeated=separate sorted.txt
```

Print all lines with each group of equal lines set apart:

```bash
uniqx --group sorted.txt
```

```powershell
uniqx --group sorted.txt
```

Compare without regard to case:

```bash
uniqx -i words.txt
```

```powershell
uniqx -i words.txt
```

Ignore the first field (a timestamp, say) when comparing:

```bash
uniqx -f 1 events.log
```

```powershell
uniqx -f 1 events.log
```

Ignore the first 4 characters when comparing:

```bash
uniqx -s 4 ids.txt
```

```powershell
uniqx -s 4 ids.txt
```

Treat lines that share their first 8 characters as equal:

```bash
uniqx -w 8 commits.txt
```

```powershell
uniqx -w 8 commits.txt
```

Deduplicate a NUL-separated list:

```bash
uniqx -z files.lst
```

```powershell
uniqx -z files.lst
```

Remove duplicates from a pipeline's output:

```bash
catx a.txt b.txt | uniqx
```

```powershell
catx a.txt b.txt | uniqx
```

Count the distinct lines of a file:

```bash
uniqx sorted.txt | wcx -l
```

```powershell
uniqx sorted.txt | wcx -l
```

Deduplicate in the portable `gnux shell`:

```bash
gnux shell -c 'catx sorted.txt | uniqx -c'
```

```powershell
gnux shell -c 'catx sorted.txt | uniqx -c'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
uniqx --help                # options, first examples, exit status
gnux help uniqx             # this page: the full manual
gnux help uniqx --OPTION    # one option in depth, with its examples
```

### See also

- [cutx](../cutx/index.md)
- [wcx](../wcx/index.md)
- [catx](../catx/index.md)
- [headx](../headx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
