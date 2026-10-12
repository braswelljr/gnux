.. _printfx:

printfx
-------

Format and print data.

Synopsis
~~~~~~~~


printfx writes the ARGUMENTs to standard output under the control of FORMAT, like C's printf. FORMAT is printed as written except for backslash escapes (\", \\, \a, \b, \c, \e, \f, \n, \r, \t, \v, \NNN octal, \xHH, \uHHHH, \UHHHHHHHH) and conversion specifications: %d %i %o %u %x %X for integers, %f %F %e %E %g %G %a %A for floating point, %c for a character, %s for a string, %b for a string with its backslash escapes interpreted (octal as \0NNN), %q for a string quoted for reuse as shell input, and %% for a percent sign. A conversion may carry the flags - + space # 0, a width and a precision, either of which may be * to take it from the next ARGUMENT. FORMAT is reused as often as needed to consume every ARGUMENT; a missing ARGUMENT reads as an empty string or zero. A numeric ARGUMENT may be decimal, 0x hexadecimal, 0 octal (integers), or a quote followed by a character, meaning that character's code. An ARGUMENT that is not a valid number is diagnosed, printed as far as it converted, and makes the exit status 1 without stopping the output. \c ends all output. Unlike the shell's builtin printf, printfx behaves the same in bash, PowerShell and the gnux shell. Differences from GNU printf: floating-point values are held in double precision rather than long double, and \c ends output with the status the run already had rather than always 0.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. printfx --help shows the options and the first examples; gnux help printfx prints the full manual, and gnux help printfx OPTION explains one option.

SEE ALSO. echox, seqx, gnux shell.

PERMISSIONS. Needs no filesystem permissions; it only writes to standard output.

USE CASES.
  - Printing text without a trailing newline, or with exact control over every byte.
  - Aligning columns in a report with field widths.
  - Formatting numbers: fixed decimals, zero padding, hexadecimal or octal.
  - Converting between number bases, or printing a character's code.
  - Repeating a line template over a list of arguments.
  - Quoting a value with %q so it can be pasted back into a shell safely.
  - Writing files with exact content (escapes, no trailing newline) from scripts.
  - Getting identical formatted output in bash, PowerShell and the gnux shell.

::

  printfx FORMAT [ARGUMENT]...

Examples
~~~~~~~~

::

    # Print a line, adding the newline explicitly
    printfx 'Hello, %s\n' world

    # Format a number with two decimal places
    printfx '%.2f\n' 3.14159

    # Pad a number with zeros to five digits
    printfx '%05d\n' 42

    # Align columns with field widths
    printfx '%-10s|%5d\n' apples 3 pears 12

    # Reuse the format for every argument
    printfx '- %s\n' one two three

    # Convert a number to hexadecimal and octal
    printfx '%x %o\n' 255 255

    # Convert hexadecimal to decimal
    printfx '%d\n' 0xff

    # Print the character code of a letter
    printfx '%d\n' "'A"

    # Take the field width from an argument
    printfx '[%*s]\n' 8 right

    # Interpret escapes in an argument with %b
    printfx '%b\n' 'tab:\there'

    # Quote a value for reuse as shell input
    printfx '%q\n' "it's a file.txt"

    # Print a Unicode character by code point
    printfx 'é\n'

    # Write a file with no trailing newline
    printfx 'v1.2.3' > VERSION

    # Notice a value that is not a number (exit status 1)
    printfx '%d\n' abc || echox "exit status $?"

    # Format a table in the portable gnux shell
    gnux shell -c 'printfx "%-6s %3s\n" name qty pen 4 ink 12'

Options
~~~~~~~

::

      --help      display this help and exit
      --version   output version information and exit

