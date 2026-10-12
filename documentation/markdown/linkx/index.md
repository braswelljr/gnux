## linkx

Call the `link` function to create a hard `link` to a file.

Mirrors GNU `link`. Run it as `linkx`, or as `gnux linkx`.

### Synopsis

```bash
linkx FILE1 FILE2
```

### GNU reference

- [GNU coreutils manual: link invocation](https://www.gnu.org/software/coreutils/manual/html_node/link-invocation.html)
- [Source: src/link.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/link.c)

### Description

`linkx` creates FILE2 as a hard `link` to the existing FILE1: a second name for the same file, sharing its contents and metadata. It is the bare `link`(2) system call, with no options: it never overwrites FILE2, never makes a symbolic `link`, and never places the `link` inside a directory. Exactly two operands are required. `lnx` does the same with options; `linkx` is for scripts that want exactly one system call and its error.

### Permissions

Needs write and search permission on the directory that will hold FILE2. FILE1 and FILE2 must be on the same filesystem, and most systems refuse a hard `link` to a directory. On Windows, hard links need an NTFS volume.

### Use cases

- Giving a file a second name without copying its data.
- Creating a hard `link` in a script that must fail rather than overwrite.
- Keeping a snapshot name for a file that is about to be renamed or replaced.
- Testing whether two paths are on the same filesystem (the `link` fails if not).
- Implementing a simple lock: `link` succeeds for exactly one of several racing processes.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Give a file a second name:

```bash
linkx report.txt report-final.txt
```

```powershell
linkx report.txt report-final.txt
```

Refuse to overwrite an existing name (exit status 1):

```bash
linkx a.txt existing.txt || echox 'already there'
```

```powershell
linkx a.txt existing.txt; if ($LASTEXITCODE -ne 0) { 'already there' }
```

Check that both names are the same file (same inode in the first column):

```bash
linkx data.bin data.bak && lsx -i data.bin data.bak
```

```powershell
linkx data.bin data.bak; lsx -i data.bin data.bak
```

Keep the old contents reachable before replacing a file:

```bash
linkx config.yaml config.yaml.old && cpx new.yaml config.yaml
```

```powershell
linkx config.yaml config.yaml.old; cpx new.yaml config.yaml
```

Take a lock that only one process can get:

```bash
touchx lock.$$ && linkx lock.$$ lock && echox 'got the lock'
```

```powershell
touchx lock.$PID; linkx lock.$PID lock; if ($?) { 'got the lock' }
```

Release that lock:

```bash
unlinkx lock
```

```powershell
unlinkx lock
```

Link a file into another directory (FILE2 names the `link`, not the directory):

```bash
linkx build/app dist/app
```

```powershell
linkx build/app dist/app
```

Count a file's names after linking it:

```bash
linkx a b && lsx -l a
```

```powershell
linkx a b; lsx -l a
```

Link a file whose name starts with a dash:

```bash
linkx -- -file copy
```

```powershell
linkx -- -file copy
```

Make a hard `link` in the portable `gnux shell`:

```bash
gnux shell -c 'linkx notes.md notes-link.md'
```

```powershell
gnux shell -c 'linkx notes.md notes-link.md'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
linkx --help                # options, first examples, exit status
gnux help linkx             # this page: the full manual
gnux help linkx --OPTION    # one option in depth, with its examples
```

### See also

- [lnx](../lnx/index.md)
- [unlinkx](../unlinkx/index.md)
- [lsx](../lsx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
