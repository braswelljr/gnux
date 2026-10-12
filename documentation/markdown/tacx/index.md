## tacx

Concatenate and print files in reverse.

Mirrors GNU `tac`. Run it as `tacx`, or as `gnux tacx`.

### Synopsis

```bash
tacx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: tac invocation](https://www.gnu.org/software/coreutils/manual/html_node/tac-invocation.html)
- [Source: src/tac.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/tac.c)

### Description

`tacx` writes each FILE to standard output last line first. With no FILE, or when FILE is -, it reads standard input. Each FILE is reversed on its own, and the files are written in the order given. A record ends with its separator, a newline by default, so a last line without one is joined to the line before it in the output, as in GNU. `-s` STRING uses STRING as the separator instead, and `-b` attaches the separator to the start of the record that follows it rather than to the end of the one before. `-r` treats the separator as a regular expression; `gnux` uses Go's RE2 syntax (close to POSIX extended regular expressions) rather than the GNU regex dialect, and reports an invalid expression in Go's words. Each input is read whole into memory before it is written.

### Permissions

Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

### Use cases

- Reading a log newest entry first.
- Reversing the order of a list or a history file.
- Processing records last to first, such as undoing steps in reverse.
- Reversing records separated by something other than a newline with `-s`.
- Finding the last match in a file by reversing it before searching.

### Flags

```text
  -b, --before             attach the separator before instead of after
      --help               display this help and exit
  -r, --regex              interpret the separator as a regular expression
  -s, --separator STRING   use STRING as the separator instead of newline
      --version            output version information and exit
```

### Examples

Read a log newest entry first:

```bash
tacx app.log
```

```powershell
tacx app.log
```

See the ten most recent log entries, newest at the top:

```bash
tacx app.log | headx
```

```powershell
tacx app.log | headx
```

Count down instead of up:

```bash
seqx 5 | tacx
```

```powershell
seqx 5 | tacx
```

Reverse two logs, each kept separate:

```bash
tacx a.txt b.txt
```

```powershell
tacx a.txt b.txt
```

Reverse items in a comma-separated list:

```bash
printfx 'a,b,c,' | tacx -s ,
```

```powershell
printfx 'a,b,c,' | tacx -s ,
```

Reverse sections of a file that each start with a '---' line:

```bash
tacx -b -s '---' notes.txt
```

```powershell
tacx -b -s '---' notes.txt
```

Reverse records separated by numbers of any length:

```bash
tacx -r -s '[0-9]+' data.txt
```

```powershell
tacx -r -s '[0-9]+' data.txt
```

Keep a newest-first copy of a history file:

```bash
tacx history.txt > reversed.txt
```

```powershell
tacx history.txt > reversed.txt
```

Get the order to undo a list of setup steps:

```bash
tacx steps.txt | catx -n
```

```powershell
tacx steps.txt | catx -n
```

Sort a list descending without a second sort:

```bash
sortx names.txt | tacx
```

```powershell
sortx names.txt | tacx
```

Show the five newest log lines in the portable `gnux shell`:

```bash
gnux shell -c 'tacx app.log | headx -n 5'
```

```powershell
gnux shell -c 'tacx app.log | headx -n 5'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
tacx --help                # options, first examples, exit status
gnux help tacx             # this page: the full manual
gnux help tacx --OPTION    # one option in depth, with its examples
```

### See also

- [catx](../catx/index.md)
- [tailx](../tailx/index.md)
- [sortx](../sortx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
