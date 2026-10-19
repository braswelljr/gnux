.. _lognamex:

lognamex
--------

Print the user's login name.

Synopsis
~~~~~~~~


lognamex prints the name the user logged in as, which stays the same after su or sudo, where whoamix prints the user the process now runs as. It takes no operands. When there is no login name, as in a cron job, a container or a service, it reports 'no login name' and exits 1. GNU logname asks the C library's getlogin; gnux has no cgo and does what glibc does instead: on Linux it reads the login user the kernel records for the process (/proc/self/loginuid), and failing that, on every Unix, the login record of the terminal on standard input. macOS and the BSDs also keep the name in the kernel session, which only the C library can read, so there a process with no terminal on standard input has no login name for lognamex even when GNU logname would print one. On Windows it prints the user the session was signed in as, without the domain.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. lognamex --help shows the options and the first examples; gnux help lognamex prints the full manual, and gnux help lognamex OPTION explains one option.

SEE ALSO. whoamix, usersx, whox, idx.

PERMISSIONS. Needs no permissions. Reads the process's login user and the login records.

USE CASES.
  - Finding out who really logged in inside a sudo or su session.
  - Addressing the human user in a script that runs as root.
  - Detecting that a script runs without a login (cron, a service) by its exit status.
  - Recording who started a deployment in its log.
  - Choosing a per-user directory in an administration script.

::

  lognamex [OPTION]...

Examples
~~~~~~~~

::

    # Print your login name
    lognamex

    # See the difference from whoamix under sudo
    sudo sh -c 'lognamex; whoamix'

    # Record who ran a deployment
    echox "deployed by $(lognamex)" >> deploy.log

    # Fall back to the effective user when there is no login
    name=$(lognamex 2>/dev/null || whoamix)

    # Detect a run without a login
    lognamex >/dev/null 2>&1 || echox 'no login session'

    # Use the login name in a path
    mkdirx -p "/srv/scratch/$(lognamex)"

    # Save the login name in a variable
    me=$(lognamex)

    # Greet the user
    printfx 'hello, %s\n' "$(lognamex)"

    # List the other sessions of the same user
    whox | grepx "^$(lognamex) "

    # Print the login name in the portable gnux shell
    gnux shell -c 'lognamex'

Options
~~~~~~~

::

      --help      display this help and exit
      --version   output version information and exit

