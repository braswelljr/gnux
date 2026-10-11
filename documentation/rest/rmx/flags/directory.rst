.. _dir:

dir
---

Remove empty directories (-d/--dir).

Synopsis
~~~~~~~~


-d/--dir removes directories that are empty, the way rmdir does. It is a safe middle ground between plain rmx (which refuses directories entirely) and -r (which deletes contents too): a non-empty directory is reported as an error and left untouched.

PERMISSIONS. Removing an empty directory requires write+execute permission on its parent. On a terminal, a write-protected directory prompts before removal unless -f is given. A non-empty directory reports 'Directory not empty' rather than a permission error.

USE CASES.
  - Pruning a single empty directory without risking its (future) contents.
  - Removing leftover scaffold directories after their files were deleted.
  - A safer alternative to -r when you expect the directory to be empty.
  - Removing files and empty directories together in one command.

::

  dir [DIR]...

Examples
~~~~~~~~

::

    # Remove a directory only if it contains no entries
    rmx -d generated/

    # Remove files and empty directories in one command
    rmx -dv stale.txt empty-dir/

    # A non-empty directory is reported and left alone
    rmx -d src/ || echox "src still has files"

    # Clear out a directory's files, then the directory itself
    rmx -f tmp/*.part && rmx -d tmp/

    # Prune empty directories with the gnux shell
    gnux shell -c 'rmx -dv out/a out/b; rmx -d out'

Options
~~~~~~~

::

  -d, --dir    remove empty directories
  -h, --help   help for dir

