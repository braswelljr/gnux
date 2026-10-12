## installx

Copy files into place and set their mode, owner and group.

Mirrors GNU `install`. Run it as `installx`, or as `gnux installx`.

### Synopsis

```bash
installx [OPTION]... SOURCE... DEST
```

### GNU reference

- [GNU coreutils manual: install invocation](https://www.gnu.org/software/coreutils/manual/html_node/install-invocation.html)
- [Source: src/install.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/install.c)

### Description

`installx` copies SOURCE to DEST, or several SOURCE files into the existing directory DEST (or into `-t` DIRECTORY), and then sets each copy's permission mode (755 unless `-m` gives another, octal or symbolic) and, with `-o` and `-g`, its owner and group. An existing DEST is removed and replaced, not rewritten, so a running program keeps its copy. With `-d` every operand is a directory to create, with its missing parents; the mode applies to the named directories, and parents get 755 minus the umask. `-D` creates the leading directories of DEST (or the `-t` directory) first. `-T` treats DEST as a file even if it is a directory. `-C` leaves a destination alone when it already has the same content, mode, owner and group. `-p` gives each copy its source's modification time (the access time is set to the same value). `-v` reports each file and directory. `-s` is accepted, but stripping symbol tables needs a strip program, so it only warns that files are installed unstripped. `-c` is accepted and ignored, as in GNU. `-D` also answers to the `gnux` long name `--create-leading`. A symbolic `-m` starts from no permissions (u=rwx,go=rx is 755), as GNU `install` does.

### Permissions

Needs read permission on each SOURCE and write and search permission on the destination directory. Setting the owner with `-o` needs root, and setting the group with `-g` needs ownership of the file and membership of the group. Off Unix, `-o` and `-g` accept numeric IDs only and fail with 'Operation not supported', only the read-only attribute of a mode is representable, and `-C` therefore always reinstalls.

### Use cases

- Installing a built binary into a bin directory with the right mode in one step.
- Writing a Makefile `install` target that works the same on Linux, macOS and Windows.
- Creating a directory tree with an exact mode with `-d`.
- Deploying configuration files readable by their owner only with `-m` 600.
- Placing a file into a path whose directories may not exist yet with `-D`.
- Avoiding needless rewrites (and changed timestamps) of unchanged files with `-C`.
- Setting ownership of system files during packaging with `-o` and `-g`.
- Logging what a deployment copied with `-v`.

### Flags

```text
  -C, --compare                      compare content of source and destination files, and if no change to content, ownership, and permissions, do not modify the destination at all
  -c, --copy                         ignored; for compatibility with old Unix versions of install
  -D, --create-leading               create all leading components of DEST except the last, or all components of --target-directory, then copy SOURCE to DEST
  -d, --directory                    treat all arguments as directory names; create all components of the specified directories
  -g, --group GROUP                  set group ownership, instead of process' current group, to GROUP
      --help                         display this help and exit
  -m, --mode MODE                    set permission mode (as in chmod) to MODE, instead of rwxr-xr-x
  -T, --no-target-directory          treat DEST as a normal file
  -o, --owner OWNER                  set ownership (super-user only) to OWNER
  -p, --preserve-timestamps          apply access/modification times of SOURCE files to corresponding destination files
  -s, --strip                        strip symbol tables (not supported: warns and installs unstripped)
  -t, --target-directory DIRECTORY   copy all SOURCE arguments into DIRECTORY
  -v, --verbose                      print the name of each created file or directory
      --version                      output version information and exit
```

### Examples

Install a binary with mode 755:

```bash
installx bin/gnux /usr/local/bin/gnux
```

```powershell
installx bin\gnux.exe C:\tools\gnux.exe
```

Install several files into a directory:

```bash
installx a.sh b.sh ~/bin
```

```powershell
installx a.ps1 b.ps1 $HOME\bin
```

Install a configuration file readable by its owner only:

```bash
installx -m 600 app.conf ~/.config/app/app.conf
```

```powershell
installx -m 600 app.conf $HOME\.config\app\app.conf
```

Use a symbolic mode:

```bash
installx -m u=rw,go=r README.md /opt/app/README.md
```

```powershell
installx -m 'u=rw,go=r' README.md C:\app\README.md
```

Create directories with a given mode:

```bash
installx -d -m 750 /srv/app/logs /srv/app/data
```

```powershell
installx -d -m 750 C:\srv\logs C:\srv\data
```

Create the leading directories of the destination:

```bash
installx -D app.conf out/etc/app/app.conf
```

```powershell
installx -D app.conf out\etc\app\app.conf
```

Install into a target directory, creating it first:

```bash
installx -D -t out/bin build/*
```

```powershell
installx -D -t out\bin (Get-ChildItem build -Name)
```

Report what is installed:

```bash
installx -v -t ~/bin tool1 tool2
```

```powershell
installx -v -t $HOME\bin tool1 tool2
```

Leave unchanged files untouched:

```bash
installx -C -m 644 docs/*.md /usr/share/doc/app/
```

```powershell
installx -C -m 644 (Get-ChildItem docs\*.md -Name) C:\app\doc
```

Keep the source's modification time:

```bash
installx -p data.db backup/data.db
```

```powershell
installx -p data.db backup\data.db
```

Set the owner and group (as root):

```bash
sudo installx -o root -g wheel -m 755 gnux /usr/local/bin/
```

```powershell
installx -m 755 gnux.exe C:\tools\
```

Install to a file name even if a directory of that name exists (an error then):

```bash
installx -T build/app dist/app
```

```powershell
installx -T build\app.exe dist\app.exe
```

An `install` step in the portable `gnux shell`:

```bash
gnux shell -e -c 'installx -D -m 644 site/index.html public/index.html'
```

```powershell
gnux shell -e -c 'installx -D -m 644 site/index.html public/index.html'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
installx --help                # options, first examples, exit status
gnux help installx             # this page: the full manual
gnux help installx --OPTION    # one option in depth, with its examples
```

### See also

- [cpx](../cpx/index.md)
- [mkdirx](../mkdirx/index.md)
- [touchx](../touchx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
