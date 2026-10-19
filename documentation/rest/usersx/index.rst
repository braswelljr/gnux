.. _usersx:

usersx
------

Print the user names of users currently logged in to the current host.

Synopsis
~~~~~~~~


usersx prints the login name of each session on this machine, sorted and separated by spaces on one line; a user logged in more than once appears once per session. With nobody logged in it prints nothing. It reads the same records as whox: the utmp file on Linux and the BSDs (sessions whose process has ended are left out) and utmpx on macOS. FILE reads another file in that format instead, such as /var/log/wtmp, without leaving any session out. On Windows the sessions are the signed-in console and Remote Desktop sessions, and FILE is not supported; in the browser playground there are no records.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. usersx --help shows the options and the first examples; gnux help usersx prints the full manual, and gnux help usersx OPTION explains one option.

SEE ALSO. whox, whoamix, lognamex.

PERMISSIONS. Needs read permission on the login records (normally readable by everyone) or on FILE. Writes nothing.

USE CASES.
  - Seeing who else is logged in before rebooting a shared machine.
  - Counting the sessions on a server.
  - Checking whether a particular user is logged in.
  - Recording who was logged in when a job ran.
  - Listing the users recorded in an old wtmp file.

::

  usersx [OPTION]... [FILE]

Examples
~~~~~~~~

::

    # List the users logged in
    usersx

    # Count the sessions
    usersx | wcx -w

    # List each user once
    usersx | trx ' ' '\n' | uniqx

    # Check whether a user is logged in
    usersx | grepx -qw alice && echox 'alice is on'

    # Count the sessions of each user
    usersx | trx ' ' '\n' | uniqx -c

    # Read the users from a wtmp file
    usersx /var/log/wtmp

    # See the same users with their terminals and times
    whox

    # Warn before a reboot when anyone else is on
    [ "$(usersx | trx ' ' '\n' | grepx -vx "$(whoamix)")" ] && echox 'others are logged in'

    # Log who was on when a job ran
    echox "$(datex): $(usersx)" >> jobs.log

    # List the users in the portable gnux shell
    gnux shell -c 'usersx'

Options
~~~~~~~

::

      --help      display this help and exit
      --version   output version information and exit

