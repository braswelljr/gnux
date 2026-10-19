## vdirx

List directory contents in the long format.

Mirrors GNU `vdir`. Run it as `vdirx`, or as `gnux vdirx`.

### Synopsis

```bash
vdirx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: vdir invocation](https://www.gnu.org/software/coreutils/manual/html_node/vdir-invocation.html)
- [Source: src/vdir.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/vdir.c)

### Description

`vdirx` is `lsx -l -b`: it lists each FILE, or the current directory, exactly as `lsx` does and takes every `lsx` option, but in the long format (mode, `link` count, owner, group, size, timestamp and name) by default, and writes unprintable characters in names as backslash escapes (\n, \t, \ooo) so that every entry fits on one line. An option that picks a format (-1, `-C`, `-x`, `-m`) or a quoting style (`-Q`) replaces those defaults. Diagnostics and exit statuses are `lsx`'s, prefixed with `vdirx`.

### Permissions

Listing a directory needs read permission on it, and the long format needs search permission to examine each entry. Nothing is ever modified.

### Use cases

- Seeing permissions, owners, sizes and times without remembering `-l`.
- Checking what a build produced, with sizes in human units (`-h`).
- Finding the most recently changed files with `-t`.
- Recording a directory's state in a log, one entry per line even for odd names.
- Giving Windows users a detailed directory listing under a familiar name.

### Flags

```text
  -x, --across                     list entries by lines instead of by columns
  -a, --all                        do not ignore entries starting with .
  -A, --almost-all                 do not list implied . and ..
  -u, --atime                      with -lt: sort by, and show, access time; with -l: show access time and sort by name; otherwise: sort by access time, newest first
  -F, --classify                   append indicator (one of */=@|) to entries
      --color WHEN[="always"]      color the output WHEN: 'always' (the default when WHEN is omitted), 'auto', or 'never' (default "never")
  -C, --columns                    list entries by columns
  -m, --commas                     fill width with a comma separated list of entries
  -c, --ctime                      with -lt: sort by, and show, ctime (time of last change of file status information); with -l: show ctime and sort by name; otherwise: sort by ctime, newest first
  -L, --dereference                when showing file information for a symbolic link, show information for the file the link references rather than for the link itself
  -H, --dereference-command-line   follow symbolic links listed on the command line
  -d, --directory                  list directories themselves, not their contents
  -b, --escape                     print C-style escapes for nongraphic characters
      --file-type                  likewise, except do not append '*'
      --full-time                  like -l --time-style=full-iso
      --group-directories-first    group directories before files; any use of --sort=none (-U) disables grouping
      --help                       display this help and exit
      --hide PATTERN               do not list implied entries matching shell PATTERN (overridden by -a or -A)
  -q, --hide-control-chars         print ? instead of nongraphic characters
  -h, --human-readable             with -l and -s, print sizes like 1K 234M 2G etc.
  -I, --ignore PATTERN             do not list implied entries matching shell PATTERN
  -B, --ignore-backups             do not list implied entries ending with ~
  -i, --inode                      print the index number of each file
  -l, --long                       use a long listing format
  -o, --long-no-group              like -l, but do not list group information
  -g, --long-no-owner              like -l, but do not list owner
  -G, --no-group                   in a long listing, don't print group names
  -n, --numeric-uid-gid            like -l, but list numeric user and group IDs
  -1, --one-per-line               list one file per line
  -Q, --quote-name                 enclose entry names in double quotes
  -R, --recursive                  list subdirectories recursively
  -r, --reverse                    reverse order while sorting
      --si                         likewise, but use powers of 1000 not 1024
  -s, --size                       print the allocated size of each file, in blocks
  -p, --slash                      append / indicator to directories
      --sort WORD                  sort by WORD instead of name: none (-U), size (-S), time (-t), version (-v), extension (-X), name, width
  -X, --sort-extension             sort alphabetically by entry extension
  -S, --sort-size                  sort by file size, largest first
  -t, --sort-time                  sort by time, newest first; see --time
  -v, --sort-version               natural sort of (version) numbers within text
  -T, --tabsize COLS               assume tab stops at each COLS instead of 8
      --time WORD                  select which timestamp is shown or sorted on, WORD: access time (-u): atime, access, use; metadata change time (-c): ctime, status; modified time (default): mtime, modification; birth time: birth, creation
      --time-style TIME_STYLE      time/date format with -l, TIME_STYLE: full-iso, long-iso, iso, locale, or +FORMAT
  -U, --unsorted                   do not sort; list entries in directory order
      --version                    output version information and exit
  -w, --width COLS                 set output width to COLS; 0 means no limit
```

### Examples

List the current directory in detail:

```bash
vdirx
```

```powershell
vdirx
```

Show sizes in human units:

```bash
vdirx -h dist
```

```powershell
vdirx -h dist
```

Include hidden entries:

```bash
vdirx -a
```

```powershell
vdirx -a
```

Newest files first:

```bash
vdirx -t
```

```powershell
vdirx -t
```

Largest files first:

```bash
vdirx -S
```

```powershell
vdirx -S
```

Show full timestamps:

```bash
vdirx --full-time
```

```powershell
vdirx --full-time
```

Show numeric user and group IDs:

```bash
vdirx -n
```

```powershell
vdirx -n
```

Describe a directory itself, not its contents:

```bash
vdirx -d src
```

```powershell
vdirx -d src
```

Go back to names only:

```bash
vdirx -1
```

```powershell
vdirx -1
```

List a directory in the portable `gnux shell`:

```bash
gnux shell -c 'vdirx -h dist'
```

```powershell
gnux shell -c 'vdirx -h dist'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success |
| 1 | minor trouble: an entry found while listing could not be examined |
| 2 | serious trouble: a command-line operand could not be accessed, a directory could not be read, or a command-line mistake |

### Help in the terminal

```bash
vdirx --help                # options, first examples, exit status
gnux help vdirx             # this page: the full manual
gnux help vdirx --OPTION    # one option in depth, with its examples
```

### See also

- [lsx](../lsx/index.md)
- [dirx](../dirx/index.md)
- [statx](../statx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
