.. _chmodx:

chmodx
------

Change file mode bits.

Synopsis
~~~~~~~~


chmodx changes the permission bits of each FILE to MODE, which is either an octal number (755, 0644, 4755) or a comma-separated list of symbolic changes. Each change names who (u, g, o or a for user, group, others or all), an operator (+ adds, - removes, = sets exactly) and permissions: r, w, x, X (execute only on directories and on files already executable by someone), s (setuid/setgid) and t (sticky), or u, g or o to copy another class's bits. A symbolic change without a who honors the umask. --reference=RFILE copies RFILE's mode instead, and then every operand is a FILE. -R changes directory trees; -H (the default) follows symbolic links given on the command line, -L follows every symbolic link and -P none, and when several are given the most cautious wins (-P, then -H, then -L). Symbolic links met while recursing are never changed. -c reports each change, -v every file, and -f suppresses most error messages. A mode may also be written as an option, as in 'chmodx -w FILE'; several such options combine as if joined with commas, and then, as in GNU, a file left with bits the mode would have cleared but for the umask is reported ('new permissions are ..., not ...') and counts as a failure. Unlike GNU chmod, which leaves the safeguard off unless --preserve-root is given, chmodx refuses to work recursively on '/' unless --no-preserve-root is given; when both are given, --preserve-root wins. GNU chmod 9.6's -h and --dereference are not implemented. On Windows only the owner's write bit maps onto the file system (the read-only attribute), so the other bits are accepted but have no effect.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. chmodx --help shows the options and the first examples; gnux help chmodx prints the full manual, and gnux help chmodx OPTION explains one option.

SEE ALSO. chownx, chgrpx, lsx, statx, installx.

PERMISSIONS. Changing a file's mode requires owning it, or running as root (or with CAP_FOWNER). Search permission is needed on the directories leading to each FILE, and with -R read and search permission on every directory in the tree.

USE CASES.
  - Making a script executable with +x.
  - Locking down a private key or config file with 600.
  - Giving a group write access to a shared directory tree with -R g+w.
  - Making directories, but not files, searchable with -R a+X.
  - Copying the permissions of one file onto others with --reference.

::

  chmodx [OPTION]... MODE[,MODE]... FILE...

Examples
~~~~~~~~

::

    # Make a script executable
    chmodx +x deploy.sh

    # Set exact permissions with an octal mode
    chmodx 600 ~/.ssh/id_ed25519

    # Remove write permission for group and others
    chmodx go-w shared.txt

    # Combine several changes
    chmodx u=rwx,g=rx,o= app

    # Give a group write access to a whole tree
    chmodx -R g+w project/

    # Make directories searchable without making files executable
    chmodx -R a+X public/

    # Report only the files that changed
    chmodx -c 644 *.txt

    # Copy the mode of one file onto another
    chmodx --reference=template.sh new.sh

    # Write the mode as an option, as old scripts do
    chmodx -w notes.txt

    # Set the sticky bit on a shared directory
    chmodx 1777 /srv/drop

    # Show every file processed
    chmodx -v -R u+rw docs/

    # Change modes in the portable gnux shell
    gnux shell -c 'chmodx +x bin/* && lsx -l bin'

Options
~~~~~~~

::

  -c, --changes            like verbose but report only when a change is made
  -L, --dereference-all    with -R, follow every symbolic link to a directory
  -H, --dereference-args   with -R, follow symbolic links given on the command line (the default)
      --help               display this help and exit
  -P, --no-dereference     with -R, do not follow any symbolic links
      --no-preserve-root   do not treat '/' specially
      --preserve-root      fail to operate recursively on '/' (the default)
      --quiet              same as --silent
  -R, --recursive          change files and directories recursively
      --reference RFILE    use RFILE's mode instead of specifying MODE values
  -f, --silent             suppress most error messages
  -v, --verbose            output a diagnostic for every file processed
      --version            output version information and exit

