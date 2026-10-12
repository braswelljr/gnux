.. _foldx:

foldx
-----

Wrap each input line to fit in a specified width.

Synopsis
~~~~~~~~


foldx writes each FILE to standard output, breaking any line longer than 80 columns into several. With no FILE, or when FILE is -, it reads standard input. -w WIDTH sets the width; the obsolete form -WIDTH (foldx -72) is accepted too, and whichever is given last wins. Columns are counted as a terminal would: a tab advances to the next multiple of 8, a backspace moves back one column and a carriage return goes back to column 0. -b counts bytes instead, so every byte takes one position. -s breaks a long line after the last blank that fits, rather than exactly at the width, so words are not split unless one is longer than the width. Like GNU fold in the C locale, foldx counts bytes rather than characters, so a multibyte UTF-8 character takes more than one column.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. foldx --help shows the options and the first examples; gnux help foldx prints the full manual, and gnux help foldx OPTION explains one option.

SEE ALSO. fmtx, cutx, expandx.

PERMISSIONS. Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

USE CASES.
  - Wrapping long log lines or generated text to fit a terminal or an email.
  - Splitting a long base64 or hex string into fixed-width lines.
  - Breaking prose at word boundaries with -s.
  - Chunking a single-line file into records of a fixed number of bytes with -b.
  - Preparing text for a fixed-width display or printer.

::

  foldx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Wrap lines at 80 columns
    foldx notes.txt

    # Wrap lines at 40 columns
    foldx -w 40 notes.txt

    # Wrap at word boundaries
    foldx -s -w 72 essay.txt

    # Use the obsolete width form
    foldx -60 notes.txt

    # Split a long string into 64-byte lines
    foldx -b -w 64 key.txt

    # Wrap standard input
    catx long.log | foldx -w 100

    # Break a word list into one character per line
    echox abc | foldx -w 1

    # Save a wrapped copy
    foldx -s -w 72 draft.txt > wrapped.txt

    # Wrap several files in turn
    foldx -w 72 a.txt b.txt

    # Count how many wrapped lines a file makes
    foldx -w 72 notes.txt | wcx -l

    # Wrap in the portable gnux shell
    gnux shell -c 'foldx -s -w 60 notes.txt | headx'

Options
~~~~~~~

::

  -b, --bytes         count bytes rather than columns
      --help          display this help and exit
  -s, --spaces        break at spaces
      --version       output version information and exit
  -w, --width WIDTH   use WIDTH columns instead of 80

