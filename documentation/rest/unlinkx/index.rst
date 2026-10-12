.. _unlinkx:

unlinkx
-------

Call the unlink function to remove the specified file.

Synopsis
~~~~~~~~


unlinkx removes the single name FILE with the unlink(2) system call. It takes exactly one operand and no options: it never recurses, never prompts, never ignores a missing file, and refuses a directory, even an empty one, with 'Is a directory' (use rmdirx for that). A symbolic link is removed, not its target. Removing the last name of a file frees its data once no process has it open.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. unlinkx --help shows the options and the first examples; gnux help unlinkx prints the full manual, and gnux help unlinkx OPTION explains one option.

SEE ALSO. rmx, rmdirx, linkx.

PERMISSIONS. Needs write and search permission on the directory containing FILE; permission on FILE itself does not matter. In a directory with the sticky bit set (such as /tmp), only the file's owner, the directory's owner or root may remove it. On Windows, a read-only file or one open in another process cannot be removed.

USE CASES.
  - Removing exactly one file in a script, failing loudly if anything is unexpected.
  - Removing a symbolic link without any risk of touching what it points to.
  - Releasing a lock file created with linkx.
  - Removing a file whose name could be mistaken for an option, with --.
  - Removing one of several hard links while keeping the others.

::

  unlinkx FILE

Examples
~~~~~~~~

::

    # Remove one file
    unlinkx old.log

    # Remove a symbolic link, leaving its target alone
    unlinkx current

    # A directory is refused (use rmdirx)
    unlinkx build || rmdirx build

    # Release a lock file
    unlinkx app.lock

    # Remove a file whose name starts with a dash
    unlinkx -- -oops

    # Drop one hard link and keep the other name
    linkx data data.bak && unlinkx data

    # Fail when the file is already gone (rmx -f would not)
    unlinkx tmp.txt || echox 'nothing to remove'

    # Remove a file with spaces in its name
    unlinkx 'my file.txt'

    # Remove a temporary file at the end of a script
    tmp=$(mktempx) && echox data > "$tmp" && unlinkx "$tmp"

    # Remove a file in the portable gnux shell
    gnux shell -c 'unlinkx out.txt'

Options
~~~~~~~

::

      --help      display this help and exit
      --version   output version information and exit

