.. _envx:

envx
----

Run a program in a modified environment, or print the environment.

Synopsis
~~~~~~~~


envx sets each NAME to VALUE in the environment and runs COMMAND with the remaining arguments; without a COMMAND it prints the resulting environment, one NAME=VALUE per line. -i (or a lone '-') starts from an empty environment, -u removes a variable, -C changes the working directory for the command and -S splits a single string into arguments, which is what makes options usable on a '#!' line. Printed variables keep their inherited order: assigning an existing name replaces it in place, and new names are appended. COMMAND is looked up on the PATH of the modified environment. Where GNU env replaces itself with the command, envx runs it as a child process connected to the same standard streams and exits with the command's status, so the behavior is identical on platforms that cannot replace a running process. The exit status is 125 if envx itself fails, 126 if COMMAND is found but cannot be run, 127 if COMMAND cannot be found, and otherwise the exit status of COMMAND (128 plus the signal number when a signal killed it).

EXIT STATUS.
  0-124, 128+  the exit status of COMMAND (on Unix, 128+N when it is killed by signal N); with no COMMAND, 0
  125  envx itself failed: a bad option, a variable it could not unset, a directory it could not enter (-C), a write error
  126  COMMAND was found but could not be run
  127  COMMAND was not found

HELP. envx --help shows the options and the first examples; gnux help envx prints the full manual, and gnux help envx OPTION explains one option.

SEE ALSO. gnux shell, echox, sleepx.

PERMISSIONS. Printing the environment needs no permissions. Running a command needs execute permission on the program and search permission on the directories leading to it, plus search permission on the -C directory when one is given.

USE CASES.
  - Setting variables for a single command in a way that works in every shell.
  - Running a command with a clean, reproducible environment using -i.
  - Locating an interpreter through PATH on a '#!' line, with arguments via -S.
  - Dumping the environment for debugging, or NUL-separated with -0 for safe parsing.
  - Removing a variable that changes a program's behavior (DEBUG, GOFLAGS, LANG) for one run with -u.
  - Running a command in another directory with -C without changing the caller's directory.
  - Cross-compiling or configuring builds with per-command variables such as GOOS and GOARCH.
  - Checking how a command fails: 127 when it is not found, 125 when envx itself fails.
  - Tracing exactly which environment and arguments a command receives with -v.
  - Running the gnux multi-call binary as a chosen tool by setting its zeroth argument with -a.

::

  envx [OPTION]... [-] [NAME=VALUE]... [COMMAND [ARG]...]

Examples
~~~~~~~~

::

    # Print the current environment
    envx

    # Show only the first few variables
    envx | headx -n 3

    # Run a command with extra variables
    envx GOOS=windows GOARCH=amd64 go build ./...

    # Check that an assignment is appended to the printed environment
    envx GREETING=hello | tailx -n 1

    # Run a command with nothing but the variables given
    envx -i PATH=/usr/bin HOME=/tmp make test

    # Print an empty environment plus a single variable
    envx -i LANG=C.UTF-8

    # Drop variables for one command (-u may be repeated)
    envx -u HOME -u USER gnux shell -c 'echox "[$HOME] [$USER]"'

    # Drop a variable and run in another directory
    envx -u DEBUG -C build ./configure

    # Run a tool in another directory without leaving this one
    envx -C build pwdx

    # Print the environment NUL-separated, with the NULs made visible
    envx -0 -i A=1 B=2 | catx -v

    # Split one string into a command and its arguments, as on a '#!' line
    envx -S 'gnux shell -e' build.sh

    # Trace each step envx takes before running the command
    envx -v -C build pwdx

    # Run the gnux binary as rmx by setting its zeroth argument
    envx -a rmx gnux -v old.log

    # Tell a missing command (status 127) from a failing one
    envx no-such-command; echox "status: $?"

    # Set a variable for a command run from the gnux shell
    gnux shell -c 'envx GREETING=hi envx | tailx -n 1'

Options
~~~~~~~

::

  -a, --argv0 ARG            pass ARG as the zeroth argument of COMMAND
  -C, --chdir DIR            change working directory to DIR
  -v, --debug                print verbose information for each processing step
      --help                 display this help and exit
  -i, --ignore-environment   start with an empty environment
  -0, --null                 end each output line with NUL, not newline
  -S, --split-string S       process and split S into separate arguments;
                             used to pass multiple arguments on shebang lines
  -u, --unset NAME           remove variable NAME from the environment
      --version              output version information and exit

