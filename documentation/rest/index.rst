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

* `gnux basenamex <basenamex/index.rst>`_ 	 - Strip the directory and suffix from file names.
* `gnux catx <catx/index.rst>`_ 	 - Concatenate files and print them on standard output.
* `gnux cpx <cpx/index.rst>`_ 	 - Copy files and directories.
* `gnux dirnamex <dirnamex/index.rst>`_ 	 - Strip the last component from file names.
* `gnux echox <echox/index.rst>`_ 	 - Display a line of text.
* `gnux envx <envx/index.rst>`_ 	 - Run a program in a modified environment, or print the environment.
* `gnux falsex <falsex/index.rst>`_ 	 - Do nothing, unsuccessfully.
* `gnux headx <headx/index.rst>`_ 	 - Print the first part of each file.
* `gnux lnx <lnx/index.rst>`_ 	 - Make hard or symbolic links between files.
* `gnux lsx <lsx/index.rst>`_ 	 - List information about files and directory contents.
* `gnux mkdirx <mkdirx/index.rst>`_ 	 - Create the DIRECTORY(ies), if they do not already exist.
* `gnux mvx <mvx/index.rst>`_ 	 - Move (rename) files and directories.
* `gnux pwdx <pwdx/index.rst>`_ 	 - Print the full name of the current working directory.
* `gnux realpathx <realpathx/index.rst>`_ 	 - Print the resolved absolute file name.
* `gnux rmdirx <rmdirx/index.rst>`_ 	 - Remove the DIRECTORY(ies), if they are empty.
* `gnux rmx <rmx/index.rst>`_ 	 - Remove (unlink) files and directories.
* `gnux shell <shell/index.rst>`_ 	 - Run a portable shell whose commands are the gnux tools.
* `gnux sleepx <sleepx/index.rst>`_ 	 - Delay for a specified amount of time.
* `gnux tailx <tailx/index.rst>`_ 	 - Print the last part of each file, optionally following it as it grows.
* `gnux touchx <touchx/index.rst>`_ 	 - Update the access and modification times of each FILE to the current time.
* `gnux truex <truex/index.rst>`_ 	 - Do nothing, successfully.
* `gnux wcx <wcx/index.rst>`_ 	 - Print newline, word, and byte counts for each file.
* `gnux yesx <yesx/index.rst>`_ 	 - Output a string repeatedly until killed.

