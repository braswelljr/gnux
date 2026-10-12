.. _readlinkx:

readlinkx
---------

Print resolved symbolic links or canonical file names.

Synopsis
~~~~~~~~


readlinkx prints the value of each FILE, which must be a symbolic link: the target exactly as it was stored, not resolved. With -f, -e or -m it prints the canonical name instead: absolute, with every symbolic link and every '.' and '..' component resolved. -f requires every component but the last to exist, -e requires all of them, and -m requires none. As in GNU readlink, failures are silent by default (-q and -s say so explicitly) and only the exit status reports them; -v prints a diagnostic for each. -n drops the trailing newline, and is ignored with a warning when there is more than one FILE; -z ends each name with NUL. GNU readlink lets the last of -f, -e and -m win; readlinkx cannot see option order, so it picks the most demanding (-e, then -f, then -m), and -v wins over -q and -s.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. readlinkx --help shows the options and the first examples; gnux help readlinkx prints the full manual, and gnux help readlinkx OPTION explains one option.

SEE ALSO. realpathx, lnx, lsx.

PERMISSIONS. Needs search permission on every directory leading to FILE, and on every directory a link leads through. Reading a link's value needs no permission on the link itself. Nothing is ever modified.

USE CASES.
  - Finding where a symbolic link points, exactly as it was created.
  - Getting the absolute, link-free path of a script or binary with -f.
  - Checking in a script that a path fully exists after resolution with -e.
  - Normalizing a path that does not exist yet with -m.
  - Detecting whether a file is a symbolic link from the exit status.
  - Comparing two paths for identity after resolving their links.
  - Producing NUL-separated output for names with newlines with -z.

::

  readlinkx [OPTION]... FILE...

Examples
~~~~~~~~

::

    # Print where a link points
    readlinkx current

    # Print the canonical, absolute name of a file
    readlinkx -f ./bin/../bin/tool

    # Require the whole path to exist
    readlinkx -e config/app.yaml

    # Normalize a path that may not exist yet
    readlinkx -m build/../dist/out

    # Explain why a name cannot be resolved
    readlinkx -v -e missing/file

    # Test whether a file is a symbolic link
    readlinkx current > /dev/null && echox 'a link'

    # Capture a resolved path without a trailing newline
    printfx 'cd %s\n' "$(readlinkx -n -f link)"

    # Resolve several links at once, NUL-separated
    readlinkx -z -f a b c | trx '\0' '\n'

    # Find the directory a script really lives in
    dirnamex "$(readlinkx -f "$0")"

    # Check that two names reach the same file
    [ "$(readlinkx -f a)" = "$(readlinkx -f b)" ] && echox same

    # Resolve a link in the portable gnux shell
    gnux shell -c 'readlinkx -f current'

Options
~~~~~~~

::

  -f, --canonicalize            canonicalize by following every symlink in every component recursively; all but the last component must exist
  -e, --canonicalize-existing   canonicalize by following every symlink in every component recursively; all components must exist
  -m, --canonicalize-missing    canonicalize by following every symlink in every component recursively, without requirements on existence
      --help                    display this help and exit
  -n, --no-newline              do not output the trailing delimiter
  -q, --quiet                   suppress most error messages (on by default)
  -s, --silent                  suppress most error messages (on by default)
  -v, --verbose                 report error messages
      --version                 output version information and exit
  -z, --zero                    end each output line with NUL, not newline

