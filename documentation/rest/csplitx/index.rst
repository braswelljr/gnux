.. _csplitx:

csplitx
-------

Split a file into sections determined by context lines.

Synopsis
~~~~~~~~


csplitx copies FILE (- for standard input) into files xx00, xx01, ..., starting a new file at each PATTERN, and prints the size in bytes of each file it makes. A PATTERN is a line number N (the next file starts at line N), /REGEXP/ (the next file starts at the next line matching the POSIX basic regular expression REGEXP), or %REGEXP% (the lines up to that match are skipped, not written). A regular expression may be followed by an offset, +N or -N, that moves the split that many lines after or before the matching line. A pattern followed by {N} is repeated N more times, and by {*} as many times as it matches; a line number repeated advances by its own value each time. The lines after the last split go to a final file. -f PREFIX replaces xx, -n N uses N digits, and -b FORMAT names the files with a printf format holding one integer conversion (%02d.txt, %03x). -s (or -q) does not print the sizes, -z removes empty files (and reuses their number), and --suppress-matched leaves the matching lines out. If a pattern cannot be satisfied (a line number past the end, a regular expression that never matches), csplitx reports it and removes every file it made, so a failed run leaves nothing half-done behind; -k keeps them. Lines are compared as bytes.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. csplitx --help shows the options and the first examples; gnux help csplitx prints the full manual, and gnux help csplitx OPTION explains one option.

SEE ALSO. splitx, headx, tailx, catx.

PERMISSIONS. Needs read permission on FILE and write permission on the directory the files are created in.

USE CASES.
  - Splitting a book or a long Markdown document into one file per chapter.
  - Breaking a concatenated log into one file per day or per request.
  - Separating the sections of a config file or a multi-document YAML stream.
  - Cutting off a file's header or preamble with %REGEXP%.
  - Pulling the first N lines into one file and the rest into another.

::

  csplitx [OPTION]... FILE PATTERN...

Examples
~~~~~~~~

::

    # Split a Markdown book into one file per chapter (each starting with '# ')
    csplitx -f chapter- -b '%02d.md' book.md '/^# /' '{*}'

    # Do the same without the empty file before the first chapter
    csplitx -z -f chapter- -b '%02d.md' book.md '/^# /' '{*}'

    # Split a multi-document Kubernetes YAML into one manifest per file, dropping the '---' lines
    csplitx -z --suppress-matched -f manifest- -b '%02d.yaml' all.yaml '/^---$/' '{*}'

    # Separate the 3-line header of a report from its body
    csplitx -s -f report- report.txt 4

    # Cut a log into one file per day, where each day starts with a '=== 2026-' banner
    csplitx -z -f day- app.log '/^=== 2026-/' '{*}'

    # Drop everything before the line 'BEGIN DATA' and keep the rest
    csplitx -s export.txt '%^BEGIN DATA%1'

    # Start each file two lines before the match, to keep a heading's preceding blank line and title
    csplitx notes.txt '/^-----$/-2' '{*}'

    # Split a 400-line file into 100-line pieces
    csplitx data.txt 101 '{2}'

    # Keep the pieces even if a later pattern fails
    csplitx -k data.txt 100 200 5000

    # Split standard input from another command
    git log --format='commit %H%n%B' | csplitx -s -z -f commit- - '/^commit /' '{*}'

    # Split by chapter in the portable gnux shell and count the pieces
    gnux shell -c 'csplitx -s -z -f ch- book.md "/^# /" "{*}" && lsx ch-* | wcx -l'

Options
~~~~~~~

::

  -n, --digits digits          use specified number of digits instead of 2
  -z, --elide-empty-files      suppress generation of empty output files
      --help                   display this help and exit
  -k, --keep-files             do not remove output files on errors
  -f, --prefix PREFIX          use PREFIX instead of 'xx'
  -s, --quiet                  do not print counts of output file sizes
  -q, --silent                 same as --quiet
  -b, --suffix-format FORMAT   use sprintf FORMAT instead of %02d
      --suppress-matched       suppress the lines matching PATTERN
      --version                output version information and exit

