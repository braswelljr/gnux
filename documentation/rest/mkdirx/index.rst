.. _mkdirx:

mkdirx
------

Create the DIRECTORY(ies), if they do not already exist.

Synopsis
~~~~~~~~


mkdirx creates each DIRECTORY in the order given. Without options the parent must already exist and the directory must not, otherwise the operand is reported and the rest are still processed. -p creates any missing parents along the way and treats a directory that is already there as success, which makes the command safe to repeat. -m sets the permissions of the new directory using either an octal mode (755) or a chmod-style symbolic one (u=rwx,go=rx), and -v reports each directory as it is created.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. mkdirx --help shows the options and the first examples; gnux help mkdirx prints the full manual, and gnux help mkdirx OPTION explains one option.

SEE ALSO. rmdirx, touchx, cpx.

PERMISSIONS. Creating a directory needs write and search permission on its parent. New directories get a=rwx minus the umask. A mode given with -m is applied exactly, without being reduced by the umask, but only to the operand: parents made by -p keep the default mode, plus owner write and search so the next level can be created inside them. On Windows only the read-only attribute is representable, so most of a mode has no effect there.

USE CASES.
  - Creating a nested output tree in one step with -p.
  - Idempotent setup in scripts and CI, where the directory may already exist.
  - Making a private directory with -m 700 regardless of the umask.
  - Logging what a provisioning script created with -v.
  - Scaffolding a project layout (src, tests, docs) in a single command.
  - Creating a shared drop directory with the sticky bit, such as mode 1777.
  - Making sure the parent of an output file exists before writing it.
  - Guarding a script step with && so later commands only run once the directory is there.

::

  mkdirx [OPTION]... DIRECTORY...

Examples
~~~~~~~~

::

    # Create a directory
    mkdirx build

    # Create several sibling directories at once
    mkdirx src docs tests

    # Create a nested tree, reporting each level
    mkdirx -pv out/reports/2024

    # Scaffold a project layout, reporting every directory made
    mkdirx -pv project/src project/tests project/docs

    # Create a directory only its owner can enter
    mkdirx -m 700 ~/.secrets

    # Use a symbolic mode
    mkdirx -m u=rwx,go=rx shared

    # Grant everything except write for others, using a symbolic mode with a removal
    mkdirx -m a=rwx,o-w drop

    # Create a world-writable scratch directory with the sticky bit set
    mkdirx -m 1777 scratch

    # Create missing parents but give only the last directory an explicit mode
    mkdirx -p -m 755 www/public

    # Run the next step only once the directory exists (safe to repeat)
    mkdirx -p out/reports/2024 && echox ready

    # React to a failure, here a directory that already exists
    mkdirx build 2>/dev/null || echox 'build already exists'

    # Create the directory a file is about to be written to
    mkdirx -p "$(dirnamex out/reports/q1.csv)"

    # Keep an otherwise empty directory under version control
    mkdirx -p cache && touchx cache/.keep && lsx -A cache

    # Do the same setup portably with the gnux shell
    gnux shell -c 'mkdirx -pv site/assets && touchx site/index.html && lsx site'

Options
~~~~~~~

::

      --help        display this help and exit
  -m, --mode MODE   set file MODE (as in chmod), not a=rwx - umask
  -p, --parents     no error if existing, make parent directories as needed
  -v, --verbose     print a message for each created directory
      --version     output version information and exit

