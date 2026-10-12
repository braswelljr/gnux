.. _timeoutx:

timeoutx
--------

Run a command with a time limit.

Synopsis
~~~~~~~~


timeoutx runs COMMAND and, if it is still running after DURATION, sends it the TERM signal, or the signal -s names. DURATION is a number, possibly decimal or hexadecimal, with an optional unit: s (seconds, the default), m (minutes), h (hours) or d (days); 0 means no limit. -k DURATION also sends KILL if COMMAND is still running that long after the first signal. By default the signal goes to COMMAND's whole process group, so the processes it started stop too; --foreground signals COMMAND alone and leaves it in the terminal's process group, so it can read from the terminal. A signal sent to timeoutx is passed on to COMMAND, and -v reports each signal sent. The exit status is 124 when COMMAND timed out (unless --preserve-status, which keeps COMMAND's own status), 125 when timeoutx itself failed, 126 when COMMAND was found but could not be run, 127 when it was not found, and otherwise COMMAND's status (128+N when signal N ended it, 137 after KILL). Options must come before DURATION. On Windows, which has no signals, the limit still works: TERM and KILL both end COMMAND, so timeoutx is a portable way to bound a step in a script on every platform.

EXIT STATUS.
  124  COMMAND timed out (and --preserve-status was not given)
  125  timeoutx itself failed
  126  COMMAND was found but could not be run
  127  COMMAND was not found
  137  COMMAND (or timeoutx) was sent the KILL signal
  other  the exit status of COMMAND

HELP. timeoutx --help shows the options and the first examples; gnux help timeoutx prints the full manual, and gnux help timeoutx OPTION explains one option.

SEE ALSO. killx, sleepx, nicex, nohupx.

PERMISSIONS. Needs execute permission on COMMAND; signaling it needs no more, since timeoutx started it.

USE CASES.
  - Stopping a CI step or test suite that hangs instead of letting it run all night.
  - Bounding a network call such as curl or ssh that may never answer.
  - Giving a service a grace period to shut down, then killing it with -k.
  - Polling for a condition for at most a minute in a script.
  - Limiting a job the same way on Linux, macOS and Windows.

::

  timeoutx [OPTION] DURATION COMMAND [ARG]...

Examples
~~~~~~~~

::

    # Give a test suite 10 minutes before stopping it
    timeoutx 10m go test ./...

    # Stop waiting on a slow server after 5 seconds
    timeoutx 5 curl -s https://example.com/health

    # Tell a timeout apart from a failure
    timeoutx 30s ./migrate.sh; [ $? -eq 124 ] && echox 'migration timed out'

    # Send INT, like Ctrl-C, instead of TERM
    timeoutx -s INT 1h ./crawler

    # Kill a server that ignores TERM 10 seconds after asking it to stop
    timeoutx -k 10s 1h ./server

    # Report the signals as they are sent
    timeoutx -v 2s sleepx 60

    # Keep the command's own exit status after a timeout
    timeoutx --preserve-status 1m ./bench.sh

    # Let an interactive command keep reading from the terminal
    timeoutx --foreground 5m ssh build-host

    # Wait at most a minute for a file to appear
    timeoutx 1m sh -c 'until [ -e ready.flag ]; do sleepx 1; done'

    # Use a fractional limit
    timeoutx 0.5 ./probe

    # Bound a step in the portable gnux shell (external programs run through envx)
    gnux shell -c 'timeoutx 30s envx make test || echox "tests failed or timed out"'

Options
~~~~~~~

::

      --foreground            when not running timeout directly from a shell prompt, allow COMMAND to read from the TTY and get TTY signals; in this mode, children of COMMAND will not be timed out
      --help                  display this help and exit
  -k, --kill-after DURATION   also send a KILL signal if COMMAND is still running this long after the initial signal was sent (DURATION)
      --preserve-status       exit with the same status as COMMAND, even when the command times out
  -s, --signal SIGNAL         specify the SIGNAL to be sent on timeout; SIGNAL may be a name like 'HUP' or a number
  -v, --verbose               diagnose to stderr any signal sent upon timeout
      --version               output version information and exit

