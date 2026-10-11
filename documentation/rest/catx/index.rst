.. _catx:

catx
----

Concatenate files and print them on standard output.

Synopsis
~~~~~~~~


catx writes each FILE to standard output, one after another. With no FILE, or when FILE is -, it reads standard input. Without options the bytes are copied untouched, so it is safe for binary data. -n numbers every output line and -b only the non-empty ones, with the numbering continuing across files; -s squeezes runs of empty lines into one. -E marks line ends with $, -T shows tabs as ^I and -v shows other control and non-ASCII bytes in ^X and M- notation; -A turns all three on. A file that cannot be read is reported and the remaining files are still printed.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. catx --help shows the options and the first examples; gnux help catx prints the full manual, and gnux help catx OPTION explains one option.

SEE ALSO. headx, tailx, wcx, echox.

PERMISSIONS. Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified; a directory operand is reported as 'Is a directory'.

USE CASES.
  - Printing a file, or joining several into one stream or file.
  - Feeding a file or standard input into a pipeline.
  - Numbering the lines of a listing with -n.
  - Finding stray tabs, trailing spaces, CRLF line endings or control characters with -A.
  - Appending one file to another with >>, such as adding a section to a changelog.
  - Collapsing runs of blank lines in generated text with -s.
  - Reassembling a file that was split into numbered parts.
  - Writing a here-document or piped text into a file from a script.
  - Checking that a set of files is readable, relying on the exit status.

::

  catx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # Print a file
    catx notes.txt

    # Join several files into one
    catx part1.txt part2.txt > whole.txt

    # Reassemble a file split into numbered parts (binary safe; on Windows let the gnux shell do the redirection)
    catx archive.tar.part-* > archive.tar

    # Append a file to the end of another
    catx release-notes.txt >> CHANGELOG.txt

    # Number the lines of a file
    catx -n main.go

    # Number only the non-empty lines
    catx -b poem.txt

    # Squeeze runs of blank lines into one, then number the result
    catx -sn draft.txt

    # Reveal tabs, line ends and control characters
    catx -A Makefile

    # Find trailing spaces and Windows line endings (shown as ^M$)
    catx -e config.ini

    # Show only line ends, or only tabs
    catx -E list.txt && catx -T Makefile

    # Show tabs and nonprinting bytes, but not line ends
    catx -t data.tsv

    # Make control and non-ASCII bytes visible in ^X and M- notation
    catx -v binary.dat | headx -n 5

    # Put standard input between two files (-u is accepted and ignored)
    echox '---' | catx -u header.txt - footer.txt

    # Write piped text into a file
    echox 'hello, world' | catx > greeting.txt

    # Count the lines of several files combined
    catx *.log | wcx -l

    # Report unreadable files but keep going, then act on the exit status
    catx a.txt missing.txt b.txt > all.txt || echox 'some files could not be read'

    # Concatenate files in the portable gnux shell
    gnux shell -c 'catx header.txt *.md > book.md && wcx -l book.md'

Options
~~~~~~~

::

      --help                    display this help and exit
  -n, --number                  number all output lines
  -b, --number-nonblank         number nonempty output lines, overrides -n
  -A, --show-all                equivalent to -vET
  -E, --show-ends               display $ at end of each line
  -v, --show-nonprinting        use ^ and M- notation, except for LFD and TAB
  -e, --show-nonprinting-ends   equivalent to -vE
  -t, --show-nonprinting-tabs   equivalent to -vT
  -T, --show-tabs               display TAB characters as ^I
  -s, --squeeze-blank           suppress repeated empty output lines
  -u, --unbuffered              (ignored)
      --version                 output version information and exit

