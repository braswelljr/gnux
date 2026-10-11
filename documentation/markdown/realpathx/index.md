## realpathx

Print the resolved absolute file name.

Mirrors GNU `realpath`. Run it as `realpathx`, or as `gnux realpathx`.

### Synopsis

```bash
realpathx [OPTION]... FILE...
```

### Description

`realpathx` prints the canonical absolute name of each FILE: every symbolic link is expanded and every '.', '..' and repeated separator is removed. By default all components but the last must exist; `-e` requires the whole name to exist and `-m` requires none of it to. Links are resolved as they are met (`-P`, the default), so '..' after a link refers to the parent of the link's target; `-L` collapses '..' first, and `-s` does not expand links at all. `--relative-to` prints each name relative to a directory, and `--relative-base` does so only for names under that directory, leaving the rest absolute. A name that cannot be resolved is reported (unless `-q`) and the remaining names are still processed.

### Permissions

Resolving a name needs search permission on every directory leading to it, including those reached through symbolic links. The files themselves are never opened, so no read permission on them is required.

### Use cases

- Finding where a symlinked command or directory really lives.
- Normalizing user-supplied paths before comparing or storing them.
- Computing a path relative to a project root with `--relative-to`.
- Building the absolute name of a file that is about to be created.
- Checking that a path exists in a script with `-e` and the exit status.
- Cleaning up '.', '..' and doubled separators without touching symbolic links (`-s`).
- Choosing how '..' after a symbolic link is read: physically (`-P`) or logically (`-L`).
- Emitting NUL-terminated names with `-z` for names that may contain newlines.

### Flags

```text
  -e, --canonicalize-existing   all components of the path must exist
  -m, --canonicalize-missing    no path components need exist or be a directory
      --help                    display this help and exit
  -L, --logical                 resolve '..' components before symlinks
      --no-symlinks             don't expand symlinks (same as --strip)
  -P, --physical                resolve symlinks as encountered (default)
  -q, --quiet                   suppress most error messages
      --relative-base DIR       print absolute paths unless paths below DIR
      --relative-to DIR         print the resolved path relative to DIR
  -s, --strip                   don't expand symlinks
      --version                 output version information and exit
  -z, --zero                    end each output line with NUL, not newline
```

### Examples

Resolve a path to its canonical absolute name:

```bash
realpathx ../lib/./util.go
```

```powershell
realpathx ..\lib\.\util.go
```

Print the absolute name of the current directory:

```bash
realpathx .
```

```powershell
realpathx .
```

Follow a symbolic link to the file it points to:

```bash
realpathx bin/tool
```

```powershell
realpathx bin\tool
```

Require the file to exist:

```bash
realpathx -e /usr/bin/python3
```

```powershell
realpathx -e $env:WINDIR\System32\cmd.exe
```

Test for existence quietly and branch on the exit status:

```bash
realpathx -q -e config.yaml >/dev/null || echox 'config.yaml is missing'
```

```powershell
realpathx -q -e config.yaml > $null; if ($LASTEXITCODE -ne 0) { echox 'config.yaml is missing' }
```

Normalize a path that does not exist yet:

```bash
realpathx -m build/out/../report.txt
```

```powershell
realpathx -m build\out\..\report.txt
```

Clean up a path without expanding symbolic links:

```bash
realpathx -s bin/tool
```

```powershell
realpathx -s bin\tool
```

Same thing, spelled out:

```bash
realpathx --no-symlinks bin/tool
```

```powershell
realpathx --no-symlinks bin\tool
```

Read '..' after a link logically (`-L`) rather than physically (`-P`, the default):

```bash
realpathx -L src/tools/.. && realpathx -P src/tools/..
```

```powershell
realpathx -L src\tools\..; realpathx -P src\tools\..
```

Print a path relative to another directory:

```bash
realpathx --relative-to=/usr/lib /usr/bin/sort
```

```powershell
realpathx --relative-to=C:\Windows C:\Windows\System32\sort.exe
```

Make names under the project relative, but leave outside ones absolute:

```bash
realpathx --relative-base=. lib/util.go /etc/hosts
```

```powershell
realpathx --relative-base=. lib\util.go $env:WINDIR\System32\drivers\etc\hosts
```

Separate the output with NUL bytes instead of newlines:

```bash
realpathx -z lib/util.go src/app/main.go | od -c
```

```powershell
realpathx -z lib\util.go src\app\main.go | Format-Hex
```

Find the directory that really contains a file:

```bash
dirnamex "$(realpathx scripts/run.sh)"
```

```powershell
dirnamex (realpathx scripts\run.sh)
```

Print several paths relative to a source directory with the `gnux shell`:

```bash
gnux shell -c 'realpathx --relative-to=src lib/util.go bin/tool'
```

```powershell
gnux shell -c 'realpathx --relative-to=src lib/util.go bin/tool'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
realpathx --help                # options, first examples, exit status
gnux help realpathx             # this page: the full manual
gnux help realpathx --OPTION    # one option in depth, with its examples
```

### See also

- [pwdx](../pwdx/index.md)
- [dirnamex](../dirnamex/index.md)
- [basenamex](../basenamex/index.md)
- [lnx](../lnx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
