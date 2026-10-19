.. _gnux:

gnux
----

The GNU core utilities' interface on every platform, in one Go binary.

Synopsis
~~~~~~~~


gnux brings the interface of the GNU core utilities to every platform. Each tool is named after its GNU counterpart with an x appended (rm is rmx, cp is cpx) and mirrors its flags, semantics and diagnostics.

Run a tool as a subcommand (gnux rmx -rf build), or link the binary to the tool's name and invoke it directly (rmx -rf build): gnux looks at the name it was started under and behaves as that tool. --list prints every tool name, one per line.

gnux shell is a small portable shell whose commands are these tools: it runs the same script on every platform, with no dependency on the system's own shell.

Every tool answers --help with its options, first examples and exit status. gnux help TOOL prints the tool's full manual - every option, permissions, use cases, every example for bash and PowerShell, exit status and related tools - and gnux help TOOL OPTION explains one option with the examples that use it.

::

  gnux [OPTION]... [TOOL [ARGUMENT]...]

Examples
~~~~~~~~

::

    # run a tool through gnux
    gnux rmx -rf build/

    # make every tool a command of its own
    for t in $(gnux --list); do ln -s "$(command -v gnux)" "$HOME/bin/$t"; done
    rmx -rf build/

    # read a tool's full manual, or one option in depth
    gnux help rmx
    gnux help rmx --force

Options
~~~~~~~

::

      --help      display this help and exit
      --list      print the name of every tool, one per line
      --version   output version information and exit

SEE ALSO
~~~~~~~~

* `gnux archx <archx/index.rst>`_ 	 - Print machine architecture.
* `gnux b2sumx <b2sumx/index.rst>`_ 	 - Print or check BLAKE2b (512-bit) checksums.
* `gnux base32x <base32x/index.rst>`_ 	 - Encode or decode data in base32.
* `gnux base64x <base64x/index.rst>`_ 	 - Encode or decode data in base64.
* `gnux basenamex <basenamex/index.rst>`_ 	 - Strip the directory and suffix from file names.
* `gnux catx <catx/index.rst>`_ 	 - Concatenate files and print them on standard output.
* `gnux chgrpx <chgrpx/index.rst>`_ 	 - Change group ownership.
* `gnux chmodx <chmodx/index.rst>`_ 	 - Change file mode bits.
* `gnux chownx <chownx/index.rst>`_ 	 - Change file owner and group.
* `gnux cksumx <cksumx/index.rst>`_ 	 - Print or check checksums and byte counts.
* `gnux commx <commx/index.rst>`_ 	 - Compare two sorted files line by line.
* `gnux cpx <cpx/index.rst>`_ 	 - Copy files and directories.
* `gnux csplitx <csplitx/index.rst>`_ 	 - Split a file into sections determined by context lines.
* `gnux cutx <cutx/index.rst>`_ 	 - Remove sections from each line of files.
* `gnux datex <datex/index.rst>`_ 	 - Print the date and time.
* `gnux ddx <ddx/index.rst>`_ 	 - Convert and copy a file, block by block.
* `gnux dfx <dfx/index.rst>`_ 	 - Report file system space usage.
* `gnux dirnamex <dirnamex/index.rst>`_ 	 - Strip the last component from file names.
* `gnux dirx <dirx/index.rst>`_ 	 - List directory contents in columns, whether or not the output is a terminal.
* `gnux dux <dux/index.rst>`_ 	 - Estimate file space usage.
* `gnux echox <echox/index.rst>`_ 	 - Display a line of text.
* `gnux envx <envx/index.rst>`_ 	 - Run a program in a modified environment, or print the environment.
* `gnux expandx <expandx/index.rst>`_ 	 - Convert tabs to spaces.
* `gnux exprx <exprx/index.rst>`_ 	 - Evaluate an expression and print its value.
* `gnux factorx <factorx/index.rst>`_ 	 - Print the prime factors of each specified integer NUMBER.
* `gnux falsex <falsex/index.rst>`_ 	 - Do nothing, unsuccessfully.
* `gnux findx <findx/index.rst>`_ 	 - Search for files in a directory hierarchy.
* `gnux fmtx <fmtx/index.rst>`_ 	 - Reformat each paragraph of text.
* `gnux foldx <foldx/index.rst>`_ 	 - Wrap each input line to fit in a specified width.
* `gnux grepx <grepx/index.rst>`_ 	 - Print the lines that match patterns.
* `gnux groupsx <groupsx/index.rst>`_ 	 - Print the groups a user is in.
* `gnux headx <headx/index.rst>`_ 	 - Print the first part of each file.
* `gnux hostnamex <hostnamex/index.rst>`_ 	 - Print the name of the current host system.
* `gnux idx <idx/index.rst>`_ 	 - Print real and effective user and group IDs.
* `gnux installx <installx/index.rst>`_ 	 - Copy files into place and set their mode, owner and group.
* `gnux joinx <joinx/index.rst>`_ 	 - Join lines of two files on a common field.
* `gnux killx <killx/index.rst>`_ 	 - Send a signal to processes, or list signals.
* `gnux linkx <linkx/index.rst>`_ 	 - Call the link function to create a hard link to a file.
* `gnux lnx <lnx/index.rst>`_ 	 - Make hard or symbolic links between files.
* `gnux lognamex <lognamex/index.rst>`_ 	 - Print the user's login name.
* `gnux lsx <lsx/index.rst>`_ 	 - List information about files and directory contents.
* `gnux md5sumx <md5sumx/index.rst>`_ 	 - Print or check MD5 (128-bit) checksums.
* `gnux mkdirx <mkdirx/index.rst>`_ 	 - Create the DIRECTORY(ies), if they do not already exist.
* `gnux mktempx <mktempx/index.rst>`_ 	 - Create a temporary file or directory, safely, and print its name.
* `gnux mvx <mvx/index.rst>`_ 	 - Move (rename) files and directories.
* `gnux nicex <nicex/index.rst>`_ 	 - Run a program with modified scheduling priority.
* `gnux nlx <nlx/index.rst>`_ 	 - Number lines of files.
* `gnux nohupx <nohupx/index.rst>`_ 	 - Run a command immune to hangups, with output to a non-terminal.
* `gnux nprocx <nprocx/index.rst>`_ 	 - Print the number of processing units available.
* `gnux odx <odx/index.rst>`_ 	 - Dump files in octal and other formats.
* `gnux pastex <pastex/index.rst>`_ 	 - Merge lines of files.
* `gnux printenvx <printenvx/index.rst>`_ 	 - Print all or part of the environment.
* `gnux printfx <printfx/index.rst>`_ 	 - Format and print data.
* `gnux pwdx <pwdx/index.rst>`_ 	 - Print the full name of the current working directory.
* `gnux readlinkx <readlinkx/index.rst>`_ 	 - Print resolved symbolic links or canonical file names.
* `gnux realpathx <realpathx/index.rst>`_ 	 - Print the resolved absolute file name.
* `gnux rmdirx <rmdirx/index.rst>`_ 	 - Remove the DIRECTORY(ies), if they are empty.
* `gnux rmx <rmx/index.rst>`_ 	 - Remove (unlink) files and directories.
* `gnux sedx <sedx/index.rst>`_ 	 - Filter and transform text with a stream editor.
* `gnux seqx <seqx/index.rst>`_ 	 - Print a sequence of numbers.
* `gnux sha1sumx <sha1sumx/index.rst>`_ 	 - Print or check SHA-1 (160-bit) checksums.
* `gnux sha224sumx <sha224sumx/index.rst>`_ 	 - Print or check SHA-224 (224-bit) checksums.
* `gnux sha256sumx <sha256sumx/index.rst>`_ 	 - Print or check SHA-256 (256-bit) checksums.
* `gnux sha384sumx <sha384sumx/index.rst>`_ 	 - Print or check SHA-384 (384-bit) checksums.
* `gnux sha512sumx <sha512sumx/index.rst>`_ 	 - Print or check SHA-512 (512-bit) checksums.
* `gnux shell <shell/index.rst>`_ 	 - Run a portable shell whose commands are the gnux tools.
* `gnux shufx <shufx/index.rst>`_ 	 - Generate random permutations.
* `gnux sleepx <sleepx/index.rst>`_ 	 - Delay for a specified amount of time.
* `gnux sortx <sortx/index.rst>`_ 	 - Sort lines of text files.
* `gnux splitx <splitx/index.rst>`_ 	 - Split a file into pieces.
* `gnux statx <statx/index.rst>`_ 	 - Display file or file system status.
* `gnux sumx <sumx/index.rst>`_ 	 - Print the checksum and block count of each FILE.
* `gnux tacx <tacx/index.rst>`_ 	 - Concatenate and print files in reverse.
* `gnux tailx <tailx/index.rst>`_ 	 - Print the last part of each file, optionally following it as it grows.
* `gnux teex <teex/index.rst>`_ 	 - Copy standard input to each FILE, and also to standard output.
* `gnux testx <testx/index.rst>`_ 	 - Check file types and compare values.
* `gnux timeoutx <timeoutx/index.rst>`_ 	 - Run a command with a time limit.
* `gnux touchx <touchx/index.rst>`_ 	 - Update the access and modification times of each FILE to the current time.
* `gnux truex <truex/index.rst>`_ 	 - Do nothing, successfully.
* `gnux truncatex <truncatex/index.rst>`_ 	 - Shrink or extend the size of each FILE to the specified size.
* `gnux trx <trx/index.rst>`_ 	 - Translate, squeeze, and/or delete characters from standard input.
* `gnux tsortx <tsortx/index.rst>`_ 	 - Write a totally ordered list consistent with the partial ordering in FILE.
* `gnux ttyx <ttyx/index.rst>`_ 	 - Print the file name of the terminal connected to standard input.
* `gnux unamex <unamex/index.rst>`_ 	 - Print system information.
* `gnux unexpandx <unexpandx/index.rst>`_ 	 - Convert spaces to tabs.
* `gnux uniqx <uniqx/index.rst>`_ 	 - Report or omit repeated lines.
* `gnux unlinkx <unlinkx/index.rst>`_ 	 - Call the unlink function to remove the specified file.
* `gnux uptimex <uptimex/index.rst>`_ 	 - Tell how long the system has been running.
* `gnux usersx <usersx/index.rst>`_ 	 - Print the user names of users currently logged in to the current host.
* `gnux vdirx <vdirx/index.rst>`_ 	 - List directory contents in the long format.
* `gnux wcx <wcx/index.rst>`_ 	 - Print newline, word, and byte counts for each file.
* `gnux whoamix <whoamix/index.rst>`_ 	 - Print the user name associated with the current effective user ID.
* `gnux whox <whox/index.rst>`_ 	 - Show who is logged on.
* `gnux xargsx <xargsx/index.rst>`_ 	 - Build and run command lines from standard input.
* `gnux yesx <yesx/index.rst>`_ 	 - Output a string repeatedly until killed.

