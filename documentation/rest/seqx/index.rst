.. _seqx:

seqx
----

Print a sequence of numbers.

Synopsis
~~~~~~~~


seqx prints the numbers from FIRST to LAST, in steps of INCREMENT, one per line: 'seqx LAST', 'seqx FIRST LAST' or 'seqx FIRST INCREMENT LAST'. FIRST and INCREMENT default to 1, and INCREMENT may be negative to count down; nothing is printed when FIRST is already past LAST. Negative operands need no quoting or '--': as in GNU seq, an argument that starts with '-' followed by a digit or '.' is a number, and options must come before the operands. Without -f, numbers are printed with as many decimal places as FIRST and INCREMENT have, and the arithmetic is exact, so 'seqx 0.1 0.1 0.3' ends at 0.3. -f FORMAT prints each number through one printf floating-point directive (%e, %f, %g or %a, with flags, width and precision) and literal text around it; -s STRING separates the numbers with STRING instead of a newline; -w pads them with leading zeros to equal width, and cannot be combined with -f. LAST may be 'inf' to count forever, until the reader of the output goes away. Differences from GNU seq: -f values are formatted from double precision rather than long double, and an infinite FIRST or INCREMENT is refused as an invalid argument.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. seqx --help shows the options and the first examples; gnux help seqx prints the full manual, and gnux help seqx OPTION explains one option.

SEE ALSO. printfx, headx, yesx.

PERMISSIONS. Needs no filesystem permissions; it only writes to standard output.

USE CASES.
  - Driving a loop over numbered items in a script, portably on every platform.
  - Generating test data: a known number of lines to feed into another tool.
  - Producing zero-padded numbers with -w for file names that sort correctly.
  - Building comma- or space-separated lists of numbers with -s.
  - Counting down with a negative increment.
  - Generating evenly spaced decimal values, exactly, for tables or plots.
  - Formatting numbers with a prefix, suffix or fixed precision through -f.
  - Producing an endless numbered stream with 'inf' that a later stage cuts short.

::

  seqx [OPTION]... [FIRST [INCREMENT]] LAST

Examples
~~~~~~~~

::

    # Print the numbers 1 to 5
    seqx 5

    # Print the numbers from 3 to 7
    seqx 3 7

    # Count in steps of 5
    seqx 0 5 25

    # Count down with a negative increment
    seqx 10 -2 0

    # Start from a negative number, with no quoting needed
    seqx -3 3

    # Step through decimals exactly
    seqx 0 0.25 1

    # Pad the numbers with zeros to equal width
    seqx -w 8 12

    # Separate the numbers with commas on one line
    seqx -s , 1 10

    # Format each number with a prefix and fixed precision
    seqx -f 'item-%03.0f' 1 3

    # Print percentages, with %% for a literal percent sign
    seqx -f '%.0f%%' 0 25 100

    # Number lines endlessly, stopping when the reader has enough
    seqx 1 inf | headx -n 3

    # Create ten numbered, zero-padded files
    touchx $(seqx -f 'log-%02.0f.txt' 1 10)

    # Sum the numbers 1 to 100 with another program
    seqx -s + 1 100 | bc

    # Count the numbers a sequence produces
    seqx 0 3 100 | wcx -l

    # Generate numbered lines in the portable gnux shell
    gnux shell -c 'seqx -f "line %g" 3 > lines.txt; catx lines.txt'

Options
~~~~~~~

::

  -w, --equal-width        equalize width by padding with leading zeroes
  -f, --format FORMAT      use printf style floating-point FORMAT
      --help               display this help and exit
  -s, --separator STRING   use STRING to separate numbers (default "\\n")
      --version            output version information and exit

