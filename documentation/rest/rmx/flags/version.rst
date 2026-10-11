.. _rmx:

rmx
---

Print version information and exit (--version).

Synopsis
~~~~~~~~


--version prints the rmx version string (injected at build time) and exits without doing anything else.

PERMISSIONS. Performs no filesystem access and needs no permissions; nothing is touched, even when operands are also supplied.

USE CASES.
  - Confirming which rmx build is installed.
  - Capturing the version in CI logs or bug reports.

::

  rmx --version

Examples
~~~~~~~~

::

    # Print the rmx version
    rmx --version

Options
~~~~~~~

::

  -h, --help      help for rmx
      --version   output version information and exit

