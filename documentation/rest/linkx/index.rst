.. _linkx:

linkx
-----

Call the link function to create a hard link to a file.

Synopsis
~~~~~~~~


linkx creates FILE2 as a hard link to the existing FILE1: a second name for the same file, sharing its contents and metadata. It is the bare link(2) system call, with no options: it never overwrites FILE2, never makes a symbolic link, and never places the link inside a directory. Exactly two operands are required. lnx does the same with options; linkx is for scripts that want exactly one system call and its error.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. linkx --help shows the options and the first examples; gnux help linkx prints the full manual, and gnux help linkx OPTION explains one option.

SEE ALSO. lnx, unlinkx, lsx.

PERMISSIONS. Needs write and search permission on the directory that will hold FILE2. FILE1 and FILE2 must be on the same filesystem, and most systems refuse a hard link to a directory. On Windows, hard links need an NTFS volume.

USE CASES.
  - Giving a file a second name without copying its data.
  - Creating a hard link in a script that must fail rather than overwrite.
  - Keeping a snapshot name for a file that is about to be renamed or replaced.
  - Testing whether two paths are on the same filesystem (the link fails if not).
  - Implementing a simple lock: link succeeds for exactly one of several racing processes.

::

  linkx FILE1 FILE2

Examples
~~~~~~~~

::

    # Give a file a second name
    linkx report.txt report-final.txt

    # Refuse to overwrite an existing name (exit status 1)
    linkx a.txt existing.txt || echox 'already there'

    # Check that both names are the same file (same inode in the first column)
    linkx data.bin data.bak && lsx -i data.bin data.bak

    # Keep the old contents reachable before replacing a file
    linkx config.yaml config.yaml.old && cpx new.yaml config.yaml

    # Take a lock that only one process can get
    touchx lock.$$ && linkx lock.$$ lock && echox 'got the lock'

    # Release that lock
    unlinkx lock

    # Link a file into another directory (FILE2 names the link, not the directory)
    linkx build/app dist/app

    # Count a file's names after linking it
    linkx a b && lsx -l a

    # Link a file whose name starts with a dash
    linkx -- -file copy

    # Make a hard link in the portable gnux shell
    gnux shell -c 'linkx notes.md notes-link.md'

Options
~~~~~~~

::

      --help      display this help and exit
      --version   output version information and exit

