.. _cksumx:

cksumx
------

Print or check checksums and byte counts.

Synopsis
~~~~~~~~


cksumx prints a checksum and the byte count of each FILE. With no FILE, or when FILE is -, it reads standard input (and, with no FILE at all, prints no name). By default it computes the POSIX CRC (the 'crc' algorithm), as plain cksum always has. -a ALGORITHM picks another: sysv and bsd (the legacy sum checksums, with a block count rather than a byte count), crc32b (the CRC-32 of zlib and gzip), or the message digests md5, sha1, sha224, sha256, sha384, sha512 and blake2b. Digests are printed as BSD-style tagged lines ('SHA256 (FILE) = DIGEST'); --untagged prints them in the md5sumx style instead, and when both --tag and --untagged are given, --tag wins. --base64 prints digests in base64, --raw writes the bare binary checksum of a single input, -l BITS shortens a blake2b digest, and -z ends lines with NUL. -c checks the files listed in tagged or untagged digest lines, detecting the algorithm from each tagged line when -a is not given, with --ignore-missing, --quiet, --status, --strict and -w as in md5sumx. The legacy algorithms cannot be checked.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. cksumx --help shows the options and the first examples; gnux help cksumx prints the full manual, and gnux help cksumx OPTION explains one option.

SEE ALSO. md5sumx, sha256sumx, b2sumx, wcx.

PERMISSIONS. Needs read permission on each FILE (and, with -c, on the checksum lists and every file they name) and search permission on the directories leading to them. Nothing is ever modified.

USE CASES.
  - Comparing a file with the CRC a POSIX system's cksum printed for it.
  - Computing any of the supported digests with one tool and one syntax.
  - Verifying mixed checksum lists that name their algorithm per line.
  - Getting a gzip-compatible CRC-32 with -a crc32b.
  - Printing a digest in base64, as package managers and SRI hashes use.

::

  cksumx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Print the POSIX CRC and size of a file
    cksumx file.iso

    # Checksum standard input
    echox hello | cksumx

    # Print a SHA-256 digest as a tagged line
    cksumx -a sha256 file.iso

    # Print it in the sha256sumx style
    cksumx -a sha256 --untagged file.iso

    # Print a digest in base64 (as SRI hashes use)
    cksumx -a sha384 --base64 app.js

    # Print a 256-bit BLAKE2b digest
    cksumx -a blake2b -l 256 file.iso

    # Print the gzip-compatible CRC-32
    cksumx -a crc32b data.bin

    # Print the legacy BSD sum checksum
    cksumx -a bsd file.txt

    # Save tagged digests and check them later
    cksumx -a sha256 *.iso > SUMS && cksumx -c SUMS

    # Check silently and branch on the result
    cksumx -c --status SUMS && echox verified

    # Write the raw binary digest of one file
    cksumx -a sha256 --raw file.iso > file.sha256.bin

    # Checksum in the portable gnux shell
    gnux shell -c 'cksumx -a sha256 file.iso > file.iso.sum'

Options
~~~~~~~

::

  -a, --algorithm TYPE   select the digest TYPE to use: sysv, bsd, crc, crc32b, md5, sha1, sha224, sha256, sha384, sha512, blake2b
      --base64           emit base64-encoded digests, not hexadecimal
  -c, --check            read checksums from the FILEs and check them
      --help             display this help and exit
      --ignore-missing   don't fail or report status for missing files
  -l, --length bits      digest length in bits; must not exceed the maximum for the blake2 algorithm and must be a multiple of 8
      --quiet            don't print OK for each successfully verified file
      --raw              emit a raw binary digest, not hexadecimal
      --status           don't output anything, status code shows success
      --strict           exit non-zero for improperly formatted checksum lines
      --tag              create a BSD-style checksum (the default)
      --untagged         create a reversed style checksum, without digest type
      --version          output version information and exit
  -w, --warn             warn about improperly formatted checksum lines
  -z, --zero             end each output line with NUL, not newline, and disable file name escaping

