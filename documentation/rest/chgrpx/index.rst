.. _chgrpx:

chgrpx
------

Change group ownership.

Synopsis
~~~~~~~~


chgrpx changes the group of each FILE to GROUP, a group name or numeric ID. An empty GROUP changes nothing but still checks that each FILE exists. --reference=RFILE takes the group from RFILE instead, and then every operand is a FILE. A symbolic link given as a FILE is followed, and its referent changed, unless -h is given; when both -h and --dereference are given, -h wins. -R changes directory trees, following no symbolic link by default (-P); -H follows the ones given on the command line and -L every one, and when several are given the most cautious wins (-P, then -H, then -L). -c reports each change, -v every file, and -f suppresses most error messages. Unlike GNU chgrp, which leaves the safeguard off unless --preserve-root is given, chgrpx refuses to work recursively on '/' unless --no-preserve-root is given; when both are given, --preserve-root wins. On Windows, which has no Unix groups, every change fails with 'Operation not supported'.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. chgrpx --help shows the options and the first examples; gnux help chgrpx prints the full manual, and gnux help chgrpx OPTION explains one option.

SEE ALSO. chownx, chmodx, idx, lsx.

PERMISSIONS. The owner of a file may change its group to any group they belong to; any other change requires root (or CAP_CHOWN). Search permission is needed on the directories leading to each FILE, and with -R read and search permission on every directory in the tree.

USE CASES.
  - Sharing a project directory with a team's group.
  - Giving a web server's group read access to a site with -R.
  - Fixing group ownership after extracting an archive.
  - Copying the group of one file onto others with --reference.
  - Checking which files would change with -c on a test run.

::

  chgrpx [OPTION]... GROUP FILE...

Examples
~~~~~~~~

::

    # Change the group of a file
    chgrpx staff report.txt

    # Use a numeric group ID
    chgrpx 1001 report.txt

    # Change a whole directory tree
    chgrpx -R developers project/

    # Follow symbolic links met while recursing
    chgrpx -R -L developers project/

    # Change a symbolic link itself
    chgrpx -h staff link

    # Copy the group of another file
    chgrpx --reference=template.conf new.conf

    # Report only the files that changed
    chgrpx -c -R staff shared/

    # Report every file processed
    chgrpx -v staff a.txt b.txt

    # Suppress most error messages
    chgrpx -f staff maybe-missing.txt

    # Give a web server's group the site files
    sudo chgrpx -R www-data /var/www/site

    # Change groups in the portable gnux shell
    gnux shell -c 'chgrpx -R staff shared && lsx -l shared'

Options
~~~~~~~

::

  -c, --changes            like verbose but report only when a change is made
      --dereference        affect the referent of each symbolic link (this is the default), rather than the symbolic link itself
  -L, --dereference-all    with -R, traverse every symbolic link to a directory encountered
  -H, --dereference-args   with -R, follow symbolic links to directories given on the command line
      --help               display this help and exit
  -h, --no-dereference     affect symbolic links instead of any referenced file
      --no-preserve-root   do not treat '/' specially
  -P, --physical           with -R, do not traverse any symbolic links (default)
      --preserve-root      fail to operate recursively on '/' (the default)
      --quiet              same as --silent
  -R, --recursive          operate on files and directories recursively
      --reference RFILE    use RFILE's group rather than specifying a GROUP value
  -f, --silent             suppress most error messages
  -v, --verbose            output a diagnostic for every file processed
      --version            output version information and exit

