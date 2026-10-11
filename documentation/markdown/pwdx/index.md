## pwdx

Print the full name of the current working directory.

Mirrors GNU `pwd`. Run it as `pwdx`, or as `gnux pwdx`.

### Synopsis

```bash
pwdx [OPTION]...
```

### Description

`pwdx` prints the absolute path of the current working directory. With `-P` (the default) every symbolic link in the path is resolved; with `-L` it prints the value of the PWD environment variable when that names the current directory, preserving the symlinks the shell walked through. If both are given, the last one wins in GNU `pwd`; `pwdx` prefers `-L`.

### Permissions

Needs search permission on every directory leading to the current one to resolve it physically.

### Use cases

- Capturing the working directory in a script.
- Finding the real location behind a symlinked directory with `-P`.
- Keeping the symlinked path the user navigated through with `-L`, for messages or relative links.
- Detecting whether the current directory is reached through a symlink by comparing `-L` and `-P`.
- Deriving the project name or parent directory with `basenamex` and `dirnamex`.
- Checking where a cd or `envx -C` actually landed in a script.

### Flags

```text
      --help       display this help and exit
  -L, --logical    use PWD from the environment, even if it contains symlinks
  -P, --physical   resolve all symlinks (default)
      --version    output version information and exit
```

### Examples

Print the physical working directory:

```bash
pwdx
```

```powershell
pwdx
```

Keep symlinks as the shell sees them:

```bash
pwdx -L
```

```powershell
pwdx -L
```

Resolve every symlink explicitly:

```bash
cd link && pwdx -P
```

```powershell
Set-Location link; pwdx -P
```

Save the directory in a variable:

```bash
here=$(pwdx); echox "working in $here"
```

```powershell
$here = pwdx; echox "working in $here"
```

Print only the current directory's name:

```bash
basenamex "$(pwdx)"
```

```powershell
basenamex (pwdx)
```

Print the parent directory:

```bash
dirnamex "$(pwdx)"
```

```powershell
dirnamex (pwdx)
```

Tell whether the current directory is reached through a symlink:

```bash
[ "$(pwdx -L)" != "$(pwdx -P)" ] && echox 'inside a symlinked directory'
```

```powershell
if ((pwdx -L) -ne (pwdx -P)) { echox 'inside a symlinked directory' }
```

Print the directory a command would run in:

```bash
envx -C build pwdx
```

```powershell
envx -C build pwdx
```

Show where cd leads inside the `gnux shell`:

```bash
gnux shell -c 'cd docs; pwdx'
```

```powershell
gnux shell -c 'cd docs; pwdx'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
pwdx --help                # options, first examples, exit status
gnux help pwdx             # this page: the full manual
gnux help pwdx --OPTION    # one option in depth, with its examples
```

### See also

- [realpathx](../realpathx/index.md)
- [dirnamex](../dirnamex/index.md)
- [gnux shell](../shell/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
