.. _tailx:

tailx
-----

Print the last part of each file, optionally following it as it grows.

Synopsis
~~~~~~~~


tailx prints the last 10 lines of each FILE to standard output. With no FILE, or when FILE is -, it reads standard input. -n NUM changes the number of lines and -c NUM switches to bytes; a leading '+' on NUM starts output at line or byte NUM instead of counting from the end. NUM accepts a multiplier suffix: b is 512, K, M, G, T, P and E are powers of 1024 (also written KiB, MiB, ...), and KB, MB, ... are powers of 1000. With more than one FILE each is preceded by a '==> name <==' header; -q suppresses the headers and -v forces them. -f keeps running and prints data as it is appended, polling every -s seconds; --follow=name tracks the file name rather than the open file, so it survives log rotation, and --retry keeps waiting for a file that is missing. -F is shorthand for --follow=name --retry. --pid ends following when the given process exits. GNU tail, polling, checks whether a followed name has been rotated only after --max-unchanged-stats=N polls without growth (5 by default); tailx checks it on every poll, so it notices a rotation at once, and N is validated but changes nothing. On a regular file tailx seeks to the end and reads backward, so its cost does not depend on the file's size.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. tailx --help shows the options and the first examples; gnux help tailx prints the full manual, and gnux help tailx OPTION explains one option.

SEE ALSO. headx, catx, wcx.

PERMISSIONS. Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified. With --follow=name the file is reopened on every poll, so permissions are re-checked as it is rotated.

USE CASES.
  - Reading the most recent entries of a log.
  - Watching a log live with -f, or across rotations with -F.
  - Skipping a header with -n +2.
  - Following a build log only for as long as the build runs, with --pid.
  - Waiting for a log file that a service has not created yet, with --retry.
  - Printing only new log lines from now on with -n 0 -f.
  - Stripping a fixed-size prefix, such as a byte-order mark, with -c +NUM.
  - Extracting a range of lines when combined with headx.
  - Reading the last record of a NUL-separated list with -z.

::

  tailx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Print the last 10 lines of a file
    tailx app.log

    # Print the last 50 lines
    tailx -n 50 app.log

    # Print the last 3 lines, using the historical -NUM form
    tailx -3 app.log

    # Print everything from the second line on (skip a header)
    tailx -n +2 data.csv

    # Print the last 100 bytes of a file
    tailx -c 100 app.log

    # Drop a 3-byte UTF-8 byte-order mark by starting at byte 4
    tailx -c +4 with-bom.txt > clean.txt

    # Print the last line of each log, with and without file name headers
    tailx -n 1 a.log b.log && tailx -q -n 1 a.log b.log

    # Force the header for a single file (--silent is the long spelling of -q)
    tailx -v -n 5 app.log

    # Print lines 11 to 20 of a file
    headx -n 20 app.log | tailx -n 10

    # Print the oldest entry of a directory
    lsx -t | tailx -n 1

    # Print the last entry of a NUL-separated list
    tailx -z -n 1 names.lst | xargs -0 echox

    # Follow a log, printing only lines appended from now on
    tailx -n 0 -f app.log

    # Follow a log across rotations, polling twice a second
    tailx -F -s 0.5 /var/log/app.log

    # Follow by name and wait for a file that does not exist yet
    tailx --follow=name --retry service.log

    # Follow a log until a process exits
    make > build.log 2>&1 & tailx -f --pid=$! build.log

    # Report a missing file through the exit status
    tailx missing.log || echox "exit status $?"

    # Show the end of every log in the portable gnux shell
    gnux shell -c 'catx *.log | tailx -n 5'

Options
~~~~~~~

::

  -c, --bytes [+]NUM                output the last NUM bytes; or use -c +NUM to output starting with byte NUM of each file
  -f, --follow HOW[="descriptor"]   output appended data as the file grows; HOW is 'name' or 'descriptor' (the default)
  -F, --follow-retry                same as --follow=name --retry
      --help                        display this help and exit
  -n, --lines [+]NUM                output the last NUM lines, instead of the last 10; or use -n +NUM to output starting with line NUM
      --max-unchanged-stats N       with --follow=name, reopen a FILE which has not changed size after N (default 5) iterations to see if it has been unlinked or renamed (default "5")
      --pid PID                     with -f, terminate after process ID, PID, dies
  -q, --quiet                       never output headers giving file names
      --retry                       keep trying to open a file if it is inaccessible
      --silent                      same as --quiet
  -s, --sleep-interval N            with -f, sleep for approximately N seconds (default 1.0) between iterations (default 1)
  -v, --verbose                     always output headers giving file names
      --version                     output version information and exit
  -z, --zero-terminated             line delimiter is NUL, not newline

