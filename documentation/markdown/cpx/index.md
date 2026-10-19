## cpx

Copy files and directories.

Mirrors GNU `cp`. Run it as `cpx`, or as `gnux cpx`.

### Synopsis

```bash
cpx [OPTION]... SOURCE... DIRECTORY
```

### GNU reference

- [GNU coreutils manual: cp invocation](https://www.gnu.org/software/coreutils/manual/html_node/cp-invocation.html)
- [Source: src/cp.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/cp.c)

### Description

`cpx` copies SOURCE to DEST, or several SOURCEs into DIRECTORY (also selectable with `-t`; `-T` forces the two-operand form). Directories are copied only with `-r`, and a directory is never copied into itself. A new file receives the source's permission bits filtered by the umask; `-p` or `--preserve`[=LIST] carries over mode, ownership, timestamps and hard links, and `-a` is shorthand for `-dR --preserve`=all. Symbolic links are followed unless copying recursively; `-P`/`-d` keep them as links, `-L` follows them everywhere and `-H` only on the command line. `-l` and `-s` create hard or symbolic links instead of copying data. Existing destinations are overwritten unless `-n` (never), `-u` (only when the source is newer) or `-i` (ask first) says otherwise; `-f` removes a destination that cannot be opened and tries again, and `--remove-destination` removes every existing destination before writing it, so a destination symbolic `link` is replaced rather than written through and a file that other names share is not changed under them. GNU `cp` lets the last of conflicting options win; `cpx` cannot see option order, so it prefers the most cautious instead: `-n` overrides `-i`, and an explicit `-P`/`-d` overrides `-H`, which overrides `-L`.

### Permissions

Copying needs read permission on each source and write and search permission on the destination directory. Without `-p` the copy is owned by the invoking user and its mode is the source's mode minus the umask. Preserving ownership requires privilege; an ordinary user silently keeps their own ownership, as with GNU `cp`. On Windows only the read-only attribute and modification time are carried over.

### Use cases

- Duplicating a file before editing it, so the original can be restored.
- Copying a directory tree with `-r`, or a faithful archive of it with `-a`.
- Refreshing a mirror with `-u` so only newer files are rewritten.
- Filling in missing defaults without touching customized files (`--update`=none).
- Guarding existing files in scripts with `-n`, or interactively with `-i`.
- Building a tree of hard links (`-al`) or symlinks (`-s`) instead of duplicating data.
- Recreating selected files under a backup directory with their paths intact (`--parents`).
- Replacing a read-only destination in a deployment step with `-f`.
- Deploying over a file a running program has open, or one hard-linked elsewhere, with `--remove-destination`.
- Collecting build artifacts into a directory named up front with `-t`.

### Flags

```text
  -a, --archive                                         same as -dR --preserve=all
  -L, --dereference                                     always follow symbolic links in SOURCE
  -H, --dereference-args                                follow command-line symbolic links in SOURCE
  -f, --force                                           if an existing destination file cannot be opened, remove it and try again
      --help                                            display this help and exit
  -i, --interactive                                     prompt before overwrite
  -l, --link                                            hard link files instead of copying
  -n, --no-clobber                                      do not overwrite an existing file
  -P, --no-dereference                                  never follow symbolic links in SOURCE
  -d, --no-dereference-links                            same as --no-dereference --preserve=links
      --no-preserve string                              don't preserve the listed attributes
  -T, --no-target-directory                             treat DEST as a normal file
  -x, --one-file-system                                 stay on this file system
      --parents                                         use the full source file name under DIRECTORY
      --preserve string[="mode,ownership,timestamps"]   preserve the listed attributes: mode, ownership, timestamps, links, all
  -p, --preserve-default                                same as --preserve=mode,ownership,timestamps
  -r, --recursive                                       copy directories recursively
      --remove-destination                              remove each existing destination file before attempting to open it (contrast with --force)
      --strip-trailing-slashes                          remove any trailing slashes from each SOURCE argument
  -s, --symbolic-link                                   make symbolic links instead of copying
  -t, --target-directory string                         copy all SOURCE arguments into DIRECTORY
  -u, --update string[="older"]                         control which existing files are updated: all, none, or older (the default for -u)
  -v, --verbose                                         explain what is being done
      --version                                         output version information and exit
```

### Examples

Copy a file to a new name:

```bash
cpx notes.txt notes.bak
```

```powershell
cpx notes.txt notes.bak
```

Copy several files into a directory, reporting each one:

```bash
cpx -v a.txt b.txt backup/
```

```powershell
cpx -v a.txt b.txt backup
```

Archive a directory tree, preserving modes, times and links:

```bash
cpx -a project/ /mnt/backup/project
```

```powershell
cpx -a project D:\backup\project
```

Update a mirror, copying only files that are newer:

```bash
cpx -ru src/. mirror/
```

```powershell
cpx -ru src\. mirror
```

Copy into a directory named first, without overwriting anything:

```bash
cpx -n -t ~/Documents report.pdf slides.pdf
```

```powershell
cpx -n -t $HOME\Documents report.pdf slides.pdf
```

Ask before overwriting an existing copy:

```bash
cpx -i config.yml config.yml.orig
```

```powershell
cpx -i config.yml config.yml.orig
```

Keep the mode and timestamps of a script while copying it:

```bash
cpx -p deploy.sh bin/
```

```powershell
cpx -p deploy.sh bin
```

Preserve only the timestamps, or archive without carrying over ownership:

```bash
cpx --preserve=timestamps photo.jpg archive/ && cpx -a --no-preserve=ownership project/ copy/
```

```powershell
cpx --preserve=timestamps photo.jpg archive; cpx -a --no-preserve=ownership project copy
```

Recreate the source paths under a backup directory:

```bash
cpx --parents src/app/main.go src/lib/util.go backup/
```

```powershell
cpx --parents src/app/main.go src/lib/util.go backup
```

Make a new release tree of hard links that share the old one's data:

```bash
cpx -al releases/v1 releases/v2
```

```powershell
cpx -al releases\v1 releases\v2
```

Create a symbolic `link` instead of a copy (use an absolute source):

```bash
cpx -s "$PWD/config.yml" config-link.yml
```

```powershell
cpx -s "$PWD\config.yml" config-link.yml
```

Copy a symbolic `link` as a `link`, rather than the file it points to:

```bash
cpx -P current.yml saved-link.yml
```

```powershell
cpx -P current.yml saved-link.yml
```

Copy a tree, replacing every symbolic `link` with the file it points to:

```bash
cpx -rL vendor/ vendor-flat/
```

```powershell
cpx -rL vendor vendor-flat
```

Copy through a symlinked directory named on the command line only:

```bash
cpx -rH latest snapshot
```

```powershell
cpx -rH latest snapshot
```

Copy a template's contents into an existing directory, not inside it:

```bash
cpx -rT template/ newproject/
```

```powershell
cpx -rT template newproject
```

Add missing default files without overwriting customized ones:

```bash
cpx -r --update=none defaults/. config/
```

```powershell
cpx -r --update=none defaults\. config
```

Replace a read-only destination file:

```bash
cpx -f new.conf readonly.conf
```

```powershell
cpx -f new.conf readonly.conf
```

Back up a tree without crossing into other mounted file systems:

```bash
cpx -ax --strip-trailing-slashes data/ /mnt/backup/data
```

```powershell
cpx -ax --strip-trailing-slashes data\ D:\backup\data
```

Make a backup and continue only if the copy succeeded:

```bash
cpx config.yml config.yml.bak && echox "backup saved"
```

```powershell
cpx config.yml config.yml.bak; if ($LASTEXITCODE -eq 0) { echox "backup saved" }
```

Copy every .conf file into a fresh directory with the `gnux shell`:

```bash
gnux shell -c 'mkdirx -p confbak && cpx -v *.conf confbak/'
```

```powershell
gnux shell -c 'mkdirx -p confbak && cpx -v *.conf confbak/'
```

Replace a symbolic `link` with a real copy instead of writing through it:

```bash
cpx --remove-destination config/base.conf current.conf
```

```powershell
cpx --remove-destination config\base.conf current.conf
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
cpx --help                # options, first examples, exit status
gnux help cpx             # this page: the full manual
gnux help cpx --OPTION    # one option in depth, with its examples
```

### See also

- [mvx](../mvx/index.md)
- [lnx](../lnx/index.md)
- [rmx](../rmx/index.md)
- [mkdirx](../mkdirx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
