## chgrpx

Change group ownership.

Mirrors GNU `chgrp`. Run it as `chgrpx`, or as `gnux chgrpx`.

### Synopsis

```bash
chgrpx [OPTION]... GROUP FILE...
```

### GNU reference

- [GNU coreutils manual: chgrp invocation](https://www.gnu.org/software/coreutils/manual/html_node/chgrp-invocation.html)
- [Source: src/chown.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/chown.c)

### Description

`chgrpx` changes the group of each FILE to GROUP, a group name or numeric ID. An empty GROUP changes nothing but still checks that each FILE exists. `--reference`=RFILE takes the group from RFILE instead, and then every operand is a FILE. `--from`=CURRENT_OWNER:CURRENT_GROUP changes only the files whose current owner and group match; either part may be omitted, as in `--from`=alice or `--from`=:staff. A symbolic `link` given as a FILE is followed, and its referent changed, unless `-h` is given; when both `-h` and `--dereference` are given, `-h` wins. `-R` changes directory trees, following no symbolic `link` by default (`-P`); `-H` follows the ones given on the command line and `-L` every one, and when several are given the most cautious wins (`-P`, then `-H`, then `-L`). `-c` reports each change, `-v` every file, and `-f` suppresses most error messages. Unlike GNU `chgrp`, which leaves the safeguard off unless `--preserve-root` is given, `chgrpx` refuses to work recursively on '/' unless `--no-preserve-root` is given; when both are given, `--preserve-root` wins. On Windows, which has no Unix groups, every change fails with 'Operation not supported'.

### Permissions

The owner of a file may change its group to any group they belong to; any other change requires root (or CAP_CHOWN). Search permission is needed on the directories leading to each FILE, and with `-R` read and search permission on every directory in the tree.

### Use cases

- Sharing a project directory with a team's group.
- Giving a web server's group read access to a site with `-R`.
- Fixing group ownership after extracting an archive.
- Copying the group of one file onto others with `--reference`.
- Moving only the files still in an old group to a new one with `--from`.
- Checking which files would change with `-c` on a `test` run.

### Flags

```text
  -c, --changes                            like verbose but report only when a change is made
      --dereference                        affect the referent of each symbolic link (this is the default), rather than the symbolic link itself
  -L, --dereference-all                    with -R, traverse every symbolic link to a directory encountered
  -H, --dereference-args                   with -R, follow symbolic links to directories given on the command line
      --from CURRENT_OWNER:CURRENT_GROUP   change the group of each file only if its current owner and/or group match those specified here; either may be omitted (CURRENT_OWNER:CURRENT_GROUP)
      --help                               display this help and exit
  -h, --no-dereference                     affect symbolic links instead of any referenced file
      --no-preserve-root                   do not treat '/' specially
  -P, --physical                           with -R, do not traverse any symbolic links (default)
      --preserve-root                      fail to operate recursively on '/' (the default)
      --quiet                              same as --silent
  -R, --recursive                          operate on files and directories recursively
      --reference RFILE                    use RFILE's group rather than specifying a GROUP value
  -f, --silent                             suppress most error messages
  -v, --verbose                            output a diagnostic for every file processed
      --version                            output version information and exit
```

### Examples

Change the group of a file:

```bash
chgrpx staff report.txt
```

```powershell
chgrpx staff report.txt
```

Use a numeric group ID:

```bash
chgrpx 1001 report.txt
```

```powershell
chgrpx 1001 report.txt
```

Change a whole directory tree:

```bash
chgrpx -R developers project/
```

```powershell
chgrpx -R developers project\
```

Follow symbolic links met while recursing:

```bash
chgrpx -R -L developers project/
```

```powershell
chgrpx -R -L developers project\
```

Change a symbolic `link` itself:

```bash
chgrpx -h staff link
```

```powershell
chgrpx -h staff link
```

Copy the group of another file:

```bash
chgrpx --reference=template.conf new.conf
```

```powershell
chgrpx --reference=template.conf new.conf
```

Move only the files still in the old team's group:

```bash
chgrpx -R --from=:oldteam newteam shared/
```

```powershell
chgrpx -R --from=:oldteam newteam shared\
```

Report only the files that changed:

```bash
chgrpx -c -R staff shared/
```

```powershell
chgrpx -c -R staff shared\
```

Report every file processed:

```bash
chgrpx -v staff a.txt b.txt
```

```powershell
chgrpx -v staff a.txt b.txt
```

Suppress most error messages:

```bash
chgrpx -f staff maybe-missing.txt
```

```powershell
chgrpx -f staff maybe-missing.txt
```

Give a web server's group the site files:

```bash
sudo chgrpx -R www-data /var/www/site
```

```powershell
chgrpx -R www-data site
```

Change groups in the portable `gnux shell`:

```bash
gnux shell -c 'chgrpx -R staff shared && lsx -l shared'
```

```powershell
gnux shell -c 'chgrpx -R staff shared && lsx -l shared'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
chgrpx --help                # options, first examples, exit status
gnux help chgrpx             # this page: the full manual
gnux help chgrpx --OPTION    # one option in depth, with its examples
```

### See also

- [chownx](../chownx/index.md)
- [chmodx](../chmodx/index.md)
- [idx](../idx/index.md)
- [lsx](../lsx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
