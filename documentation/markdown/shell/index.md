## shell

Run a portable shell whose commands are the `gnux` tools.

### Synopsis

```bash
gnux shell [OPTION]... [SCRIPT]
```

### Description

`gnux shell` is a small command interpreter whose commands are the `gnux` tools, run inside the `gnux` process. Because nothing in it depends on the system's own shell or utilities, the same script behaves identically on Linux, macOS and Windows. With no SCRIPT it reads commands from standard input, prompting when that is a terminal; with `-c` it runs the given string.

The language is a deliberately small subset of POSIX sh: pipelines (a | b), the &&, || and ; connectors, redirection (<, >, >>, and 2> for standard error; /dev/null discards on every platform), NAME=value variables with $NAME, ${NAME}, $? and $PWD expansion, single and double quotes, globbing (*, ?, [...]), ~ for the home directory and # comments. The builtins are cd, exit and help. Tools answer to both names, so `rm` and `rmx` are the same command. There are no loops, functions or conditionals beyond && and ||, and a variable's value is never re-`split`, so paths with spaces need no quoting once stored. External programs are not looked up; run one through the env tool (`envx` PROGRAM ARG...).

The exit status is that of the last command run, 2 for a syntax error, and 127 for an unknown command. With `-e` the script stops at the first failing command.

### Permissions

The shell itself needs only read access to SCRIPT. Each command it runs needs whatever its tool needs; redirection creates files with mode 0644 (before the umask).

### Use cases

- Build, clean and packaging scripts that must run unchanged on Windows, macOS and Linux.
- A consistent interactive shell on machines without the GNU tools installed.
- Embedding small file-management scripts in a Go application without spawning processes.

### Flags

```text
  -c, --command STRING   run the commands in STRING instead of reading a script
  -e, --errexit          exit at the first command that fails
      --help             display this help and exit
      --version          output version information and exit
```

### Examples

Start an interactive session:

```bash
gnux shell
```

```powershell
gnux shell
```

Run a one-liner:

```bash
gnux shell -c 'mkdirx -p dist && cpx -r assets dist/ && lsx -l dist'
```

```powershell
gnux shell -c 'mkdirx -p dist && cpx -r assets dist/ && lsx -l dist'
```

Run a script file, stopping at the first failure:

```bash
gnux shell -e build.gnux
```

```powershell
gnux shell -e build.gnux
```

Pipe a script in:

```bash
echo 'catx *.log | wcx -l' | gnux shell
```

```powershell
'catx *.log | wcx -l' | gnux shell
```

### Exit status

| Status | Meaning |
|--------|---------|
| N | the exit status of the last command run |
| 1 | a redirection could not be opened (that command is not run) |
| 2 | a syntax error, exit with a non-numeric argument, or a command-line mistake |
| 127 | an unknown command, or a SCRIPT that cannot be opened |

### Help in the terminal

```bash
gnux help shell             # this page: the full manual
gnux help shell --OPTION    # one option in depth, with its examples
```

### See also

- [envx](../envx/index.md)
- [truex](../truex/index.md)
- [falsex](../falsex/index.md)
- [echox](../echox/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
