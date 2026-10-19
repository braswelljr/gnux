.. _lsx:

lsx
---

List information about files and directory contents.

Synopsis
~~~~~~~~


lsx lists each FILE, or the current directory when none is given. Operands that are not directories are listed first, then the contents of each directory under a 'dir:' header. Entries whose names start with '.' are hidden unless -a or -A is given. On a terminal the names are arranged in columns sized to the window; when the output is piped or redirected there is one name per line. -l selects the long format (mode, link count, owner, group, size, timestamp and name), where timestamps from the last six months show the time of day and older ones show the year. Entries are sorted by name using plain byte comparison - the C locale's collation, so uppercase sorts before lowercase regardless of the environment's locale - or by time (-t), size (-S), extension (-X) or version (-v), optionally reversed with -r. When several flags of one family are given (-l/-1/-C/-x/-m, the sort keys, -a/-A, -c/-u/--time, …) the last one on the command line wins, as in GNU ls. Names are quoted in GNU's styles: on a terminal, for the shell where a shell would need it ('a b', 'new'$'\n''line'), with an unquoted name moved one column right to line up with quoted ones; otherwise as they are. --quoting-style=WORD picks literal (-N), shell, shell-always, shell-escape, shell-escape-always, c (-Q), c-maybe, escape (-b), locale or clocale, and the QUOTING_STYLE variable does when no option does. On a terminal -q, or a style that leaves control characters as they are, prints '?' for them, unless --show-control-chars. --format=WORD and --indicator-style=WORD are the long forms of the format and indicator options. --zero ends each line with NUL instead of a newline (and implies -1, -N, --show-control-chars and no color unless later options say otherwise). --block-size=SIZE scales sizes in the long format and the -s counts (K, 1M, KB, human-readable, si; LS_BLOCK_SIZE and BLOCK_SIZE when no option gives one), and -k counts -s blocks in KiB. --author adds the author column, which is the owner. --hyperlink[=WHEN] makes each name a terminal hyperlink to its file:// URL. -D (--dired) adds the marks Emacs' dired reads: each long-format line indented two spaces, then //DIRED// and //SUBDIRED// with the byte offsets of each name and directory header; it implies -l, turns --hyperlink off, and cannot be combined with --zero. The exit status is 0 on success, 1 for minor problems such as an entry that cannot be examined, and 2 for serious trouble such as an operand that cannot be accessed.

EXIT STATUS.
  0  success
  1  minor trouble: an entry found while listing could not be examined
  2  serious trouble: a command-line operand could not be accessed, a directory could not be read, or a command-line mistake

HELP. lsx --help shows the options and the first examples; gnux help lsx prints the full manual, and gnux help lsx OPTION explains one option.

SEE ALSO. realpathx, wcx, touchx.

PERMISSIONS. Listing a directory needs read permission on it; examining its entries (anything beyond bare names: -l, -s, -i, -F, --color, sorting by time or size) also needs search permission. Entries that cannot be examined are still listed, with '?' in place of the unknown fields. On Windows, which has no POSIX owner, group, inode or block count, those columns hold placeholders, and with no POSIX permission bits every directory is colored as a plain directory, never as other-writable or sticky.

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

    # Pass every name to another tool safely, even ones with spaces or newlines
    lsx --zero downloads | xargsx -0 sha256sumx

    # Print names you can paste straight back into a shell command
    lsx --quoting-style=shell-escape uploads

    # Show the size of each video in megabytes
    lsx -l --block-size=M videos

    # Make every name clickable in a terminal that supports hyperlinks
    lsx --hyperlink=auto ~/projects

Options
~~~~~~~

::

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

