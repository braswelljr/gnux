.. _preserve-root:

preserve-root
-------------

Refuse to recurse on '/' (--preserve-root, default; --no-preserve-root overrides).

Synopsis
~~~~~~~~


--preserve-root is the default failsafe: rmx refuses to operate recursively on the filesystem root '/'. --no-preserve-root disables the failsafe for the rare, deliberate case where you really do mean to recurse from a root. Use it with extreme care.

PERMISSIONS. This is a path check, not a permission check: rmx refuses '/' before it touches the filesystem, regardless of whether you would otherwise have permission. It never grants access you do not already have.

USE CASES.
  - Left at its default, protecting against a catastrophic 'rmx -r /' typo.
  - Deliberately overridden (--no-preserve-root) only in throwaway or container roots.
  - Guarding scripts where an empty variable could turn a path into '/'.
  - Stating the default explicitly (--preserve-root) in scripts for reviewers.

::

  preserve-root [FILE]...

Examples
~~~~~~~~

::

    # The default failsafe refuses to recurse on the root
    rmx -r /

    # Disable the failsafe (dangerous - only when you mean it)
    rmx -r --no-preserve-root /mnt/scratch

    # State the default explicitly in a script
    rmx -rf --preserve-root "$BUILD_DIR/"

    # The failsafe makes the command fail, which a script can detect
    rmx -rf / || echox "refused, status $?"

Options
~~~~~~~

::

  -h, --help               help for preserve-root
      --no-preserve-root   do not treat '/' specially
      --preserve-root      do not remove '/' (default) (default true)

