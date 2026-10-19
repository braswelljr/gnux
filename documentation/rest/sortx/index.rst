.. _sortx:

sortx
-----

Sort lines of text files.

Synopsis
~~~~~~~~


sortx writes the sorted concatenation of all FILEs to standard output. With no FILE, or when FILE is -, it reads standard input. By default whole lines are compared byte by byte. -n, -g, -h, -M and -V compare as numbers, floating-point numbers, sizes with SI suffixes (2K, 1G), month names and version numbers; -b, -d, -f and -i ignore leading blanks, keep only blanks and alphanumerics, fold case and keep only printable characters; -r reverses the result. -k POS1[,POS2] sorts on a key, where each POS is F[.C][OPTS]: field F, character C, and any of the ordering letters bdfghiMnRrV, which then apply to that key alone. A key with no letters inherits the global options. Fields are separated by -t SEP, or else by the change from a non-blank to a blank character, so a field keeps its leading blanks. When all keys compare equal, the whole lines are compared as a last resort, unless -s (stable) or -u (unique, which keeps the first of each run of equal lines) is given. -c and -C check whether the input is already sorted, -m merges inputs that already are, -o FILE writes the result to FILE (which may be one of the inputs), and -z makes NUL the line delimiter. Ordering options that contradict each other, such as -n with -g, are refused. -R shuffles the lines but keeps equal keys together, by ordering on a hash of each key chosen by 16 random bytes; --random-source=FILE takes those bytes from FILE instead, which makes the order repeatable and the same as GNU sort's for the same FILE. Differences from GNU sort: comparison always follows the C locale (byte order, ASCII case folding and month names, '.' as the decimal point, no thousands separator) whatever LANG and LC_ALL say; --debug is not implemented; -S, -T, --parallel, --batch-size and --compress-program are accepted and ignored, because sortx sorts in memory; when -c and -C are both given, the diagnosing -c wins whatever their order, so disorder is never hidden.

EXIT STATUS.
  0  success; with -c or -C, the input is sorted
  1  with -c or -C, the input is not sorted
  2  trouble: an input could not be read, the output could not be written, or a command-line mistake

HELP. sortx --help shows the options and the first examples; gnux help sortx prints the full manual, and gnux help sortx OPTION explains one option.

SEE ALSO. headx, tailx, wcx, catx.

PERMISSIONS. Needs read permission on each FILE and search permission on the directories leading to it. With -o, needs write permission on the output file, or on its directory when it does not exist yet. Every input is checked to be readable before the output file is opened, and the output is only emptied once all input has been read, so sorting a file onto itself is safe.

USE CASES.
  - Putting a list of names, words or paths into a predictable order.
  - Ordering numbers or sizes correctly with -n or -h instead of alphabetically.
  - Sorting a CSV-like or colon-separated file on one column with -t and -k.
  - Removing duplicate lines in one step with -u.
  - Sorting release tags or file names that contain version numbers with -V.
  - Shuffling a playlist or a test order with -R, repeatably with --random-source.
  - Finding the largest entries of a listing with -rn and headx.
  - Merging several already sorted files cheaply with -m.
  - Verifying in a script that a file is sorted with -c or -C.
  - Sorting a file in place with -o.
  - Handling NUL-separated file lists safely with -z.

::

  sortx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Sort the lines of a file
    sortx names.txt

    # Sort in reverse order
    sortx -r names.txt

    # Sort numbers by value rather than as text
    sortx -n numbers.txt

    # Sort and drop duplicate lines
    sortx -u words.txt

    # Sort case-insensitively
    sortx -f names.txt

    # Sort a colon-separated file numerically on its third field
    sortx -t : -k 3,3n /etc/passwd

    # Sort on the second column, then by the first in reverse
    sortx -k 2,2 -k 1,1r scores.txt

    # Sort human-readable sizes
    printf '2K\n1G\n10M\n' | sortx -h

    # Sort version numbers naturally
    printf 'v1.10\nv1.2\nv1.9\n' | sortx -V

    # Print the three largest numbers of a file
    sortx -rn numbers.txt | headx -n 3

    # Sort a file in place
    sortx -o names.txt names.txt

    # Merge files that are already sorted
    sortx -m jan.txt feb.txt mar.txt

    # Fail a script when a file is not sorted (exit status 1)
    sortx -C names.txt || echox 'names.txt is not sorted'

    # Report the first line out of order
    sortx -c names.txt

    # Sort a NUL-separated list of file names
    find . -print0 | sortx -z

    # Sort a directory listing in the portable gnux shell
    gnux shell -c 'lsx -1 | sortx -r'

    # Run the test files in a shuffled order that a seed file makes repeatable
    lsx tests/*.sh | sortx -R --random-source=ci-seed.bin

Options
~~~~~~~

::

      --batch-size NMERGE               accepted for compatibility and ignored (merge at most NMERGE inputs at once)
  -S, --buffer-size SIZE                accepted for compatibility and ignored (SIZE of the main memory buffer)
  -c, --check WHEN[="diagnose-first"]   check for sorted input; do not sort. WHEN (diagnose-first, quiet, silent) says whether to report the first disorder
  -C, --check-quiet                     like -c, but do not report first bad line
      --compress-program PROG           accepted for compatibility and ignored (PROG to compress temporaries)
  -d, --dictionary-order                consider only blanks and alphanumeric characters
  -t, --field-separator SEP             use SEP instead of non-blank to blank transition
      --files0-from F                   read input from the files specified by NUL-terminated names in file F; if F is - then read names from standard input
  -g, --general-numeric-sort            compare according to general numerical value
      --help                            display this help and exit
  -h, --human-numeric-sort              compare human readable numbers (e.g., 2K 1G)
  -f, --ignore-case                     fold lower case to upper case characters
  -b, --ignore-leading-blanks           ignore leading blanks
  -i, --ignore-nonprinting              consider only printable characters
  -k, --key KEYDEF                      sort via a key; KEYDEF gives location and type
  -m, --merge                           merge already sorted files; do not sort
  -M, --month-sort                      compare (unknown) < 'JAN' < ... < 'DEC'
  -n, --numeric-sort                    compare according to string numerical value
  -o, --output FILE                     write result to FILE instead of standard output
      --parallel N                      accepted for compatibility and ignored (N concurrent sorts)
  -R, --random-sort                     shuffle, but group identical keys
      --random-source FILE              get random bytes from FILE
  -r, --reverse                         reverse the result of comparisons
      --sort WORD                       sort according to WORD: general-numeric -g, human-numeric -h, month -M, numeric -n, random -R, version -V
  -s, --stable                          stabilize sort by disabling last-resort comparison
  -T, --temporary-directory DIR         accepted for compatibility and ignored (DIR for temporaries)
  -u, --unique                          output only the first of an equal run
      --version                         output version information and exit
  -V, --version-sort                    natural sort of (version) numbers within text
  -z, --zero-terminated                 line delimiter is NUL, not newline

