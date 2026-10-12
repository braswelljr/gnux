.. _expandx:

expandx
-------

Convert tabs to spaces.

Synopsis
~~~~~~~~


expandx writes each FILE to standard output with every tab replaced by the spaces that reach the next tab stop. With no FILE, or when FILE is -, it reads standard input. Tab stops are every 8 columns unless -t says otherwise: -t N puts them every N columns, and -t LIST (numbers separated by commas or blanks, in ascending order) puts them at those columns; the last number may be written /N for a stop at every multiple of N after it, or +N for a stop every N columns after it. Past the last listed stop a tab becomes one space. Several -t options add to one list, and the obsolete -N form (expandx -4) is a list of its own, in the order given; GNU's comma form -4,8 is not accepted, and the digits of two -N options in a row run together (-4 -8 reads as -48), so write -t 4,8 instead. -i converts only the tabs before the first non-blank character of each line. A backspace moves back one column. Like GNU expand in the C locale, expandx counts bytes rather than characters.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. expandx --help shows the options and the first examples; gnux help expandx prints the full manual, and gnux help expandx OPTION explains one option.

SEE ALSO. unexpandx, foldx, trx.

PERMISSIONS. Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

USE CASES.
  - Converting tab-indented source to spaces for a style guide.
  - Making tab-separated output line up the same way in every viewer.
  - Preparing text for tools that count characters, like cutx -c or foldx.
  - Converting only the indentation with -i, leaving tabs inside lines alone.
  - Laying out columns at specific positions with -t LIST.

::

  expandx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Make a tab-indented C file look the same in every editor
    expandx main.c

    # Convert a Go file to 4-space indentation for a code review tool
    expandx -t 4 main.go

    # Run an old script that uses the obsolete form
    expandx -4 main.go

    # Line a tab-separated report up in fixed columns for printing
    expandx -t 10,20,40 table.tsv

    # Indent the first level by 4 and the rest by 8
    expandx -t 4,/8 data.txt

    # Fix YAML indentation (tabs are invalid there) without touching tabs inside values
    expandx -i -t 2 config.yml

    # See exactly where a tab lands
    printfx 'a\tb\n' | expandx

    # Convert a Python file that mixes tabs and spaces
    expandx -t 4 tabs.py > spaces.py

    # Preview a TSV file with readable columns
    expandx -t 16 report.tsv | headx

    # Combine several files into one, with tabs converted
    expandx a.txt b.txt > combined.txt

    # Find the longest line after expansion in the portable gnux shell
    gnux shell -c 'expandx -t 4 main.go | wcx -L'

Options
~~~~~~~

::

      --help           display this help and exit
  -i, --initial        do not convert tabs after non blanks
  -t, --tabs N, LIST   have tabs N characters apart, not 8, or use comma separated list of explicit tab positions
      --version        output version information and exit

