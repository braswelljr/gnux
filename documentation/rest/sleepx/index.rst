.. _sleepx:

sleepx
------

Delay for a specified amount of time.

Synopsis
~~~~~~~~


sleepx pauses for NUMBER seconds. SUFFIX may be 's' for seconds (the default), 'm' for minutes, 'h' for hours or 'd' for days. NUMBER need not be an integer: fractions (0.5), exponents (1e3) and hexadecimal floating-point values are accepted, as is 'infinity' (or 'inf') to pause until interrupted. Given two or more arguments, sleepx pauses for the sum of their values. Every operand is checked before the pause begins, so an invalid interval fails immediately.

EXIT STATUS.
  0  the full interval elapsed
  1  an invalid interval or a command-line mistake

HELP. sleepx --help shows the options and the first examples; gnux help sleepx prints the full manual, and gnux help sleepx OPTION explains one option.

SEE ALSO. envx, yesx, gnux shell.

PERMISSIONS. Needs no permissions; it touches nothing.

USE CASES.
  - Pacing a retry or polling loop in a script.
  - Giving a service time to start before the next step runs.
  - Writing delays that behave the same in every shell, fractional seconds included.
  - Rate-limiting a loop that calls an API or writes to a slow device.
  - Waiting for a file or flag to appear before continuing.
  - Keeping a container or background job alive with sleepx infinity.
  - Spacing out progress messages or demo output so it can be read.
  - Simulating a slow command when testing timeouts, spinners or pipelines.

::

  sleepx NUMBER[SUFFIX]...

Examples
~~~~~~~~

::

    # Pause for half a second
    sleepx 0.5

    # Pause for one and a half minutes
    sleepx 1m 30s

    # Pause for two hours
    sleepx 2h

    # Use a fraction of a larger unit (6 minutes)
    sleepx 0.1h

    # Write the interval in scientific or hexadecimal notation
    sleepx 1e-1 0x0.1p0

    # Run the next command only after the pause
    sleepx 2 && echox ready

    # Print a line every second, three times
    for i in 1 2 3; do echox "tick $i"; sleepx 1; done

    # Retry a command every two seconds until it succeeds
    until curl -fs localhost:8080/health; do sleepx 2; done

    # Wait until a file appears
    until [ -e ready.flag ]; do sleepx 0.5; done; echox found

    # Run a slow stand-in command in the background
    sleepx 30 & echox "started $!"

    # See an invalid interval rejected before any pause
    sleepx 10x; echox "status: $?"

    # Space out the steps of a gnux shell script
    gnux shell -c 'echox start; sleepx 1; echox done'

    # Block until interrupted
    sleepx infinity

Options
~~~~~~~

::

      --help      display this help and exit
      --version   output version information and exit

