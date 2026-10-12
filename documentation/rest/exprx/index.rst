.. _exprx:

exprx
-----

Evaluate an expression and print its value.

Synopsis
~~~~~~~~


exprx evaluates EXPRESSION, given as separate arguments, and prints its value. From the loosest binding: A | B is A if it is neither empty nor 0, otherwise B (or 0); A & B is A if neither is empty nor 0, otherwise 0; A = B (also ==), A != B, and the orderings compare as integers when both sides are integers and as strings otherwise, giving 1 or 0; + and - add and subtract; * / and % multiply, divide (truncating toward zero) and take the remainder; STRING : REGEX matches. Integers are an optional minus sign and decimal digits, of any size. A match is anchored at the start of STRING: when REGEX has a \( \) group the value is the text the first group matched (or empty), otherwise the number of bytes matched (or 0). REGEX is a POSIX basic regular expression with GNU's \+, \? and \| extensions and character classes such as [[:digit:]]. The keywords match STRING REGEX, substr STRING POS LENGTH, index STRING CHARS and length STRING do the same as their names, and + TOKEN takes TOKEN as a string even if it is a keyword or an operator. Parentheses group. Operators must be separate arguments, and shell characters such as * | & ( ) must be quoted. A leading -- is skipped. The exit status is 0 when the value is neither empty nor 0, 1 when it is, 2 for an invalid expression (syntax error, non-integer argument, division by zero, a bad regular expression) and 3 for other errors. Differences from GNU expr: strings are measured and sliced in bytes and compared byte by byte (the C locale), and back references (\1) and the [= =] and [. .] bracket forms are not supported in a REGEX.

EXIT STATUS.
  0  the value is neither empty nor 0
  1  the value is empty or 0
  2  EXPRESSION is invalid, or missing
  3  another error occurred, such as a failed write

HELP. exprx --help shows the options and the first examples; gnux help exprx prints the full manual, and gnux help exprx OPTION explains one option.

SEE ALSO. testx, seqx, printfx, gnux shell.

PERMISSIONS. Needs no filesystem permissions; it only writes to standard output.

USE CASES.
  - Doing integer arithmetic in a script, with no size limit.
  - Incrementing a counter in the portable gnux shell, where there is no $((...)).
  - Extracting part of a string with a regular expression group.
  - Getting the length of a string or a substring of it.
  - Checking that a value is numeric before using it.
  - Comparing two values numerically or as strings and branching on the status.

::

  exprx EXPRESSION

Examples
~~~~~~~~

::

    # Add two numbers
    exprx 2 + 3

    # Multiply (quote the * from the shell)
    exprx 6 '*' 7

    # Integer division and remainder
    exprx 17 / 5 && exprx 17 % 5

    # Increment a counter in a script
    n=$(exprx "$n" + 1)

    # Work with numbers of any size
    exprx 99999999999999999999 '*' 3

    # Compare two values (prints 1 or 0)
    exprx 10 '>' 9

    # Get the length of a string
    exprx length 'hello world'

    # Take a substring (position 1, three bytes)
    exprx substr release-2024 9 4

    # Strip an extension with a regular expression group
    exprx report.txt : '\(.*\)\.txt'

    # Check that a value is all digits
    exprx "$VALUE" : '[0-9]*$' > /dev/null && echox numeric

    # Find the first vowel
    exprx index gnux aeiou

    # Use a keyword as a plain string
    exprx + length

    # See the status of an invalid expression (2)
    exprx 1 / 0; echox "exit status $?"

    # Count in the portable gnux shell
    gnux shell -c 'N=4; exprx $N + 1'

Options
~~~~~~~

::

      --help      display this help and exit
      --version   output version information and exit

