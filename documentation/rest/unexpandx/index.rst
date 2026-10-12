.. _unexpandx:

unexpandx
---------

Convert spaces to tabs.

Synopsis
~~~~~~~~


unexpandx writes each FILE to standard output with runs of blanks replaced by tabs where they reach a tab stop. With no FILE, or when FILE is -, it reads standard input. By default only the blanks at the start of each line are converted; -a converts every run of two or more blanks that ends at a tab stop, and --first-only goes back to the leading blanks only. Tab stops are every 8 columns unless -t says otherwise, with the same syntax as expandx: -t N, or -t LIST with an optional final /N or +N. -t implies -a, as in GNU. Several -t options add to one list, and the obsolete -N form (unexpandx -4) is a list of its own, in the order given, without implying -a; GNU's comma form -4,8 is not accepted, and the digits of two -N options in a row run together (-4 -8 reads as -48), so write -t 4,8 instead. When both -a and --first-only are given, --first-only wins.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. unexpandx --help shows the options and the first examples; gnux help unexpandx prints the full manual, and gnux help unexpandx OPTION explains one option.

SEE ALSO. expandx, foldx, trx.

PERMISSIONS. Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

USE CASES.
  - Converting space-indented files to tab indentation (Makefiles, Go code).
  - Shrinking files that are padded with spaces.
  - Reversing expandx after editing.
  - Converting the indentation alone, leaving aligned columns inside lines untouched.
  - Normalizing indentation width with -t before a commit.

::

  unexpandx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Convert leading spaces to tabs every 8 columns
    unexpandx notes.txt

    # Convert all runs of blanks, not just leading ones
    unexpandx -a table.txt

    # Use 4-column tab stops (implies -a)
    unexpandx -t 4 main.go

    # Use 4-column stops but only for indentation
    unexpandx -t 4 --first-only main.go

    # Use the obsolete form, which converts only leading blanks
    unexpandx -4 main.go

    # Put tab stops at specific columns
    unexpandx -t 10,20,40 report.txt

    # Round-trip a file through expandx
    expandx -t 4 a.go | unexpandx -t 4 --first-only > b.go

    # Convert standard input
    printfx '        x\n' | unexpandx

    # Save a converted copy
    unexpandx -t 8 spaces.mk > Makefile

    # See how many bytes the conversion saves
    unexpandx -a big.txt | wcx -c

    # Convert in the portable gnux shell
    gnux shell -c 'unexpandx -t 4 main.go > tabs.go'

Options
~~~~~~~

::

  -a, --all            convert all blanks, instead of just initial blanks
      --first-only     convert only leading sequences of blanks (overrides -a)
      --help           display this help and exit
  -t, --tabs N, LIST   have tabs N characters apart instead of 8 (enables -a), or use comma separated list of explicit tab positions
      --version        output version information and exit

