.. _mvx:

mvx
---

Move (rename) files and directories.

Synopsis
~~~~~~~~


mvx renames SOURCE to DEST, or moves several SOURCEs into DIRECTORY (also selectable with -t; -T forces the two-operand form). Within one filesystem a move is an atomic rename. Across filesystems mvx copies the source with all of its attributes, symbolic links and hard links intact and removes the original only after the copy has fully succeeded; --no-copy refuses such a move instead ('Invalid cross-device link'), so a move is always a rename. An existing destination is replaced unless -n (never), -u (only when the source is newer) or -i (ask first) says otherwise; a directory is never moved into itself, and a directory only replaces an empty directory. --backup[=CONTROL] (or -b) first renames each existing destination to a backup: CONTROL simple (or never) appends a suffix, numbered (t) appends .~N~ with N one past the highest already there, existing (nil, the default) numbers the backup only when numbered backups already exist, and none (off) makes none; without a CONTROL the VERSION_CONTROL variable chooses. The suffix is ~, or SUFFIX with -S (which asks for backups by itself), or the SIMPLE_BACKUP_SUFFIX variable. A destination directory is backed up too, so with a backup a directory may replace a file and a file a directory; --backup cannot be combined with -n. GNU mv lets the last of -f, -i and -n win; mvx cannot see option order, so it prefers the most cautious instead: -n over -i over -f.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. mvx --help shows the options and the first examples; gnux help mvx prints the full manual, and gnux help mvx OPTION explains one option.

SEE ALSO. cpx, rmx, lnx.

PERMISSIONS. Renaming needs write and search permission on both the source and destination directories, not on the file itself; in a sticky directory you must also own the entry. On a terminal, replacing a write-protected destination asks for confirmation unless -f is given. A cross-filesystem move additionally needs read permission on everything being moved, and preserves ownership only when privileged.

USE CASES.
  - Renaming a file or directory in place.
  - Sorting files into a directory, including with find/xargs via -t.
  - Relocating a tree to another disk without losing modes, times or links.
  - Protecting existing files in scripts with -n, or reviewing overwrites with -i.
  - Promoting a staged file over the live one only when it is newer (-u).
  - Publishing a file atomically, never by a slow copy, with --no-copy.
  - Atomically swapping a freshly written file into place after it is complete.
  - Replacing a directory's name with -T instead of moving into it.
  - Logging every move in a cleanup script with -v.

::

  mvx [OPTION]... SOURCE... DIRECTORY

Examples
~~~~~~~~

::

    # Rename a file
    mvx draft.md final.md

    # Move several files into a directory, reporting each one
    mvx -v a.log b.log archive/

    # Move without ever overwriting an existing file
    mvx -n photo.jpg ~/Pictures/

    # Name the destination directory first (handy with xargs)
    mvx -t build/out lib.a lib.so

    # Ask before replacing an existing file
    mvx -i new-config.yml config.yml

    # Replace a write-protected destination without being asked
    mvx -f settings.tmp settings.json

    # Replace the destination only when the source is newer
    mvx -u staging/index.html public/index.html

    # Skip files that already exist at the destination, using --update
    mvx --update=none -v incoming/*.csv data/

    # Rename a directory over an existing empty one instead of moving into it
    mvx -T build-new build

    # Move a directory given with a trailing slash
    mvx --strip-trailing-slashes logs/ old-logs

    # Write to a temporary file, then move it into place only on success
    echox "generated" > report.tmp && mvx report.tmp report.txt

    # Move every log file into an archive directory found with find
    find . -maxdepth 1 -name '*.log' -print0 | xargs -0 mvx -t archive/

    # Report an error when the move fails
    mvx missing.txt dest/ || echox "move failed with status $?"

    # Archive text files into a dated directory with the gnux shell
    gnux shell -c 'mkdirx -p archive/2024 && mvx -v *.txt archive/2024/'

    # Publish a finished build atomically, or fail if it is on another disk
    mvx --no-copy build/site.new www/site

    # Replace a log with a new one, keeping the old one as server.log.old
    mvx -S .old server.log.new server.log

Options
~~~~~~~

::

      --backup CONTROL[="$VERSION_CONTROL"]   make a backup of each existing destination file; CONTROL is none (off), simple (never), existing (nil) or numbered (t)
  -b, --backup-default                        like --backup but does not accept an argument
  -f, --force                                 do not prompt before overwriting
      --help                                  display this help and exit
  -i, --interactive                           prompt before overwrite
  -n, --no-clobber                            do not overwrite an existing file
      --no-copy                               do not copy if renaming fails
  -T, --no-target-directory                   treat DEST as a normal file
      --strip-trailing-slashes                remove any trailing slashes from each SOURCE argument
  -S, --suffix SUFFIX                         override the usual backup suffix, '~', with SUFFIX
  -t, --target-directory string               move all SOURCE arguments into DIRECTORY
  -u, --update string[="older"]               control which existing files are updated: all, none, or older (the default for -u)
  -v, --verbose                               explain what is being done
      --version                               output version information and exit

