## foldx

Wrap each input line to fit in a specified width.

Mirrors GNU `fold`. Run it as `foldx`, or as `gnux foldx`.

### Synopsis

```bash
foldx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: fold invocation](https://www.gnu.org/software/coreutils/manual/html_node/fold-invocation.html)
- [Source: src/fold.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/fold.c)

### Description

`foldx` writes each FILE to standard output, breaking any line longer than 80 columns into several. With no FILE, or when FILE is -, it reads standard input. `-w` WIDTH sets the width; the obsolete form `-WIDTH` (`foldx` -72) is accepted too, and whichever is given last wins. Columns are counted as a terminal would: a tab advances to the next multiple of 8, a backspace moves back one column and a carriage return goes back to column 0. `-b` counts bytes instead, so every byte takes one position. `-s` breaks a long line after the last blank that fits, rather than exactly at the width, so words are not `split` unless one is longer than the width. Like GNU `fold` in the C locale, `foldx` counts bytes rather than characters, so a multibyte UTF-8 character takes more than one column.

### Permissions

Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

### Use cases

- Wrapping long log lines or generated text to fit a terminal or an email.
- Splitting a long `base64` or hex string into fixed-width lines.
- Breaking prose at word boundaries with `-s`.
- Chunking a single-line file into records of a fixed number of bytes with `-b`.
- Preparing text for a fixed-width display or printer.

### Flags

```text
  -b, --bytes         count bytes rather than columns
      --help          display this help and exit
  -s, --spaces        break at spaces
      --version       output version information and exit
  -w, --width WIDTH   use WIDTH columns instead of 80
```

### Examples

Make a log with very long lines readable in an 80-column terminal:

```bash
foldx notes.txt
```

```powershell
foldx notes.txt
```

Fit text into a narrow 40-column display or receipt printer:

```bash
foldx -w 40 notes.txt
```

```powershell
foldx -w 40 notes.txt
```

Wrap a commit message body at 72 columns without breaking words:

```bash
foldx -s -w 72 essay.txt
```

```powershell
foldx -s -w 72 essay.txt
```

Run an old script that uses the obsolete width form:

```bash
foldx -60 notes.txt
```

```powershell
foldx -60 notes.txt
```

Break a one-line key or `base64` blob into 64-byte lines:

```bash
foldx -b -w 64 key.txt
```

```powershell
foldx -b -w 64 key.txt
```

Wrap the output of another command:

```bash
catx long.log | foldx -w 100
```

```powershell
catx long.log | foldx -w 100
```

Split a word into one character per line, to count or sort letters:

```bash
echox abc | foldx -w 1
```

```powershell
echox abc | foldx -w 1
```

Save a wrapped copy of a draft for an email client that does not wrap:

```bash
foldx -s -w 72 draft.txt > wrapped.txt
```

```powershell
foldx -s -w 72 draft.txt > wrapped.txt
```

Wrap several chapter files in one go:

```bash
foldx -w 72 a.txt b.txt
```

```powershell
foldx -w 72 a.txt b.txt
```

Check how many screen lines a note will take once wrapped:

```bash
foldx -w 72 notes.txt | wcx -l
```

```powershell
foldx -w 72 notes.txt | wcx -l
```

Preview a wrapped file in the portable `gnux shell`, the same on every OS:

```bash
gnux shell -c 'foldx -s -w 60 notes.txt | headx'
```

```powershell
gnux shell -c 'foldx -s -w 60 notes.txt | headx'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
foldx --help                # options, first examples, exit status
gnux help foldx             # this page: the full manual
gnux help foldx --OPTION    # one option in depth, with its examples
```

### See also

- [fmtx](../fmtx/index.md)
- [cutx](../cutx/index.md)
- [expandx](../expandx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
