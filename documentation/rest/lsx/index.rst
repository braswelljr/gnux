.. _lsx:

lsx
---

List information about files and directory contents.

Synopsis
~~~~~~~~


lsx lists each FILE, or the current directory when none is given. Operands that are not directories are listed first, then the contents of each directory under a 'dir:' header. Entries whose names start with '.' are hidden unless -a or -A is given. On a terminal the names are arranged in columns sized to the window; when the output is piped or redirected there is one name per line. -l selects the long format (mode, link count, owner, group, size, timestamp and name), where timestamps from the last six months show the time of day and older ones show the year. Entries are sorted by name using plain byte comparison - the C locale's collation, so uppercase sorts before lowercase regardless of the environment's locale - or by time (-t), size (-S), extension (-X) or version (-v), optionally reversed with -r. When several flags of one family are given (-l/-1/-C/-x/-m, the sort keys, -a/-A, -c/-u/--time, …) the last one on the command line wins, as in GNU ls. The exit status is 0 on success, 1 for minor problems such as an entry that cannot be examined, and 2 for serious trouble such as an operand that cannot be accessed.

EXIT STATUS.
  0  success
  1  minor trouble: an entry found while listing could not be examined
  2  serious trouble: a command-line operand could not be accessed, a directory could not be read, or a command-line mistake

HELP. lsx --help shows the options and the first examples; gnux help lsx prints the full manual, and gnux help lsx OPTION explains one option.

SEE ALSO. realpathx, wcx, touchx.

PERMISSIONS. Listing a directory needs read permission on it; examining its entries (anything beyond bare names: -l, -s, -i, -F, --color, sorting by time or size) also needs search permission. Entries that cannot be examined are still listed, with '?' in place of the unknown fields. On Windows, which has no POSIX owner, group, inode or block count, those columns hold placeholders.

USE CASES.
  - Seeing what is in a directory at a glance.
  - Inspecting permissions, ownership, sizes and timestamps with -l.
  - Finding the newest or largest files with -lt or -lS.
  - Walking a tree with -R, or feeding one name per line to a script with -1.
  - Showing hidden files with -A, or telling file types apart with -F or --color.
  - Checking that a build or download produced the expected files before the next step runs.
  - Comparing versioned release directories in natural order with -v.
  - Auditing which files changed recently by sorting on modification, access or change time.
  - Producing stable, script-friendly listings (one per line, no colors) that behave the same on every platform.
  - Spotting oddly named files (spaces, control characters) with -Q, -b or -q.

::

  lsx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # List the current directory
    lsx

    # Long listing including hidden files, with human-readable sizes
    lsx -lah

    # Newest files first, in the long format
    lsx -lt /var/log

    # Largest files first, with sizes in powers of 1000
    lsx -lS --si downloads/

    # Smallest files first (reverse the size sort)
    lsx -lShr downloads/

    # Recursively list a tree, marking directories and executables
    lsx -RF src/

    # One name per line, directories first, skipping object files
    lsx -1 --group-directories-first -I '*.o' build/

    # List a directory itself rather than its contents, with ISO timestamps
    lsx -ld --time-style=long-iso ~/projects

    # Sort release directories by version number, so v1.10 comes after v1.9
    lsx -v releases/

    # Group files by extension, as a comma-separated list
    lsx -X -m assets/

    # Fill rows across rather than down columns, in at most 60 columns
    lsx -x -w 60 /usr/bin

    # Show inode numbers and allocated blocks, with directories marked by a slash
    lsx -isp

    # Long format with numeric user and group IDs, or without the owner or group columns
    lsx -n data/ && lsx -g data/ && lsx -o data/ && lsx -lG data/

    # Sort by last access time (-u) or status change time (-c), showing that time
    lsx -ltu cache/ && lsx -lt --time=ctime cache/

    # Show complete timestamps, or a custom format
    lsx --full-time notes.txt && lsx -l --time-style=+%Y-%m-%d notes.txt

    # Hide temporary and backup files, unless -a or -A is also given
    lsx -B --hide='*.tmp' work/

    # Quote names, or escape control characters, to spot odd file names
    lsx -Q && lsx -b && lsx -q

    # Follow symbolic links: those named on the command line (-H), or all of them (-L)
    lsx -lH current && lsx -lL lib/

    # List in directory order without sorting, or sort with a word instead of a letter
    lsx -U && lsx --sort=size -l

    # Color by file type and mark types (but not executables) when writing to a terminal
    lsx --color=auto --file-type -C -T 4

    # Count the entries in a directory, hidden ones included
    lsx -A | wcx -l

    # Print the name of the most recently modified file
    lsx -t | headx -n 1

    # Fail a script when an expected file is missing (exit status 2)
    lsx dist/app.js > /dev/null || echox 'build output missing'

    # Run the same listing through the portable gnux shell
    gnux shell -c 'lsx -1 *.log | wcx -l'

Options
~~~~~~~

::

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

