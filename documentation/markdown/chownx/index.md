## chownx

Change file owner and group.

Mirrors GNU `chown`. Run it as `chownx`, or as `gnux chownx`.

### Synopsis

```bash
chownx [OPTION]... [OWNER][:[GROUP]] FILE...
```

### GNU reference

- [GNU coreutils manual: chown invocation](https://www.gnu.org/software/coreutils/manual/html_node/chown-invocation.html)
- [Source: src/chown.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/chown.c)

### Description

`chownx` changes the owner and/or group of each FILE. OWNER and GROUP are names or numeric IDs: 'OWNER' changes the owner only, 'OWNER:GROUP' both, 'OWNER:' the owner and the group to OWNER's login group, and ':GROUP' the group only; an empty spec changes nothing but still checks that each FILE exists. `--reference`=RFILE takes the owner and group from RFILE instead, and then every operand is a FILE. `--from`=CURRENT_OWNER:CURRENT_GROUP changes only the files currently owned as stated (either part may be omitted). A symbolic `link` given as a FILE is followed, and its referent changed, unless `-h` is given; when both `-h` and `--dereference` are given, `-h` wins. `-R` changes directory trees, following no symbolic `link` by default (`-P`); `-H` follows the ones given on the command line and `-L` every one, and when several are given the most cautious wins (`-P`, then `-H`, then `-L`). `-c` reports each change, `-v` every file, and `-f` suppresses most error messages. Unlike GNU `chown`, which leaves the safeguard off unless `--preserve-root` is given, `chownx` refuses to work recursively on '/' unless `--no-preserve-root` is given; when both are given, `--preserve-root` wins. On Windows, which has no Unix owners, every change fails with 'Operation not supported'.

### Permissions

Changing a file's owner requires root (or CAP_CHOWN). Without it, the owner of a file may change its group, but only to a group they belong to. Search permission is needed on the directories leading to each FILE, and with `-R` read and search permission on every directory in the tree.

### Use cases

- Handing files created as root back to the user `who` will maintain them.
- Giving a service account ownership of its data directory with `-R`.
- Fixing ownership after copying files between machines or containers.
- Changing only the files still owned by a departed user with `--from`.
- Copying the ownership of one file onto others with `--reference`.

### Flags

```text
  -c, --changes                            like verbose but report only when a change is made
      --dereference                        affect the referent of each symbolic link (this is the default), rather than the symbolic link itself
  -L, --dereference-all                    with -R, traverse every symbolic link to a directory encountered
  -H, --dereference-args                   with -R, follow symbolic links to directories given on the command line
      --from CURRENT_OWNER:CURRENT_GROUP   change the ownership of each file only if its current owner and/or group match those specified here; either may be omitted (CURRENT_OWNER:CURRENT_GROUP)
      --help                               display this help and exit
  -h, --no-dereference                     affect symbolic links instead of any referenced file
      --no-preserve-root                   do not treat '/' specially
  -P, --physical                           with -R, do not traverse any symbolic links (default)
      --preserve-root                      fail to operate recursively on '/' (the default)
      --quiet                              same as --silent
  -R, --recursive                          operate on files and directories recursively
      --reference RFILE                    use RFILE's ownership rather than specifying values
  -f, --silent                             suppress most error messages
  -v, --verbose                            output a diagnostic for every file processed
      --version                            output version information and exit
```

### Examples

Give a file to another user:

```bash
sudo chownx alice report.txt
```

```powershell
chownx alice report.txt
```

Change the owner and group:

```bash
sudo chownx alice:staff report.txt
```

```powershell
chownx alice:staff report.txt
```

Change only the group:

```bash
chownx :staff report.txt
```

```powershell
chownx :staff report.txt
```

Use the owner's login group:

```bash
sudo chownx alice: report.txt
```

```powershell
chownx alice: report.txt
```

Use numeric IDs:

```bash
sudo chownx 1000:1000 data.db
```

```powershell
chownx 1000:1000 data.db
```

Change a whole directory tree:

```bash
sudo chownx -R www-data:www-data /var/www/site
```

```powershell
chownx -R www-data:www-data site
```

Change a symbolic `link` itself:

```bash
sudo chownx -h alice link
```

```powershell
chownx -h alice link
```

Change only the files a departed user still owns:

```bash
sudo chownx -R --from=bob alice /srv/shared
```

```powershell
chownx -R --from=bob alice shared
```

Copy the ownership of another file:

```bash
sudo chownx --reference=template.conf new.conf
```

```powershell
chownx --reference=template.conf new.conf
```

Report only the files that changed:

```bash
sudo chownx -c -R alice project/
```

```powershell
chownx -c -R alice project\
```

Change ownership in the portable `gnux shell`:

```bash
sudo gnux shell -c 'chownx -R alice:staff project && lsx -l project'
```

```powershell
gnux shell -c 'chownx -R alice:staff project && lsx -l project'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
chownx --help                # options, first examples, exit status
gnux help chownx             # this page: the full manual
gnux help chownx --OPTION    # one option in depth, with its examples
```

### See also

- [chgrpx](../chgrpx/index.md)
- [chmodx](../chmodx/index.md)
- [idx](../idx/index.md)
- [lsx](../lsx/index.md)
- [statx](../statx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
