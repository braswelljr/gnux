.. _fmtx:

fmtx
----

Reformat each paragraph of text.

Synopsis
~~~~~~~~


fmtx fills and joins the lines of each paragraph so that they come as close as possible to a goal width without exceeding the maximum width, and writes the result to standard output. With no FILE, or when FILE is -, it reads standard input. Paragraphs are separated by blank lines and by changes of indentation; the indentation of each paragraph is kept. The line breaks are chosen as GNU fmt chooses them, minimizing raggedness over the whole paragraph and avoiding breaking after the first word of a sentence or before its last. -w WIDTH sets the maximum width (75 by default, at most 2500), also written -WIDTH; -g WIDTH sets the goal (93% of the maximum by default, and without -w the maximum becomes the goal plus 10). -c (crown margin) keeps the indentation of the first two lines and aligns the rest with the second; -t (tagged paragraph) is the same, except that a first line indented like the second is a paragraph of its own. -s only splits long lines and never joins short ones, which suits code. -u puts one space between words and two after sentences. -p STRING reformats only the lines that begin with STRING (after any indentation) and puts it back on each output line; the other lines are copied unchanged. A sentence ends at '.', '?' or '!' followed by the end of the line or two spaces, as in GNU fmt.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. fmtx --help shows the options and the first examples; gnux help fmtx prints the full manual, and gnux help fmtx OPTION explains one option.

SEE ALSO. foldx, nlx.

PERMISSIONS. Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

USE CASES.
  - Rewrapping a commit message, README paragraph or email to a fixed width.
  - Joining hard-wrapped text into evenly filled lines.
  - Reflowing comment blocks with -p '#' or -p '//'.
  - Splitting overlong lines without joining short ones with -s.
  - Normalizing spacing between words and sentences with -u.

::

  fmtx [-WIDTH] [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Reformat a file to the default 75 columns
    fmtx notes.txt

    # Reformat to 72 columns
    fmtx -w 72 message.txt

    # Use the obsolete width form
    fmtx -60 notes.txt

    # Split long lines but never join short ones
    fmtx -s -w 80 script.txt

    # Reflow a shell comment block
    fmtx -p '#' -w 72 config.sh

    # Use uniform spacing
    fmtx -u draft.txt

    # Keep a hanging indent
    fmtx -t list.txt

    # Keep the first two lines' indentation
    fmtx -c letter.txt

    # Set the goal width as well as the maximum
    fmtx -w 80 -g 70 notes.txt

    # Reformat standard input
    catx a.txt b.txt | fmtx -w 66

    # Save a reformatted copy
    fmtx -w 72 draft.txt > final.txt

    # Reformat in the portable gnux shell
    gnux shell -c 'fmtx -w 60 notes.txt | headx'

Options
~~~~~~~

::

  -c, --crown-margin       preserve indentation of first two lines
  -g, --goal string        goal width (default of 93% of width)
      --help               display this help and exit
  -p, --prefix STRING      reformat only lines beginning with STRING, reattaching the prefix to reformatted lines
  -s, --split-only         split long lines, but do not refill
  -t, --tagged-paragraph   indentation of first line different from second
  -u, --uniform-spacing    one space between words, two after sentences
      --version            output version information and exit
  -w, --width WIDTH        maximum line width (default of 75 columns)

