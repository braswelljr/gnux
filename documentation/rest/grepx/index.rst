.. _grepx:

grepx
-----

Print the lines that match patterns.

Synopsis
~~~~~~~~


grepx searches each FILE for PATTERNS and prints every line that matches one. PATTERNS is the first operand, unless -e or -f supplied patterns, and may hold several patterns separated by newlines; a line is selected when any of them matches. With no FILE, or when FILE is -, standard input is searched (shown as '(standard input)', or as the --label); with -r and no FILE, the working directory.

Pattern syntax. Patterns are POSIX basic regular expressions by default (-G), where \( \) \{ \} \| \+ \? are the operators; -E takes extended regular expressions, where ( ) { } | + ? are, and a { that does not start an interval is literal; -F takes fixed strings. Both regular syntaxes accept bracket expressions with POSIX classes ([[:alpha:]], [[:digit:]], ...), and the GNU escapes \w \W \s \S \b \B \< \> \` \'. -i ignores case, -w accepts only matches that are whole words, -x only matches of the whole line, and -v selects the lines that do not match. An empty pattern matches every line; an empty -f file matches none.

Output. Each selected line is printed, prefixed on request with the file name (-H, the default when several files or a directory are searched; -h turns it off), the line number (-n) and the byte offset (-b), each followed by ':'. -o prints only the matched parts, one to a line. -c prints a count of selected lines per file, -l the names of the files with a selected line, -L the names of those without, and -q nothing at all, stopping at the first selected line. -m NUM stops reading a file after NUM selected lines. -A, -B and -C (or -NUM) add lines of context, which are prefixed with '-' in place of ':' and set apart in groups by a line holding '--' (--group-separator, --no-group-separator). -Z ends each file name with a NUL and -z reads and writes lines ending with a NUL. --color highlights matches, file names, line numbers, offsets and separators with GNU's default colors, which GREP_COLORS overrides; --color=auto colors only when standard output is a terminal and TERM is set (which is not required on Windows) and is not 'dumb'.

File selection. -r searches directories recursively, following only the symbolic links named as operands; -R follows them all. Files are visited in name order. --include and --exclude select files by wildcard on their base name (for a file named on the command line, on the name or any trailing part of it after a slash); when several match, the last one given decides, and when none matches a file is searched unless the first of them is an --include. --exclude-dir skips directories. Without -r a directory operand is an error ('Is a directory') unless -d skip. Devices, FIFOs and sockets are read when named and skipped when met during a recursive search, unless -D says otherwise. A file with a NUL byte is binary: its lines are not printed, and 'grepx: FILE: binary file matches' is written to standard error instead (as GNU grep has done since 3.5), unless -a treats it as text or -I as a file without a match.

Where GNU grep lets the last of several conflicting options win, grepx applies a fixed order, whatever order they were given in: -F beats -E, which beats -G; -H beats -h; --no-ignore-case beats -i; -A and -B beat -C, which beats -NUM; -I beats -a, and either beats --binary-files; -R beats -r, and either beats -d; -l beats -L; --no-group-separator beats --group-separator; --color beats --colour. As in GNU, -q overrides -l and -L, which override -c.

Differences from GNU grep: -P (Perl-compatible regular expressions) is refused, because the regular expression engine is RE2, not PCRE. Back references (\1 to \9) are refused for the same reason. \< and \> both match at any word boundary, as \b does, and the boundary escapes see ASCII words only; \w, \W, -w and the letter classes know Unicode letters and digits. Input is read as UTF-8 whatever the locale, a byte that is not valid UTF-8 matches like any other character, and such bytes do not make a file binary: only a NUL byte does. With -z nothing is binary. Multi-character collating symbols and equivalence classes are refused. A repetition count above 1000 is refused ('Regular expression too big'), where GNU allows 32767. A regular expression error is reported without the file and line number of a -f pattern. GREP_COLOR (the deprecated variable) is not read; GREP_COLORS is. The digits of -NUM given in separate arguments (-1 -2) are read as one number (12). -D does not apply to standard input. --line-buffered is implied when standard output is a terminal. -U (--binary) is accepted and has no effect: no platform's line endings are translated. The obsolete -u and the short form -V (use --version) are not accepted. A file that is also the output is not detected. A missing pattern is reported as 'missing PATTERNS operand' rather than with a usage line. Directory entries are searched in name order, where GNU uses directory order.

EXIT STATUS.
  0  a line was selected
  1  no line was selected
  2  an error occurred (a file could not be read, a pattern is invalid, a command-line mistake), even if a line was selected, unless -q found one

HELP. grepx --help shows the options and the first examples; gnux help grepx prints the full manual, and gnux help grepx OPTION explains one option.

SEE ALSO. sedx, findx, xargsx, headx, tailx, wcx, cutx, sortx, uniqx, gnux shell.

PERMISSIONS. Needs read permission on every FILE searched and on the files named by -f and --exclude-from, and search (execute) permission on the directories leading to them. A recursive search also needs read permission on each directory it descends into. A file that cannot be opened is reported ('Permission denied') and the search goes on with the rest; the exit status is then 2. -s silences those messages but not the status. grepx never modifies the files it searches.

USE CASES.
  - Finding the lines of a log that mention an error, with or without surrounding context.
  - Searching a source tree for every use of a name with -r, -n and -w.
  - Listing the files that contain, or lack, a string with -l and -L.
  - Counting occurrences per file with -c.
  - Testing in a script whether a file contains a pattern with -q and the exit status.
  - Filtering a pipeline, keeping (or with -v dropping) the lines that match.
  - Extracting only the matching parts, such as addresses or identifiers, with -o.
  - Searching for text that contains regular expression characters with -F.
  - Restricting a recursive search to some file types with --include and --exclude-dir.
  - Feeding NUL-separated file names to xargsx -0 with -lZ.

::

  grepx [OPTION]... PATTERNS [FILE]...

Examples
~~~~~~~~

::

    # Find the lines of a log that mention an error
    grepx error app.log

    # Search a source tree for a whole word, with file names and line numbers
    grepx -rnw 'TODO' src

    # Ignore case and show two lines of context around each match
    grepx -i -C 2 'timeout' server.log

    # Match either of two words with an extended regular expression
    grepx -E 'warn(ing)?|error' build.log

    # Search for text that holds regular expression characters, literally
    grepx -F 'price[0] * 1.5' *.js

    # List the files under the current directory that contain a string
    grepx -rl 'deprecated' .

    # List the Go files that lack a license header
    grepx -rL --include='*.go' 'SPDX-License-Identifier' .

    # Count the matching lines in each file
    grepx -c '^func ' *.go

    # Drop comment lines and empty lines from a configuration file
    grepx -v -e '^#' -e '^$' app.conf

    # Print only the matched parts, here every IPv4-looking address
    grepx -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' access.log

    # Test in a script whether a file contains a line, printing nothing
    grepx -q '^version = ' config.toml && echo found

    # Search a tree but skip the dependency and VCS directories
    grepx -rn --exclude-dir=node_modules --exclude-dir=.git 'fetch(' .

    # Stop after the first three matches and number them
    grepx -n -m 3 'panic' crash.log

    # Take the patterns from a file, one to a line, as fixed strings
    grepx -F -f blocklist.txt users.csv

    # Filter another tool's output in a pipeline
    lsx -l | grepx -i '\.log$'

    # Search in the portable gnux shell, the same on every platform
    gnux shell -c 'catx notes.txt | grepx -n urgent'

Options
~~~~~~~

::

  -A, --after-context NUM       print NUM lines of trailing context
  -G, --basic-regexp            PATTERNS are basic regular expressions (the default)
  -B, --before-context NUM      print NUM lines of leading context
  -U, --binary                  accepted for compatibility; has no effect
      --binary-files TYPE       assume that binary files are TYPE: binary (the default), text or without-match; -I is --binary-files=without-match
  -b, --byte-offset             print the byte offset with output lines
      --color WHEN[="auto"]     highlight matches, names and separators; WHEN is always, never or auto
  -C, --context NUM             print NUM lines of output context; -NUM is the same
  -c, --count                   print only a count of selected lines per FILE
  -R, --dereference-recursive   likewise, but follow all symbolic links
  -D, --devices ACTION          how to handle devices, FIFOs and sockets; ACTION is read or skip
  -d, --directories ACTION      how to handle directories; ACTION is read (the default), recurse or skip
      --exclude GLOB            skip files that match GLOB
      --exclude-dir GLOB        skip directories that match GLOB
      --exclude-from FILE       skip files that match any file pattern from FILE
  -E, --extended-regexp         PATTERNS are extended regular expressions
  -f, --file FILE               take PATTERNS from FILE
  -l, --files-with-matches      print only the names of FILEs with selected lines
  -L, --files-without-match     print only the names of FILEs with no selected lines
  -F, --fixed-strings           PATTERNS are strings
      --group-separator SEP     print SEP on the line between groups of matches with context
      --help                    display this help and exit
  -i, --ignore-case             ignore case distinctions in patterns and data
      --include GLOB            search only files that match GLOB (a file pattern)
  -T, --initial-tab             make tabs line up (put a tab before the line)
  -v, --invert-match            select the lines that do not match
      --label LABEL             use LABEL as the file name of standard input
      --line-buffered           flush the output on every line
  -n, --line-number             print the line number with output lines
  -x, --line-regexp             match only whole lines
  -m, --max-count NUM           stop after NUM selected lines
  -h, --no-filename             suppress the file name prefix on output
      --no-group-separator      do not print a separator between groups of matches with context
      --no-ignore-case          do not ignore case distinctions (the default)
  -s, --no-messages             suppress error messages about files that do not exist or cannot be read
  -Z, --null                    print a NUL byte after each FILE name
  -z, --null-data               a data line ends in a NUL byte, not a newline
  -o, --only-matching           print only the matched (non-empty) parts of lines, one to a line
  -P, --perl-regexp             PATTERNS are Perl regular expressions (not supported: refused)
  -q, --quiet                   suppress all normal output; stop at the first selected line
  -r, --recursive               like --directories=recurse
  -e, --regexp PATTERNS         use PATTERNS for matching
      --silent                  same as --quiet
  -a, --text                    equivalent to --binary-files=text
      --version                 output version information and exit
  -H, --with-filename           print the file name with output lines
  -w, --word-regexp             match only whole words

