.. _datex:

datex
-----

Print the date and time.

Synopsis
~~~~~~~~


datex prints the current date and time, or the one -d STRING describes, in the C locale's default format ('Mon Jan  2 15:04:05 UTC 2006') or in +FORMAT. FORMAT takes GNU's conversions: %Y %m %d %H %M %S and friends, %a %A %b %B %p (in English), %s (seconds since the epoch), %N (nanoseconds), %z %:z %::z %:::z and %Z (zones), %j %U %W %V %G %u %w (day and week numbers), %F %T %D %R %r %c %x %X %h %e %k %l %C %y %g %n %t %%, and the flags -, _, 0, ^ and # with a field width (%-d, %_H, %10s). -I[FMT] prints ISO 8601 (date, hours, minutes, seconds or ns; date by default), --rfc-3339=FMT RFC 3339 (date, seconds or ns) and -R RFC 5322; only one output format may be given. -u uses UTC, and otherwise the TZ environment variable or the local zone applies. -d STRING accepts the forms GNU date accepts most often: 'now', 'today', 'yesterday', 'tomorrow', '@SECONDS', ISO 8601 and RFC 3339 dates and times, MM/DD/YYYY, 'Jan 2 2006', '2 Jan 2006', times of day (14:30, 2:30pm, noon, midnight) with UTC, Z or a numeric offset, weekdays ('next friday', 'last monday') and relative items ('3 days ago', '+2 hours', 'next month', 'fortnight'), combined as in '2026-01-31 +1 month' or 'tomorrow 09:00'. Anything else is reported as an invalid date, where GNU's grammar is larger still. -f FILE prints one date per line of FILE (- for standard input) and -r FILE the last modification time of FILE. gnux never changes system settings, so -s and a MMDDhhmm[[CC]YY][.ss] operand fail with 'cannot set date: Operation not supported'. --debug and --resolution are not implemented.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. datex --help shows the options and the first examples; gnux help datex prints the full manual, and gnux help datex OPTION explains one option.

SEE ALSO. touchx, sleepx, lsx, uptimex.

PERMISSIONS. Printing needs no permissions; -f needs read permission on FILE and -r search permission on the directories leading to it.

USE CASES.
  - Timestamping file names, log lines and backups.
  - Converting between epoch seconds and readable dates.
  - Computing dates relative to now or to another date, such as 'next friday'.
  - Printing machine-readable ISO 8601 or RFC 3339 timestamps.
  - Showing the current time in another zone with TZ or -u.

::

  datex [OPTION]... [+FORMAT]

Examples
~~~~~~~~

::

    # Print the current date and time
    datex

    # Print the date in ISO 8601
    datex -I

    # Print an ISO 8601 timestamp to the second
    datex -Iseconds

    # Print the time in UTC
    datex -u

    # Use a custom format
    datex '+%Y-%m-%d %H:%M:%S'

    # Name a backup after today's date
    cpx db.sqlite "db-$(datex +%F).sqlite"

    # Print the seconds since the epoch
    datex +%s

    # Convert epoch seconds to a date
    datex -u -d @1700000000

    # Compute a relative date
    datex -d '3 days ago' +%F

    # Find next Friday's date
    datex -d 'next friday' +%A,%F

    # Print a file's modification time
    datex -r report.txt

    # Print the time in another zone
    TZ=Asia/Tokyo datex

    # Print an RFC 3339 timestamp
    datex --rfc-3339=seconds

    # Convert a list of dates
    datex -f dates.txt +%s

    # Timestamp a log line in the portable gnux shell
    gnux shell -c 'datex -Iseconds >> run.log'

Options
~~~~~~~

::

  -d, --date STRING             display time described by STRING, not 'now'
  -f, --file DATEFILE           like --date; once for each line of DATEFILE
      --help                    display this help and exit
  -I, --iso-8601 FMT[="date"]   output date/time in ISO 8601 format. FMT='date' for date only (the default), 'hours', 'minutes', 'seconds', or 'ns'
  -r, --reference FILE          display the last modification time of FILE
      --rfc-3339 FMT            output date/time in RFC 3339 format. FMT='date', 'seconds', or 'ns'
  -R, --rfc-email               output date and time in RFC 5322 format
  -s, --set STRING              set time described by STRING (not supported)
      --universal               same as --utc
  -u, --utc                     print or set Coordinated Universal Time (UTC)
      --version                 output version information and exit

