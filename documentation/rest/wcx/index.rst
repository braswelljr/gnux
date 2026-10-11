.. _wcx:

wcx
---

Print newline, word, and byte counts for each file.

Synopsis
~~~~~~~~


wcx prints the number of lines, words and bytes in each FILE, followed by the file name, and a 'total' line when more than one FILE is given. With no FILE, or when FILE is -, it reads standard input. -l, -w, -m, -c and -L select which counts to print: newlines, words, characters, bytes and the display width of the longest line. Whatever the order of the options, the counts are always printed in the order lines, words, characters, bytes, maximum line length. A word is a run of non-whitespace characters; input is decoded as UTF-8, and bytes that are not valid UTF-8 count toward words and bytes but not characters. --total=WHEN controls the summary line and --files0-from=F reads the file names, NUL-terminated, from F. Columns are right-aligned to a width derived from the files' sizes, as in GNU wc.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. wcx --help shows the options and the first examples; gnux help wcx prints the full manual, and gnux help wcx OPTION explains one option.

SEE ALSO. catx, headx, tailx, lsx.

PERMISSIONS. Needs read permission on each FILE and search permission on the directories leading to it. With -c alone on a regular file only its metadata is consulted, so search permission on the directory is enough. Nothing is ever modified.

USE CASES.
  - Counting the lines of a file or of a pipeline's output with -l.
  - Measuring the size of a set of source files, with a total.
  - Checking a text against a word or character limit.
  - Finding the widest line of a file with -L before formatting it.
  - Getting a file's size in bytes cheaply with -c.
  - Counting the entries in a directory by piping lsx into it.
  - Getting a single grand total for a large, generated list of files with --files0-from and --total=only.
  - Verifying in a script that a file is not empty, or that output has an expected length.

::

  wcx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Count lines, words and bytes of a file
    wcx README.md

    # Count only the lines
    wcx -l access.log

    # Count the words of an essay
    wcx -w essay.txt

    # Print the size of a file in bytes
    wcx -c image.png

    # Count characters rather than bytes
    wcx -m unicode.txt

    # Find the display width of the longest line
    wcx -L README.md

    # Print lines and words together (always in that order, whatever the flag order)
    wcx -wl essay.txt

    # Count the lines of several files, with a total
    wcx -l *.go

    # Count several files without the total line, or force it for a single file
    wcx -l --total=never *.go && wcx -l --total=always main.go

    # Count the entries a command produces
    lsx | wcx -l

    # Count the words of text from standard input
    echox 'the quick brown fox' | wcx -w

    # Read the list of files to count from a NUL-separated file
    wcx -l --files0-from=sources.lst

    # Print only the grand total for a list of files found elsewhere
    find . -name '*.go' -print0 | wcx -l --files0-from=- --total=only

    # Fail a script when a file is missing (exit status 1)
    wcx -l missing.txt || echox "exit status $?"

    # Count matching files in the portable gnux shell
    gnux shell -c 'lsx -1 *.md | wcx -l'

Options
~~~~~~~

::

  -c, --bytes             print the byte counts
  -m, --chars             print the character counts
      --files0-from F     read input from the files specified by NUL-terminated names in file F; if F is - then read names from standard input
      --help              display this help and exit
  -l, --lines             print the newline counts
  -L, --max-line-length   print the maximum display width
      --total WHEN        when to print a line with total counts; WHEN can be: auto, always, only, never (default "auto")
      --version           output version information and exit
  -w, --words             print the word counts

