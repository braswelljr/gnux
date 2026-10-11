.. _rmx:

rmx
---

Remove (unlink) files and directories.

Synopsis
~~~~~~~~


rmx removes each FILE. By default it does not remove directories; use -r to remove a directory and everything under it, or -d to remove an empty directory. Prompting is controlled by -i (before every removal), -I (once before a bulk or recursive removal) and --interactive; -f disables all prompts and ignores missing files. rmx refuses to remove '.' and '..' and, by default, refuses to recurse on the filesystem root. Independent arguments are removed concurrently whenever no prompt or ordered output is involved.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. rmx --help shows the options and the first examples; gnux help rmx prints the full manual, and gnux help rmx OPTION explains one option.

SEE ALSO. rmdirx, mvx, cpx, gnux shell.

PERMISSIONS. Removing an entry needs write and search permission on the directory that contains it, not on the entry itself. On a terminal, a write-protected file triggers a confirmation prompt unless -f is given.

USE CASES.
  - Everyday file cleanup.
  - Unattended deletion in CI or scripts with -f.
  - Cautious, per-file review with -i, or one confirmation before a large delete with -I.
  - Auditing a recursive delete with -rv.
  - Clearing build output (build/, dist/, node_modules/) before a fresh build.
  - Pruning directories that should already be empty with -d, without risking contents.
  - Removing temporary files a script created, even if some were never written (-f).
  - Deleting a tree without crossing into mounted volumes (--one-file-system).
  - Removing files whose names start with '-' by ending options with --.

::

  rmx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Remove a few files
    rmx notes.txt draft.md

    # Recursively remove a directory, reporting each deletion
    rmx -rv build/

    # Unattended cleanup: no prompts, ignore missing paths
    rmx -rf /tmp/cache

    # Remove an empty directory only
    rmx -d generated/

    # Ask before each removal
    rmx -i *.bak

    # Ask once before a recursive delete
    rmx -I -r old-project/

    # Choose the prompt policy explicitly
    rmx --interactive=once *.log

    # Answer every prompt with yes from a pipeline
    yesx | rmx -i a.tmp b.tmp

    # Remove a file whose name starts with a dash
    rmx -- -draft.txt

    # Remove a tree but skip file systems mounted beneath it
    rmx -r --one-file-system scratch/

    # Disable the root failsafe for a deliberate recursive delete
    rmx -rf --no-preserve-root /mnt/scratch

    # Clean up and rebuild only if the removal succeeded
    rmx -rf dist/ && mkdirx dist

    # Detect a failed removal in a script
    rmx missing.txt || echox "rmx exited with $?"

    # Count how many entries a recursive delete removed
    rmx -rv cache/ | wcx -l

    # Remove temporary files with the gnux shell
    gnux shell -c 'rmx -fv *.tmp && echox done'

Options
~~~~~~~

::

  -d, --dir                             remove empty directories
  -f, --force                           ignore nonexistent files and arguments, never prompt
      --help                            display this help and exit
      --interactive string[="always"]   prompt according to WHEN: never, once, or always
      --no-preserve-root                do not treat '/' specially
      --one-file-system                 when removing recursively, skip directories on a different file system
      --preserve-root                   do not remove '/' (default) (default true)
  -i, --prompt-each                     prompt before every removal
  -I, --prompt-once                     prompt once before removing more than three files, or when removing recursively
  -r, --recursive                       remove directories and their contents recursively
  -v, --verbose                         explain what is being done
      --version                         output version information and exit

