.. _statx:

statx
-----

Display file or file system status.

Synopsis
~~~~~~~~


statx prints everything the system records about each FILE: its type, size, blocks, device, inode, links, permissions, owner and group, and its access, modification, change and birth times. A symbolic link is described itself unless -L follows it. -f describes the file system holding FILE instead: its type, block sizes, and total, free and available blocks and inodes. -t prints the same fields on one line, for scripts. -c FORMAT prints only what FORMAT asks for, followed by a newline; --printf FORMAT does the same but interprets backslash escapes (\n, \t, \x41, \101) and adds no newline; when both are given, the last wins. FORMAT takes GNU's directives: %n (name), %N (quoted name, with its target for a link), %s (size), %F (file type), %a and %A (permissions in octal and as rwx), %u %U %g %G (owner and group), %h (links), %i (inode), %b %B %o (blocks, block unit, I/O size), %d %D %Hd %Ld (device), %r %R %t %T %Hr %Lr (device type of a special file), %m (mount point), %w %x %y %z (birth, access, modification and change times) and %W %X %Y %Z (the same as seconds since the epoch, which take a precision: %.3Y), with printf flags and widths (%-10s, %04a). With -f: %n %i %l %t %T %s %S %b %f %a %c %d. %N quotes a name (and a link's target) in the style the QUOTING_STYLE variable names, as lsx --quoting-style takes it, or else for the shell, always ('a.txt', 'a b', "it's", 'a'$'\n''b'); the default formats name files as they are. The device is shown as major,minor, as GNU stat 9 shows it. Missing data is reported honestly rather than invented: the birth time prints '-' where the system does not record it (Linux without statx), and on Windows, which has no numeric owners, %u and %g print '?'. SELinux contexts (%C) are not supported. --cached is accepted for compatibility and has no effect.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. statx --help shows the options and the first examples; gnux help statx prints the full manual, and gnux help statx OPTION explains one option.

SEE ALSO. lsx, dfx, dux, chmodx, touchx.

PERMISSIONS. Needs search permission on the directories leading to each FILE; the FILE itself need not be readable. Owner and group names are read from the system's account databases.

USE CASES.
  - Finding out exactly when a file was last modified, accessed or created.
  - Getting a file's size or permissions in a script without parsing lsx -l.
  - Checking which file system, and how much free space, a path is on.
  - Comparing inodes and devices to tell whether two paths are the same file.
  - Auditing ownership and permissions of sensitive files.

::

  statx [OPTION]... FILE...

Examples
~~~~~~~~

::

    # See everything about a config file before editing it
    statx /etc/hosts

    # Get a file's size in bytes in a script, without parsing lsx -l
    size=$(statx -c %s backup.tar.gz)

    # Check that an SSH key is private before ssh refuses to use it
    statx -c '%a %U %n' ~/.ssh/id_ed25519

    # Find out when a log file last changed, as seconds since the epoch, to alert if it went quiet
    [ $(( $(datex +%s) - $(statx -c %Y app.log) )) -gt 600 ] && echox 'app.log silent for 10 minutes'

    # List several files with their permissions and owners, aligned
    statx -c '%-25n %A %U:%G' /etc/passwd /etc/shadow /etc/sudoers

    # Tell whether two paths are the same file (same device and inode)
    [ "$(statx -L -c %d:%i a)" = "$(statx -L -c %d:%i b)" ] && echox 'same file'

    # See where a symbolic link points
    statx -c %N /usr/bin/python3

    # Check the file system type and free space under a directory
    statx -f /var/lib/docker

    # Print modification times with millisecond precision, tab-separated
    statx --printf '%n\t%.3Y\n' *.csv

    # Find a file's mount point
    statx -c %m ~/Downloads/movie.mkv

    # Record a file's full status on one line, for a log or a diff
    statx -t release.zip >> inventory.txt

    # Show file details in the portable gnux shell, the same on every OS
    gnux shell -c 'statx -c "%n: %s bytes, modified %y" report.pdf'

Options
~~~~~~~

::

      --cached string   specify how to use cached attributes (always, never or default); accepted, no effect
  -L, --dereference     follow links
  -f, --file-system     display file system status instead of file status
  -c, --format FORMAT   use the specified FORMAT instead of the default; output a newline after each use of FORMAT
      --help            display this help and exit
      --printf FORMAT   like --format, but interpret backslash escapes, and do not output a mandatory trailing newline
  -t, --terse           print the information in terse form
      --version         output version information and exit

