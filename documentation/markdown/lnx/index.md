## lnx

Make hard or symbolic links between files.

Mirrors GNU `ln`. Run it as `lnx`, or as `gnux lnx`.

### Synopsis

```bash
lnx [OPTION]... TARGET... [LINK_NAME|DIRECTORY]
```

### GNU reference

- [GNU coreutils manual: ln invocation](https://www.gnu.org/software/coreutils/manual/html_node/ln-invocation.html)
- [Source: src/ln.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/ln.c)

### Description

`lnx` creates a `link` to TARGET. It accepts `ln`'s four forms: TARGET LINK_NAME; TARGET alone (a `link` of the same name in the current directory); TARGET... DIRECTORY; and `-t` DIRECTORY TARGET.... Links are hard links by default and symbolic with `-s`; a symbolic `link` stores TARGET exactly as written unless `-r` rewrites it relative to the `link`'s own location. An existing destination is an error unless `-f` (replace it) or `-i` (ask first) is given, and the replacement is atomic, so the old entry survives a failure. `-n` treats a LINK_NAME that is a symbolic `link` to a directory as the entry to replace rather than a directory to `link` into, and `-T` always treats LINK_NAME as a plain file. `--backup`[=CONTROL] (or `-b`) first renames each existing destination to a backup: CONTROL simple (or never) appends a suffix, numbered (t) appends .~N~ with N one past the highest already there, existing (nil, the default) numbers the backup only when numbered backups already exist, and none (off) makes none; without a CONTROL the VERSION_CONTROL variable chooses. The suffix is ~, or SUFFIX with `-S` (which asks for backups by itself), or the SIMPLE_BACKUP_SUFFIX variable. A backup replaces the destination as `-f` would, and `-v` reports it as 'BACKUP' ~ 'LINK' => 'TARGET'. For hard links, `-L` links to what a symbolic `link` TARGET points at and `-P` (the default) to the `link` itself. GNU `ln` lets the last of conflicting options win; `lnx` cannot see option order, so it prefers the most cautious (`-i` over `-f`) and `-L` over `-P`.

### Permissions

Creating a `link` needs write and search permission on the directory that will hold it. A symbolic `link` needs no access to TARGET, which may not even exist; a hard `link` requires TARGET to exist on the same filesystem and cannot point at a directory. On Windows, creating symbolic links requires Developer Mode or an elevated prompt; a `link`'s target is stored with backslashes, the only separator a Windows `link` follows; and replacing a `link` to a directory (`-sfn`) removes the old `link` before renaming the new one in, since Windows cannot rename over it, so it is not atomic there.

### Use cases

- Pointing a stable name (current, latest) at a versioned file or directory.
- Atomically switching a symlink to a new release with `-sfn`.
- Creating relocatable links inside a tree with `-sr`.
- Giving one file a second name without duplicating its data.
- Exposing scripts on PATH by linking them into ~/bin with `-t`.
- Installing the `gnux` per-tool commands as links to one binary.
- Snapshotting a file with a hard `link` before it is replaced.
- Linking dotfiles from a repository into the home directory.

### Flags

```text
      --backup CONTROL[="$VERSION_CONTROL"]   make a backup of each existing destination file; CONTROL is none (off), simple (never), existing (nil) or numbered (t)
  -b, --backup-default                        like --backup but does not accept an argument
  -f, --force                                 remove existing destination files
      --help                                  display this help and exit
  -i, --interactive                           prompt whether to remove destinations
  -L, --logical                               dereference TARGETs that are symbolic links
  -n, --no-dereference                        treat LINK_NAME as a normal file if it is a symbolic link to a directory
  -T, --no-target-directory                   treat LINK_NAME as a normal file always
  -P, --physical                              make hard links directly to symbolic links (default)
  -r, --relative                              with -s, create links relative to link location
  -S, --suffix SUFFIX                         override the usual backup suffix, '~', with SUFFIX
  -s, --symbolic                              make symbolic links instead of hard links
  -t, --target-directory string               specify the DIRECTORY in which to create the links
  -v, --verbose                               print name of each linked file
      --version                               output version information and exit
```

### Examples

Create a symbolic `link`:

```bash
lnx -s /opt/app/releases/v2 current
```

```powershell
lnx -s C:\app\releases\v2 current
```

Repoint an existing symlink to a directory, in one atomic step:

```bash
lnx -sfn releases/v3 current
```

```powershell
lnx -sfn releases\v3 current
```

Create a hard `link`, reporting what was done:

```bash
lnx -v data.db data.db.snapshot
```

```powershell
lnx -v data.db data.db.snapshot
```

Link several files into a directory with relative symlinks:

```bash
lnx -sr -t ~/bin tools/build tools/deploy
```

```powershell
lnx -sr -t $HOME\bin tools\build tools\deploy
```

Create a `link` with the same name in the current directory:

```bash
lnx -s ../shared/config.yml
```

```powershell
lnx -s ..\shared\config.yml
```

Link several targets into an existing directory:

```bash
lnx -sv ../dotfiles/.vimrc ../dotfiles/.gitconfig home/
```

```powershell
lnx -sv ..\dotfiles\.vimrc ..\dotfiles\.gitconfig home
```

Ask before replacing an existing `link`:

```bash
lnx -si settings.prod.json settings.json
```

```powershell
lnx -si settings.prod.json settings.json
```

Replace an existing file with a `link`:

```bash
lnx -sf app-v2.conf app.conf
```

```powershell
lnx -sf app-v2.conf app.conf
```

Treat the `link` name as a plain file, never as a directory to `link` into:

```bash
lnx -sT releases/v3 current-release
```

```powershell
lnx -sT releases\v3 current-release
```

Hard-`link` the file a symlink points to, instead of the symlink itself:

```bash
lnx -L current.log saved.log
```

```powershell
lnx -L current.log saved.log
```

Hard-`link` the symlink itself (the default, spelled out):

```bash
lnx -P current.log current-link.log
```

```powershell
lnx -P current.log current-link.log
```

Create the per-tool commands as links to the `gnux` binary:

```bash
for t in $(gnux --list); do lnx -sf gnux "$t"; done
```

```powershell
gnux --list | ForEach-Object { lnx -sf gnux.exe "$_.exe" }
```

Report whether linking succeeded:

```bash
lnx -s target.txt link.txt && echox linked || echox "link exists already"
```

```powershell
lnx -s target.txt link.txt; if ($LASTEXITCODE -eq 0) { echox linked } else { echox "link exists already" }
```

Create a release directory and point current at it with the `gnux shell`:

```bash
gnux shell -c 'mkdirx -p releases/v4 && lnx -sfnv releases/v4 current'
```

```powershell
gnux shell -c 'mkdirx -p releases/v4 && lnx -sfnv releases/v4 current'
```

Repoint a symbolic `link`, keeping the old one as current~ in case you need to roll back:

```bash
lnx -sbv releases/v5 current
```

```powershell
lnx -sbv releases\v5 current
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
lnx --help                # options, first examples, exit status
gnux help lnx             # this page: the full manual
gnux help lnx --OPTION    # one option in depth, with its examples
```

### See also

- [cpx](../cpx/index.md)
- [realpathx](../realpathx/index.md)
- [lsx](../lsx/index.md)
- [rmx](../rmx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
