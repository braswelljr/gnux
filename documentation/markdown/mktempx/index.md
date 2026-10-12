## mktempx

Create a temporary file or directory, safely, and print its name.

Mirrors GNU `mktemp`. Run it as `mktempx`, or as `gnux mktempx`.

### Synopsis

```bash
mktempx [OPTION]... [TEMPLATE]
```

### GNU reference

- [GNU coreutils manual: mktemp invocation](https://www.gnu.org/software/coreutils/manual/html_node/mktemp-invocation.html)
- [Source: src/mktemp.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/mktemp.c)

### Description

`mktempx` creates a new, empty file (or a directory with `-d`) whose name is TEMPLATE with its run of trailing X's replaced by random letters and digits, and prints the name. TEMPLATE must contain at least 3 consecutive X's at the end of the name; whatever follows the last X is kept as a suffix, and `--suffix` adds one explicitly. With no TEMPLATE, tmp.XXXXXXXXXX is used and the file is created in the temporary directory. A relative TEMPLATE is resolved against the current directory unless `-p` DIR or `--tmpdir`[=DIR] is given; then it goes under DIR, else $TMPDIR, else the system temporary directory (/tmp on Unix, the user's temporary folder on Windows), and must not be absolute. The file is created with mode 600 and the directory with mode 700, and creation is atomic: an existing name is never reused. `-u` only prints a name that did not exist at the time, which is unsafe. `-q` suppresses the diagnostic when creation fails. `-t` is the deprecated form: TEMPLATE is a single name component under $TMPDIR, else `-p` DIR, else the temporary directory. `-p` and `-t` also answer to the `gnux` long names `--dir` and `--in-tmpdir`.

### Permissions

Needs write and search permission on the directory the name is created in. The new file is readable and writable by its owner only (600), and a new directory is accessible by its owner only (700), whatever the umask. On Windows only the read-only attribute is representable, so the entry inherits the folder's access control list.

### Use cases

- Getting a scratch file in a script that cannot collide with another run.
- Creating a private working directory with `-d` and removing it at the end.
- Staging output before moving it into place atomically with `mvx`.
- Keeping temporary files of a job together under a directory of your choice with `-p`.
- Giving a temporary file the extension a tool expects with `--suffix`.
- Writing portable scripts that do not hard-code /tmp, which does not exist on Windows.
- Generating a unique name for a lock or marker file.
- Reserving a name without creating it with `-u`, when another program must create it.

### Flags

```text
  -p, --dir DIR                  use DIR as the directory, like --tmpdir=DIR
  -d, --directory                create a directory, not a file
  -u, --dry-run                  do not create anything; merely print a name (unsafe)
      --help                     display this help and exit
  -t, --in-tmpdir                interpret TEMPLATE as a single file name component, relative to $TMPDIR, else DIR (-p), else the temporary directory (deprecated)
  -q, --quiet                    suppress diagnostics about file/dir-creation failure
      --suffix SUFF              append SUFF to TEMPLATE; SUFF must not contain a slash
      --tmpdir DIR[="$TMPDIR"]   interpret TEMPLATE relative to DIR; if DIR is not given, use $TMPDIR if set, else the temporary directory
      --version                  output version information and exit
```

### Examples

Create a temporary file and print its name:

```bash
mktempx
```

```powershell
mktempx
```

Create a temporary directory:

```bash
mktempx -d
```

```powershell
mktempx -d
```

Use the name in a script and clean up afterwards:

```bash
tmp=$(mktempx) && echox hello > "$tmp" && catx "$tmp" && rmx "$tmp"
```

```powershell
$tmp = mktempx; echox hello > $tmp; catx $tmp; rmx $tmp
```

Work in a private scratch directory:

```bash
work=$(mktempx -d) && cpx -r src "$work" && rmx -r "$work"
```

```powershell
$work = mktempx -d; cpx -r src $work; rmx -r $work
```

Create a file from a template in the current directory:

```bash
mktempx build.XXXXXX
```

```powershell
mktempx build.XXXXXX
```

Create it under the temporary directory instead:

```bash
mktempx --tmpdir build.XXXXXX
```

```powershell
mktempx --tmpdir build.XXXXXX
```

Create it under a directory of your choice:

```bash
mktempx -p out report.XXXX
```

```powershell
mktempx -p out report.XXXX
```

Give the file an extension:

```bash
mktempx --suffix=.json data.XXXXXX
```

```powershell
mktempx --suffix=.json data.XXXXXX
```

Keep a suffix written after the X's:

```bash
mktempx notes.XXXX.md
```

```powershell
mktempx notes.XXXX.md
```

Print an unused name without creating anything:

```bash
mktempx -u
```

```powershell
mktempx -u
```

Fail quietly and handle it yourself:

```bash
mktempx -q -p /nonexistent || echox 'no temp dir'
```

```powershell
mktempx -q -p C:\nonexistent; if ($LASTEXITCODE -ne 0) { echox 'no temp dir' }
```

Use the deprecated `-t` form, relative to $TMPDIR:

```bash
mktempx -t job.XXXXXX
```

```powershell
mktempx -t job.XXXXXX
```

Create a temporary file in the portable `gnux shell`:

```bash
gnux shell -c 'mktempx -p . scratch.XXXX'
```

```powershell
gnux shell -c 'mktempx -p . scratch.XXXX'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
mktempx --help                # options, first examples, exit status
gnux help mktempx             # this page: the full manual
gnux help mktempx --OPTION    # one option in depth, with its examples
```

### See also

- [mkdirx](../mkdirx/index.md)
- [touchx](../touchx/index.md)
- [rmx](../rmx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
