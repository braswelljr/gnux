.. _hostnamex:

hostnamex
---------

Print the name of the current host system.

Synopsis
~~~~~~~~


hostnamex prints the name of the current host, as the operating system reports it. GNU coreutils hostname also sets the name when given NAME (with sethostname(2), as root); gnux never changes system settings, so hostnamex refuses NAME with 'cannot set name to NAME: Operation not supported' and exits 1. Use the system's own tool (hostnamectl, scutil, Rename-Computer) to set it. Note that the hostname program shipped by many Linux distributions is not the coreutils one and has options such as -f and -I; hostnamex has none of them.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. hostnamex --help shows the options and the first examples; gnux help hostnamex prints the full manual, and gnux help hostnamex OPTION explains one option.

SEE ALSO. unamex, whoamix.

PERMISSIONS. Needs no permissions to print the name.

USE CASES.
  - Recording which machine produced a log line or a build artifact.
  - Naming per-host files or directories.
  - Branching a script on the machine it runs on.
  - Checking a container's or a CI runner's host name.
  - Getting the host name the same way on Linux, macOS and Windows.

::

  hostnamex [NAME]

Examples
~~~~~~~~

::

    # Print the host name
    hostnamex

    # Tag a log line with the host
    echox "$(hostnamex): backup done" >> backup.log

    # Name an archive after the machine
    cpx backup.tar "backup-$(hostnamex).tar"

    # Run a step only on one machine
    [ "$(hostnamex)" = build-01 ] && make release

    # Print the user and the host together
    echox "$(whoamix)@$(hostnamex)"

    # Compare with the network node name unamex reports
    hostnamex; unamex -n

    # Setting the name is refused (exit status 1)
    hostnamex new-name || echox 'use hostnamectl instead'

    # Print the host name in upper case
    hostnamex | trx a-z A-Z

    # Create a per-host directory
    mkdirx -p "logs/$(hostnamex)"

    # Print the host name in the portable gnux shell
    gnux shell -c 'hostnamex'

Options
~~~~~~~

::

      --help      display this help and exit
      --version   output version information and exit

