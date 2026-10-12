.. _sha384sumx:

sha384sumx
----------

Print or check SHA-384 (384-bit) checksums.

Synopsis
~~~~~~~~


sha384sumx prints or checks SHA-384 message digests (FIPS 180-4, 384 bits). For each FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text mode, '*' with -b), and the file name. With no FILE, or when FILE is -, it reads standard input. Files are always read as bytes, so -b and -t change only the mark; when both are given, -t wins. --tag prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a backslash, newline or carriage return is escaped and its line starts with a backslash, unless -z ends lines with NUL instead. -c reads such lines (in either style) from the FILEs and checks each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches, unreadable files and improperly formatted lines; --quiet drops the OK lines, --status prints nothing, --ignore-missing skips listed files that do not exist, -w warns about each improperly formatted line and --strict fails because of them. The exit status is 0 when everything was read (and, with -c, matched), 1 otherwise.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. sha384sumx --help shows the options and the first examples; gnux help sha384sumx prints the full manual, and gnux help sha384sumx OPTION explains one option.

SEE ALSO. cksumx, md5sumx, sha1sumx, sha256sumx, sha512sumx, b2sumx.

PERMISSIONS. Needs read permission on each FILE (and, with -c, on the checksum lists and every file they name) and search permission on the directories leading to them. Nothing is ever modified.

USE CASES.
  - Verifying a download against the checksum its publisher lists.
  - Recording digests of release artifacts so users can check them.
  - Detecting whether files changed between two points in time.
  - Finding duplicate files by comparing their digests.
  - Checking a backup or a copy against the original.

::

  sha384sumx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Print the digest of a file
    sha384sumx file.iso

    # Digest several files
    sha384sumx *.tar.gz

    # Digest standard input
    echox -n hello | sha384sumx

    # Save digests to a file
    sha384sumx *.iso > sha384.sums

    # Verify files against saved digests
    sha384sumx -c sha384.sums

    # Verify quietly, showing only failures
    sha384sumx -c --quiet sha384.sums

    # Verify silently and branch on the result
    sha384sumx -c --status sha384.sums && echox verified

    # Verify only the listed files that exist
    sha384sumx -c --ignore-missing sha384.sums

    # Fail on malformed lines in the list
    sha384sumx -c --strict -w sha384.sums

    # Print BSD-style tagged lines
    sha384sumx --tag file.iso

    # Mark the files as read in binary mode
    sha384sumx -b file.iso

    # Digest in the portable gnux shell
    gnux shell -c 'sha384sumx file.iso > file.iso.sha384sum'

Options
~~~~~~~

::

  -b, --binary           read in binary mode
  -c, --check            read checksums from the FILEs and check them
      --help             display this help and exit
      --ignore-missing   don't fail or report status for missing files
      --quiet            don't print OK for each successfully verified file
      --status           don't output anything, status code shows success
      --strict           exit non-zero for improperly formatted checksum lines
      --tag              create a BSD-style checksum
  -t, --text             read in text mode (default)
      --version          output version information and exit
  -w, --warn             warn about improperly formatted checksum lines
  -z, --zero             end each output line with NUL, not newline, and disable file name escaping

