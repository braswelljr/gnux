.. _pwdx:

pwdx
----

Print the full name of the current working directory.

Synopsis
~~~~~~~~


pwdx prints the absolute path of the current working directory. With -P (the default) every symbolic link in the path is resolved; with -L it prints the value of the PWD environment variable when that names the current directory, preserving the symlinks the shell walked through. If both are given, the last one wins in GNU pwd; pwdx prefers -L.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. pwdx --help shows the options and the first examples; gnux help pwdx prints the full manual, and gnux help pwdx OPTION explains one option.

SEE ALSO. realpathx, dirnamex, gnux shell.

PERMISSIONS. Needs search permission on every directory leading to the current one to resolve it physically.

USE CASES.
  - Capturing the working directory in a script.
  - Finding the real location behind a symlinked directory with -P.
  - Keeping the symlinked path the user navigated through with -L, for messages or relative links.
  - Detecting whether the current directory is reached through a symlink by comparing -L and -P.
  - Deriving the project name or parent directory with basenamex and dirnamex.
  - Checking where a cd or envx -C actually landed in a script.

::

  pwdx [OPTION]...

Examples
~~~~~~~~

::

    # Print the physical working directory
    pwdx

    # Keep symlinks as the shell sees them
    pwdx -L

    # Resolve every symlink explicitly
    cd link && pwdx -P

    # Save the directory in a variable
    here=$(pwdx); echox "working in $here"

    # Print only the current directory's name
    basenamex "$(pwdx)"

    # Print the parent directory
    dirnamex "$(pwdx)"

    # Tell whether the current directory is reached through a symlink
    [ "$(pwdx -L)" != "$(pwdx -P)" ] && echox 'inside a symlinked directory'

    # Print the directory a command would run in
    envx -C build pwdx

    # Show where cd leads inside the gnux shell
    gnux shell -c 'cd docs; pwdx'

Options
~~~~~~~

::

      --help       display this help and exit
  -L, --logical    use PWD from the environment, even if it contains symlinks
  -P, --physical   resolve all symlinks (default)
      --version    output version information and exit

