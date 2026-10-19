.. _datex:

datex
-----

Print the date and time.

Synopsis
~~~~~~~~


datex prints the current date and time, or the one -d STRING describes, in the C locale's default format ('Mon Jan  2 15:04:05 UTC 2006') or in +FORMAT. FORMAT takes GNU's conversions: %Y %m %d %H %M %S and friends, %a %A %b %B %p (in English), %s (seconds since the epoch), %N (nanoseconds), %z %:z %::z %:::z and %Z (zones), %j %U %W %V %G %u %w (day and week numbers), %F %T %D %R %r %c %x %X %h %e %k %l %C %y %g %n %t %%, and the flags -, _, 0, ^ and # with a field width (%-d, %_H, %10s). -I[FMT] prints ISO 8601 (date, hours, minutes, seconds or ns; date by default), --rfc-3339=FMT RFC 3339 (date, seconds or ns) and -R RFC 5322; only one output format may be given. -u uses UTC, and otherwise the TZ environment variable or the local zone applies. -d STRING accepts the forms GNU date accepts most often: 'now', 'today', 'yesterday', 'tomorrow', '@SECONDS', ISO 8601 and RFC 3339 dates and times, MM/DD/YYYY, 'Jan 2 2006', '2 Jan 2006', times of day (14:30, 2:30pm, noon, midnight) with UTC, Z or a numeric offset, weekdays ('next friday', 'last monday') and relative items ('3 days ago', '+2 hours', 'next month', 'fortnight'), combined as in '2026-01-31 +1 month' or 'tomorrow 09:00'. Anything else is reported as an invalid date, where GNU's grammar is larger still. -f FILE prints one date per line of FILE (- for standard input) and -r FILE the last modification time of FILE. gnux never changes system settings, so -s and a MMDDhhmm[[CC]YY][.ss] operand fail with 'cannot set date: Operation not supported'. --resolution prints the resolution of the system clock, measured as GNU measures it, as a time that many seconds after the epoch (0.000000001 for a nanosecond clock), and in any format '%-N' prints as many digits of nanoseconds as that resolution needs. --debug is not implemented.

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

    # Check the date and time on a server
    datex

    # Get today's date in a sortable form for a file name
    datex -I

    # Stamp a log line with a precise, sortable time
    datex -Iseconds

    # Check UTC when coordinating with people in other time zones
    datex -u

    # Print the time the way your report needs it
    datex '+%Y-%m-%d %H:%M:%S'

    # Name tonight's database backup after today's date
    cpx db.sqlite "db-$(datex +%F).sqlite"

    # Get a Unix timestamp to store or compare
    datex +%s

    # Read a Unix timestamp you found in a log or a database
    datex -u -d @1700000000

    # Find the date three days ago, to clean up older files
    datex -d '3 days ago' +%F

    # Find the date of next Friday for a deadline
    datex -d 'next friday' +%A,%F

    # See when a report was last saved
    datex -r report.txt

    # See what time it is in Tokyo before calling a colleague
    TZ=Asia/Tokyo datex

    # Produce a timestamp an API expects in RFC 3339
    datex --rfc-3339=seconds

    # Convert a column of dates into timestamps for sorting
    datex -f dates.txt +%s

    # Timestamp a log line in the portable gnux shell, the same on every OS
    gnux shell -c 'datex -Iseconds >> run.log'

    # Check how finely this system's clock measures time
    datex --resolution

    # Timestamp a log line to the clock's own precision
    datex '+%H:%M:%S.%-N build started'

Options
~~~~~~~

::

  -d, --date STRING             display time described by STRING, not 'now'
  -f, --file DATEFILE           like --date; once for each line of DATEFILE
      --help                    display this help and exit
  -I, --iso-8601 FMT[="date"]   output date/time in ISO 8601 format. FMT='date' for date only (the default), 'hours', 'minutes', 'seconds', or 'ns'
  -r, --reference FILE          display the last modification time of FILE
      --resolution              output the available resolution of timestamps
      --rfc-3339 FMT            output date/time in RFC 3339 format. FMT='date', 'seconds', or 'ns'
  -R, --rfc-email               output date and time in RFC 5322 format
  -s, --set STRING              set time described by STRING (not supported)
      --universal               same as --utc
  -u, --utc                     print or set Coordinated Universal Time (UTC)
      --version                 output version information and exit

