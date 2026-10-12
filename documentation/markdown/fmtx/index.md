## fmtx

Reformat each paragraph of text.

Mirrors GNU `fmt`. Run it as `fmtx`, or as `gnux fmtx`.

### Synopsis

```bash
fmtx [-WIDTH] [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: fmt invocation](https://www.gnu.org/software/coreutils/manual/html_node/fmt-invocation.html)
- [Source: src/fmt.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/fmt.c)

### Description

`fmtx` fills and joins the lines of each paragraph so that they come as close as possible to a goal width without exceeding the maximum width, and writes the result to standard output. With no FILE, or when FILE is -, it reads standard input. Paragraphs are separated by blank lines and by changes of indentation; the indentation of each paragraph is kept. The line breaks are chosen as GNU `fmt` chooses them, minimizing raggedness over the whole paragraph and avoiding breaking after the first word of a sentence or before its last. `-w` WIDTH sets the maximum width (75 by default, at most 2500), also written `-WIDTH`; `-g` WIDTH sets the goal (93% of the maximum by default, and without `-w` the maximum becomes the goal plus 10). `-c` (crown margin) keeps the indentation of the first two lines and aligns the rest with the second; `-t` (tagged paragraph) is the same, except that a first line indented like the second is a paragraph of its own. `-s` only splits long lines and never joins short ones, which suits code. `-u` puts one space between words and two after sentences. `-p` STRING reformats only the lines that begin with STRING (after any indentation) and puts it back on each output line; the other lines are copied unchanged. A sentence ends at '.', '?' or '!' followed by the end of the line or two spaces, as in GNU `fmt`.

### Permissions

Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

### Use cases

- Rewrapping a commit message, README paragraph or email to a fixed width.
- Joining hard-wrapped text into evenly filled lines.
- Reflowing comment blocks with `-p` '#' or `-p` '//'.
- Splitting overlong lines without joining short ones with `-s`.
- Normalizing spacing between words and sentences with `-u`.

### Flags

```text
  -c, --crown-margin       preserve indentation of first two lines
  -g, --goal string        goal width (default of 93% of width)
      --help               display this help and exit
  -p, --prefix STRING      reformat only lines beginning with STRING, reattaching the prefix to reformatted lines
  -s, --split-only         split long lines, but do not refill
  -t, --tagged-paragraph   indentation of first line different from second
  -u, --uniform-spacing    one space between words, two after sentences
      --version            output version information and exit
  -w, --width WIDTH        maximum line width (default of 75 columns)
```

### Examples

Reformat a file to the default 75 columns:

```bash
fmtx notes.txt
```

```powershell
fmtx notes.txt
```

Reformat to 72 columns:

```bash
fmtx -w 72 message.txt
```

```powershell
fmtx -w 72 message.txt
```

Use the obsolete width form:

```bash
fmtx -60 notes.txt
```

```powershell
fmtx -60 notes.txt
```

Split long lines but never `join` short ones:

```bash
fmtx -s -w 80 script.txt
```

```powershell
fmtx -s -w 80 script.txt
```

Reflow a shell comment block:

```bash
fmtx -p '#' -w 72 config.sh
```

```powershell
fmtx -p '#' -w 72 config.sh
```

Use uniform spacing:

```bash
fmtx -u draft.txt
```

```powershell
fmtx -u draft.txt
```

Keep a hanging indent:

```bash
fmtx -t list.txt
```

```powershell
fmtx -t list.txt
```

Keep the first two lines' indentation:

```bash
fmtx -c letter.txt
```

```powershell
fmtx -c letter.txt
```

Set the goal width as well as the maximum:

```bash
fmtx -w 80 -g 70 notes.txt
```

```powershell
fmtx -w 80 -g 70 notes.txt
```

Reformat standard input:

```bash
catx a.txt b.txt | fmtx -w 66
```

```powershell
catx a.txt b.txt | fmtx -w 66
```

Save a reformatted copy:

```bash
fmtx -w 72 draft.txt > final.txt
```

```powershell
fmtx -w 72 draft.txt > final.txt
```

Reformat in the portable `gnux shell`:

```bash
gnux shell -c 'fmtx -w 60 notes.txt | headx'
```

```powershell
gnux shell -c 'fmtx -w 60 notes.txt | headx'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
fmtx --help                # options, first examples, exit status
gnux help fmtx             # this page: the full manual
gnux help fmtx --OPTION    # one option in depth, with its examples
```

### See also

- [foldx](../foldx/index.md)
- [nlx](../nlx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
