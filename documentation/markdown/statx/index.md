## statx

Display file or file system status.

Mirrors GNU `stat`. Run it as `statx`, or as `gnux statx`.

### Synopsis

```bash
statx [OPTION]... FILE...
```

### GNU reference

- [GNU coreutils manual: stat invocation](https://www.gnu.org/software/coreutils/manual/html_node/stat-invocation.html)
- [Source: src/stat.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/stat.c)

### Description

`statx` prints everything the system records about each FILE: its type, size, blocks, device, inode, links, permissions, owner and group, and its access, modification, change and birth times. A symbolic `link` is described itself unless `-L` follows it. `-f` describes the file system holding FILE instead: its type, block sizes, and total, free and available blocks and inodes. `-t` prints the same fields on one line, for scripts. `-c` FORMAT prints only what FORMAT asks for, followed by a newline; --`printf` FORMAT does the same but interprets backslash escapes (\n, \t, \x41, \101) and adds no newline; when both are given, the last wins. FORMAT takes GNU's directives: %n (name), %N (quoted name, with its target for a `link`), %s (size), %F (file type), %a and %A (permissions in octal and as rwx), %u %U %g %G (owner and group), %h (links), %i (inode), %b %B %o (blocks, block unit, I/O size), %d %D %Hd %Ld (device), %r %R %t %T %Hr %Lr (device type of a special file), %m (mount point), %w %x %y %z (birth, access, modification and change times) and %W %X %Y %Z (the same as seconds since the epoch, which take a precision: %.3Y), with `printf` flags and widths (%-10s, %04a). With `-f`: %n %i %l %t %T %s %S %b %f %a %c %d. Names are quoted only when a shell would need it. The device is shown as major,minor, as GNU `stat` 9 shows it. Missing data is reported honestly rather than invented: the birth time prints '-' where the system does not record it (Linux without `statx`), and on Windows, which has no numeric owners, %u and %g print '?'. SELinux contexts (%C) are not supported. `--cached` is accepted for compatibility and has no effect.

### Permissions

Needs search permission on the directories leading to each FILE; the FILE itself need not be readable. Owner and group names are read from the system's account databases.

### Use cases

- Finding out exactly when a file was last modified, accessed or created.
- Getting a file's size or permissions in a script without parsing `lsx -l`.
- Checking which file system, and how much free space, a path is on.
- Comparing inodes and devices to tell whether two paths are the same file.
- Auditing ownership and permissions of sensitive files.

### Flags

```text
      --cached string   specify how to use cached attributes (always, never or default); accepted, no effect
  -L, --dereference     follow links
  -f, --file-system     display file system status instead of file status
  -c, --format FORMAT   use the specified FORMAT instead of the default; output a newline after each use of FORMAT
      --help            display this help and exit
      --printf FORMAT   like --format, but interpret backslash escapes, and do not output a mandatory trailing newline
  -t, --terse           print the information in terse form
      --version         output version information and exit
```

### Examples

See everything about a config file before editing it:

```bash
statx /etc/hosts
```

```powershell
statx C:\Windows\System32\drivers\etc\hosts
```

Output:

```text
  File: /etc/hosts
  Size: 221       	Blocks: 8          IO Block: 4096   regular file
Device: 259,2	Inode: 1835014     Links: 1
Access: (0644/-rw-r--r--)  Uid: (    0/    root)   Gid: (    0/    root)
Access: 2026-10-12 08:02:11.120375871 +0000
Modify: 2026-09-30 17:44:02.006417455 +0000
Change: 2026-09-30 17:44:02.006417455 +0000
 Birth: 2026-03-01 10:15:40.773102148 +0000
```

Get a file's size in bytes in a script, without parsing `lsx -l`:

```bash
size=$(statx -c %s backup.tar.gz)
```

```powershell
$size = statx -c %s backup.tar.gz
```

Output:

```text
1073741824
```

Check that an SSH key is private before ssh refuses to use it:

```bash
statx -c '%a %U %n' ~/.ssh/id_ed25519
```

```powershell
statx -c '%a %U %n' $HOME\.ssh\id_ed25519
```

Output:

```text
600 alice /home/alice/.ssh/id_ed25519
```

Find out when a log file last changed, as seconds since the epoch, to alert if it went quiet:

```bash
[ $(( $(datex +%s) - $(statx -c %Y app.log) )) -gt 600 ] && echox 'app.log silent for 10 minutes'
```

```powershell
if (((datex +%s) - (statx -c %Y app.log)) -gt 600) { 'app.log silent for 10 minutes' }
```

List several files with their permissions and owners, aligned:

```bash
statx -c '%-25n %A %U:%G' /etc/passwd /etc/shadow /etc/sudoers
```

```powershell
statx -c '%-25n %A' (Get-ChildItem *.ps1 -Name)
```

Output:

```text
/etc/passwd               -rw-r--r-- root:root
/etc/shadow               -rw-r----- root:shadow
/etc/sudoers              -r--r----- root:root
```

Tell whether two paths are the same file (same device and inode):

```bash
[ "$(statx -L -c %d:%i a)" = "$(statx -L -c %d:%i b)" ] && echox 'same file'
```

```powershell
if ((statx -L -c %d:%i a) -eq (statx -L -c %d:%i b)) { 'same file' }
```

See where a symbolic `link` points:

```bash
statx -c %N /usr/bin/python3
```

```powershell
statx -c %N link
```

Output:

```text
/usr/bin/python3 -> python3.12
```

Check the file system type and free space under a directory:

```bash
statx -f /var/lib/docker
```

```powershell
statx -f C:\
```

Print modification times with millisecond precision, tab-separated:

```bash
statx --printf '%n\t%.3Y\n' *.csv
```

```powershell
statx --printf '%n\t%.3Y\n' (Get-ChildItem *.csv -Name)
```

Output:

```text
jan.csv	1767225600.123
feb.csv	1769904000.456
```

Find a file's mount point:

```bash
statx -c %m ~/Downloads/movie.mkv
```

```powershell
statx -c %m $HOME\Downloads\movie.mkv
```

Output:

```text
/home
```

Record a file's full status on one line, for a log or a diff:

```bash
statx -t release.zip >> inventory.txt
```

```powershell
statx -t release.zip >> inventory.txt
```

Show file details in the portable `gnux shell`, the same on every OS:

```bash
gnux shell -c 'statx -c "%n: %s bytes, modified %y" report.pdf'
```

```powershell
gnux shell -c 'statx -c "%n: %s bytes, modified %y" report.pdf'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
statx --help                # options, first examples, exit status
gnux help statx             # this page: the full manual
gnux help statx --OPTION    # one option in depth, with its examples
```

### See also

- [lsx](../lsx/index.md)
- [dfx](../dfx/index.md)
- [dux](../dux/index.md)
- [chmodx](../chmodx/index.md)
- [touchx](../touchx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
