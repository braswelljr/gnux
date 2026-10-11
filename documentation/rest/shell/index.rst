.. _shell:

shell
-----

Run a portable shell whose commands are the gnux tools.

Synopsis
~~~~~~~~


gnux shell is a small command interpreter whose commands are the gnux tools, run inside the gnux process. Because nothing in it depends on the system's own shell or utilities, the same script behaves identically on Linux, macOS and Windows. With no SCRIPT it reads commands from standard input, prompting when that is a terminal; with -c it runs the given string.

The language is a deliberately small subset of POSIX sh: pipelines (a | b), the &&, || and ; connectors, redirection (<, >, >>, and 2> for standard error; /dev/null discards on every platform), NAME=value variables with $NAME, ${NAME}, $? and $PWD expansion, single and double quotes, globbing (*, ?, [...]), ~ for the home directory and # comments. The builtins are cd, exit and help. Tools answer to both names, so rm and rmx are the same command. There are no loops, functions or conditionals beyond && and ||, and a variable's value is never re-split, so paths with spaces need no quoting once stored. External programs are not looked up; run one through the env tool (envx PROGRAM ARG...).

The exit status is that of the last command run, 2 for a syntax error, and 127 for an unknown command. With -e the script stops at the first failing command.

EXIT STATUS.
  N  the exit status of the last command run
  1  a redirection could not be opened (that command is not run)
  2  a syntax error, exit with a non-numeric argument, or a command-line mistake
  127  an unknown command, or a SCRIPT that cannot be opened

HELP. gnux shell --help shows the options and the first examples; gnux help shell prints the full manual, and gnux help shell OPTION explains one option.

SEE ALSO. envx, truex, falsex, echox.

PERMISSIONS. The shell itself needs only read access to SCRIPT. Each command it runs needs whatever its tool needs; redirection creates files with mode 0644 (before the umask).

USE CASES.
  - Build, clean and packaging scripts that must run unchanged on Windows, macOS and Linux.
  - A consistent interactive shell on machines without the GNU tools installed.
  - Embedding small file-management scripts in a Go application without spawning processes.

::

  shell [OPTION]... [SCRIPT]

Examples
~~~~~~~~

::

    # Start an interactive session
    gnux shell

    # Run a one-liner
    gnux shell -c 'mkdirx -p dist && cpx -r assets dist/ && lsx -l dist'

    # Run a script file, stopping at the first failure
    gnux shell -e build.gnux

    # Pipe a script in
    echo 'catx *.log | wcx -l' | gnux shell

Options
~~~~~~~

::

  -c, --command STRING   run the commands in STRING instead of reading a script
  -e, --errexit          exit at the first command that fails
      --help             display this help and exit
      --version          output version information and exit

