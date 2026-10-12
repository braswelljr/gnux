## basenamex

Strip the directory and suffix from file names.

Mirrors GNU `basename`. Run it as `basenamex`, or as `gnux basenamex`.

### Synopsis

```bash
basenamex [OPTION]... NAME [SUFFIX]
```

### GNU reference

- [GNU coreutils manual: basename invocation](https://www.gnu.org/software/coreutils/manual/html_node/basename-invocation.html)
- [Source: src/basename.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/basename.c)

### Description

`basenamex` prints NAME with any leading directory components removed. Trailing slashes are ignored, so 'dir/' yields 'dir', and a name made only of slashes yields the root. If SUFFIX is given and NAME ends with it, the suffix is removed too - unless that would leave nothing. With `-a` every operand is a NAME; `-s` supplies the suffix for all of them and implies `-a`. `-z` separates the output with NUL instead of newlines, for names that may contain newlines.

### Permissions

`basenamex` works on the text of its operands only: it never touches the filesystem, so the names need not exist and no permissions are required.

### Use cases

- Getting a script's own name from $0.
- Deriving an output file name by dropping the directory and extension of an input.
- Stripping one extension from a whole batch of paths with `-s`.
- Naming a directory after an archive, minus its .tar.gz.
- Renaming or copying files in a loop while keeping their base names.
- Printing several base names at once with `-a`, without a suffix.
- Producing NUL-separated output with `-z` for names that may contain newlines.

### Flags

```text
      --help            display this help and exit
  -a, --multiple        support multiple arguments and treat each as a NAME
  -s, --suffix SUFFIX   remove a trailing SUFFIX; implies -a
      --version         output version information and exit
  -z, --zero            end each output line with NUL, not newline
```

### Examples

Print the last component of a path:

```bash
basenamex /usr/bin/sort
```

```powershell
basenamex C:\Windows\System32\sort.exe
```

Ignore trailing slashes:

```bash
basenamex /home/user/
```

```powershell
basenamex C:\Users\user\
```

Drop the directory and the extension:

```bash
basenamex include/stdio.h .h
```

```powershell
basenamex include\stdio.h .h
```

Remove a multi-part extension:

```bash
basenamex --suffix=.tar.gz release/gnux-1.0.tar.gz
```

```powershell
basenamex --suffix=.tar.gz release\gnux-1.0.tar.gz
```

Keep the name when removing the suffix would leave nothing:

```bash
basenamex .h .h
```

```powershell
basenamex .h .h
```

Treat every operand as a NAME:

```bash
basenamex -a /var/log/syslog /etc/hosts
```

```powershell
basenamex -a C:\Windows\win.ini C:\Windows\System32\drivers\etc\hosts
```

Strip a suffix from several names at once:

```bash
basenamex -s .go cmd/main.go core/core.go
```

```powershell
basenamex -s .go cmd\main.go core\core.go
```

Strip an extension from every match of a glob:

```bash
basenamex -s .jpg photos/*.jpg
```

```powershell
basenamex -s .jpg (Get-ChildItem photos\*.jpg)
```

Separate the output with NUL bytes instead of newlines:

```bash
basenamex -a -z a/b c/d | od -c
```

```powershell
basenamex -a -z a\b c\d | Format-Hex
```

Print the name of the running script:

```bash
basenamex "$0"
```

```powershell
basenamex $PSCommandPath
```

Back up every photo under its base name:

```bash
for f in photos/*.jpg; do cpx "$f" "out/$(basenamex "$f" .jpg).bak"; done
```

```powershell
Get-ChildItem photos\*.jpg | ForEach-Object { cpx $_.FullName "out\$(basenamex $_.Name .jpg).bak" }
```

Create a directory named after an archive:

```bash
name=$(basenamex -s .tar.gz gnux-1.0.tar.gz) && mkdirx "$name"
```

```powershell
$name = basenamex -s .tar.gz gnux-1.0.tar.gz; mkdirx $name
```

Strip a directory and extension from several logs with the `gnux shell`:

```bash
gnux shell -c 'basenamex -s .log /var/log/app.log /var/log/db.log'
```

```powershell
gnux shell -c 'basenamex -s .log /var/log/app.log /var/log/db.log'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
basenamex --help                # options, first examples, exit status
gnux help basenamex             # this page: the full manual
gnux help basenamex --OPTION    # one option in depth, with its examples
```

### See also

- [dirnamex](../dirnamex/index.md)
- [realpathx](../realpathx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
