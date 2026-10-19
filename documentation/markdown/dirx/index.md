## dirx

List directory contents in columns, whether or not the output is a terminal.

Mirrors GNU `dir`. Run it as `dirx`, or as `gnux dirx`.

### Synopsis

```bash
dirx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: dir invocation](https://www.gnu.org/software/coreutils/manual/html_node/dir-invocation.html)
- [Source: src/dir.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/dir.c)

### Description

`dirx` is `lsx -C -b`: it lists each FILE, or the current directory, exactly as `lsx` does and takes every `lsx` option, but arranges the names in columns even when the output is piped or redirected (80 columns wide then, or `-w` COLS), and writes unprintable characters in names as backslash escapes (\n, \t, \ooo) so that every name fits on one line, and never prints '?' for one unless `-q` is given. An option that picks a format (`-l`, -1, `-x`, `-m`, `--format`) or a quoting style (`-N`, `-Q`, `--quoting-style`, or the QUOTING_STYLE variable) replaces those defaults. Diagnostics and exit statuses are `lsx`'s, prefixed with `dirx`.

### Permissions

Listing a directory needs read permission on it; examining its entries (`-l`, `-s`, `-i`, `-F`, `--color`, sorting by time or size) also needs search permission. Nothing is ever modified.

### Use cases

- Getting a compact column listing in a log or a file, where `lsx` prints one name per line.
- Listing a directory the way Windows users expect a dir command to work.
- Seeing names with newlines or tabs in them as escapes rather than broken lines.
- Keeping a script's listing the same whether or not it runs in a terminal.
- Comparing directory contents side by side in a narrow terminal with `-w`.

### Flags

```text
  -x, --across                      list entries by lines instead of by columns
  -a, --all                         do not ignore entries starting with .
  -A, --almost-all                  do not list implied . and ..
  -u, --atime                       with -lt: sort by, and show, access time; with -l: show access time and sort by name; otherwise: sort by access time, newest first
      --author                      with -l, print the author of each file
      --block-size SIZE             with -l, scale sizes by SIZE when printing them; e.g., '--block-size=M'
  -F, --classify                    append indicator (one of */=@|) to entries
      --color WHEN[="always"]       color the output WHEN: 'always' (the default when WHEN is omitted), 'auto', or 'never'
  -C, --columns                     list entries by columns
  -m, --commas                      fill width with a comma separated list of entries
  -c, --ctime                       with -lt: sort by, and show, ctime (time of last change of file status information); with -l: show ctime and sort by name; otherwise: sort by ctime, newest first
  -L, --dereference                 when showing file information for a symbolic link, show information for the file the link references rather than for the link itself
  -H, --dereference-command-line    follow symbolic links listed on the command line
  -d, --directory                   list directories themselves, not their contents
  -D, --dired                       generate output designed for Emacs' dired mode
  -b, --escape                      print C-style escapes for nongraphic characters
      --file-type                   likewise, except do not append '*'
      --format WORD                 across -x, commas -m, horizontal -x, long -l, single-column -1, verbose -l, vertical -C (WORD)
      --full-time                   like -l --time-style=full-iso
      --group-directories-first     group directories before files; any use of --sort=none (-U) disables grouping
      --help                        display this help and exit
      --hide PATTERN                do not list implied entries matching shell PATTERN (overridden by -a or -A)
  -q, --hide-control-chars          print ? instead of nongraphic characters
  -h, --human-readable              with -l and -s, print sizes like 1K 234M 2G etc.
      --hyperlink WHEN[="always"]   hyperlink file names WHEN: 'always' (the default when WHEN is omitted), 'auto', or 'never'
  -I, --ignore PATTERN              do not list implied entries matching shell PATTERN
  -B, --ignore-backups              do not list implied entries ending with ~
      --indicator-style WORD        append indicator with style WORD to entry names: none (default), slash (-p), file-type (--file-type), classify (-F)
  -i, --inode                       print the index number of each file
  -k, --kibibytes                   default to 1024-byte blocks for file system usage; used only with -s and per directory totals
  -N, --literal                     print entry names without quoting
  -l, --long                        use a long listing format
  -o, --long-no-group               like -l, but do not list group information
  -g, --long-no-owner               like -l, but do not list owner
  -G, --no-group                    in a long listing, don't print group names
  -n, --numeric-uid-gid             like -l, but list numeric user and group IDs
  -1, --one-per-line                list one file per line
  -Q, --quote-name                  enclose entry names in double quotes
      --quoting-style WORD          use quoting style WORD for entry names: literal, locale, shell, shell-always, shell-escape, shell-escape-always, c, escape (overrides QUOTING_STYLE environment variable)
  -R, --recursive                   list subdirectories recursively
  -r, --reverse                     reverse order while sorting
      --show-control-chars          show nongraphic characters as-is (the default, unless program is 'ls' and output is a terminal)
      --si                          likewise, but use powers of 1000 not 1024
  -s, --size                        print the allocated size of each file, in blocks
  -p, --slash                       append / indicator to directories
      --sort WORD                   sort by WORD instead of name: none (-U), size (-S), time (-t), version (-v), extension (-X), name, width
  -X, --sort-extension              sort alphabetically by entry extension
  -S, --sort-size                   sort by file size, largest first
  -t, --sort-time                   sort by time, newest first; see --time
  -v, --sort-version                natural sort of (version) numbers within text
  -T, --tabsize COLS                assume tab stops at each COLS instead of 8
      --time WORD                   select which timestamp is shown or sorted on, WORD: access time (-u): atime, access, use; metadata change time (-c): ctime, status; modified time (default): mtime, modification; birth time: birth, creation
      --time-style TIME_STYLE       time/date format with -l, TIME_STYLE: full-iso, long-iso, iso, locale, or +FORMAT
  -U, --unsorted                    do not sort; list entries in directory order
      --version                     output version information and exit
  -w, --width COLS                  set output width to COLS; 0 means no limit
      --zero                        end each output line with NUL, not newline
```

### Examples

List the current directory in columns:

```bash
dirx
```

```powershell
dirx
```

Keep the columns when writing to a file:

```bash
dirx > listing.txt
```

```powershell
dirx > listing.txt
```

Include hidden entries:

```bash
dirx -a
```

```powershell
dirx -a
```

List entries across rather than down:

```bash
dirx -x src
```

```powershell
dirx -x src
```

Fit the columns in 40 characters:

```bash
dirx -w 40 /etc
```

```powershell
dirx -w 40 C:\Windows
```

Mark directories and executables:

```bash
dirx -F
```

```powershell
dirx -F
```

Sort by modification time, newest first:

```bash
dirx -t
```

```powershell
dirx -t
```

Show sizes in blocks next to each name:

```bash
dirx -sh
```

```powershell
dirx -sh
```

Switch to the long format, as `vdirx` does:

```bash
dirx -l
```

```powershell
dirx -l
```

List a directory in the portable `gnux shell`:

```bash
gnux shell -c 'dirx -A dist'
```

```powershell
gnux shell -c 'dirx -A dist'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success |
| 1 | minor trouble: an entry found while listing could not be examined |
| 2 | serious trouble: a command-line operand could not be accessed, a directory could not be read, or a command-line mistake |

### Help in the terminal

```bash
dirx --help                # options, first examples, exit status
gnux help dirx             # this page: the full manual
gnux help dirx --OPTION    # one option in depth, with its examples
```

### See also

- [lsx](../lsx/index.md)
- [vdirx](../vdirx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
