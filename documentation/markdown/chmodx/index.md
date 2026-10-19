## chmodx

Change file mode bits.

Mirrors GNU `chmod`. Run it as `chmodx`, or as `gnux chmodx`.

### Synopsis

```bash
chmodx [OPTION]... MODE[,MODE]... FILE...
```

### GNU reference

- [GNU coreutils manual: chmod invocation](https://www.gnu.org/software/coreutils/manual/html_node/chmod-invocation.html)
- [Source: src/chmod.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/chmod.c)

### Description

`chmodx` changes the permission bits of each FILE to MODE, which is either an octal number (755, 0644, 4755) or a comma-separated list of symbolic changes. Each change names `who` (u, g, o or a for user, group, others or all), an operator (+ adds, - removes, = sets exactly) and permissions: r, w, x, X (execute only on directories and on files already executable by someone), s (setuid/setgid) and t (sticky), or u, g or o to copy another class's bits. A symbolic change without a `who` honors the umask. `--reference`=RFILE copies RFILE's mode instead, and then every operand is a FILE. `-R` changes directory trees; `-H` (the default) follows symbolic links given on the command line, `-L` follows every symbolic `link` and `-P` none, and when several are given the most cautious wins (`-P`, then `-H`, then `-L`). A symbolic `link` given as a FILE is followed unless `-h` is given, and one met while recursing is not, unless `--dereference` is given (which changes its referent without descending into it) or `-L`; when both `-h` and `--dereference` are given, `-h` wins. A `link` that is not followed has its own mode changed where the system gives links one (macOS, the BSDs); on Linux it is reported as unchanged. `-c` reports each change, `-v` every file, and `-f` suppresses most error messages. A mode may also be written as an option, as in '`chmodx -w` FILE'; several such options combine as if joined with commas, and then, as in GNU, a file left with bits the mode would have cleared but for the umask is reported ('new permissions are ..., not ...') and counts as a failure. Unlike GNU `chmod`, which leaves the safeguard off unless `--preserve-root` is given, `chmodx` refuses to work recursively on '/' unless `--no-preserve-root` is given; when both are given, `--preserve-root` wins. On Windows only the owner's write bit maps onto the file system (the read-only attribute), so the other bits are accepted but have no effect.

### Permissions

Changing a file's mode requires owning it, or running as root (or with CAP_FOWNER). Search permission is needed on the directories leading to each FILE, and with `-R` read and search permission on every directory in the tree.

### Use cases

- Making a script executable with +x.
- Locking down a private key or config file with 600.
- Giving a group write access to a shared directory tree with `-R` g+w.
- Making directories, but not files, searchable with `-R` a+X.
- Copying the permissions of one file onto others with `--reference`.

### Flags

```text
  -c, --changes            like verbose but report only when a change is made
      --dereference        affect the referent of each symbolic link, rather than the symbolic link itself
  -L, --dereference-all    with -R, follow every symbolic link to a directory
  -H, --dereference-args   with -R, follow symbolic links given on the command line (the default)
      --help               display this help and exit
  -h, --no-dereference     affect each symbolic link, rather than the referent
      --no-preserve-root   do not treat '/' specially
  -P, --physical           with -R, do not follow any symbolic links
      --preserve-root      fail to operate recursively on '/' (the default)
      --quiet              same as --silent
  -R, --recursive          change files and directories recursively
      --reference RFILE    use RFILE's mode instead of specifying MODE values
  -f, --silent             suppress most error messages
  -v, --verbose            output a diagnostic for every file processed
      --version            output version information and exit
```

### Examples

You wrote deploy.sh and the shell says 'Permission denied': make it executable:

```bash
chmodx +x deploy.sh
```

```powershell
chmodx +x deploy.sh
```

ssh refuses your key because it is readable by others: make it private:

```bash
chmodx 600 ~/.ssh/id_ed25519
```

```powershell
chmodx 600 $HOME\.ssh\id_ed25519
```

Stop other users changing a shared file:

```bash
chmodx go-w shared.txt
```

```powershell
chmodx go-w shared.txt
```

Give an app full access for its owner, read for its group, none for others:

```bash
chmodx u=rwx,g=rx,o= app
```

```powershell
chmodx u=rwx,g=rx,o= app
```

Let your team edit everything in a project directory:

```bash
chmodx -R g+w project/
```

```powershell
chmodx -R g+w project\
```

Make a website folder browsable without making every file executable:

```bash
chmodx -R a+X public/
```

```powershell
chmodx -R a+X public\
```

Normalize text file permissions and see which ones were different:

```bash
chmodx -c 644 *.txt
```

```powershell
chmodx -c 644 (Get-ChildItem *.txt -Name)
```

Give a new script the same permissions as an existing one:

```bash
chmodx --reference=template.sh new.sh
```

```powershell
chmodx --reference=template.sh new.sh
```

Make a file read-only the way old scripts write it:

```bash
chmodx -w notes.txt
```

```powershell
chmodx -w notes.txt
```

Make a shared drop folder where people cannot delete each other's files:

```bash
chmodx 1777 /srv/drop
```

```powershell
chmodx 1777 C:\srv\drop
```

Fix permissions on your docs and see every file processed:

```bash
chmodx -v -R u+rw docs/
```

```powershell
chmodx -v -R u+rw docs\
```

Make scripts executable in the portable `gnux shell`, the same on every OS:

```bash
gnux shell -c 'chmodx +x bin/* && lsx -l bin'
```

```powershell
gnux shell -c 'chmodx +x bin/* && lsx -l bin'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
chmodx --help                # options, first examples, exit status
gnux help chmodx             # this page: the full manual
gnux help chmodx --OPTION    # one option in depth, with its examples
```

### See also

- [chownx](../chownx/index.md)
- [chgrpx](../chgrpx/index.md)
- [lsx](../lsx/index.md)
- [statx](../statx/index.md)
- [installx](../installx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
