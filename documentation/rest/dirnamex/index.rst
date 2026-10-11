.. _dirnamex:

dirnamex
--------

Strip the last component from file names.

Synopsis
~~~~~~~~


dirnamex prints each NAME with its last non-slash component and trailing slashes removed. If NAME contains no slash the result is '.', the current directory, and anything directly under the root yields the root itself. The rest of the path is left exactly as written: no symbolic links are resolved and no '.' or '..' components are collapsed. -z separates the output with NUL instead of newlines, for names that may contain newlines.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. dirnamex --help shows the options and the first examples; gnux help dirnamex prints the full manual, and gnux help dirnamex OPTION explains one option.

SEE ALSO. basenamex, realpathx, mkdirx.

PERMISSIONS. dirnamex works on the text of its operands only: it never touches the filesystem, so the names need not exist and no permissions are required.

USE CASES.
  - Finding the directory a script lives in from $0.
  - Computing the parent directory to create before writing a file.
  - Splitting a list of paths into their directories in a pipeline.
  - Walking up a path several levels by applying dirnamex repeatedly.
  - Getting the name of a file's containing folder together with basenamex.
  - Finding the real directory of a file behind symbolic links together with realpathx.
  - Producing NUL-separated output with -z for names that may contain newlines.

::

  dirnamex [OPTION] NAME...

Examples
~~~~~~~~

::

    # Print the directory part of a path
    dirnamex /usr/bin/sort

    # Handle several names at once
    dirnamex dir1/str dir2/str stdio.h

    # Ignore trailing slashes
    dirnamex /usr/bin/

    # Get '.' for a bare name and the root for a top-level entry
    dirnamex stdio.h /etc

    # Leave '.' and '..' exactly as written (no normalization)
    dirnamex a/../b/./c

    # Separate the output with NUL bytes instead of newlines
    dirnamex -z a/b c/d | od -c

    # Create the directory a file will be written to
    mkdirx -p "$(dirnamex out/reports/q1.csv)"

    # Create a log file's directory, then the file
    d=$(dirnamex logs/2024/app.log) && mkdirx -p "$d" && touchx logs/2024/app.log

    # Change to the directory a script lives in
    cd "$(dirnamex "$0")" && pwdx

    # Go up two levels
    dirnamex "$(dirnamex /srv/app/releases/v1/bin)"

    # Print the name of the folder that contains a file
    basenamex "$(dirnamex /var/lib/app/data.db)"

    # Find the real directory of a file behind symbolic links
    dirnamex "$(realpathx scripts/run.sh)"

    # Split several paths into their directories with the gnux shell
    gnux shell -c 'dirnamex src/main.go docs/guide.md README.md'

Options
~~~~~~~

::

      --help      display this help and exit
      --version   output version information and exit
  -z, --zero      end each output line with NUL, not newline

