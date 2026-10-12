## gnux

The GNU core utilities' interface on every platform, in one Go binary.

### Synopsis

`gnux` brings the interface of the GNU core utilities to every platform. Each tool is named after its GNU counterpart with an x appended (`rm` is `rmx`, `cp` is `cpx`) and mirrors its flags, semantics and diagnostics.

Run a tool as a subcommand (`gnux rmx -rf` build), or `link` the binary to the tool's name and invoke it directly (`rmx -rf` build): `gnux` looks at the name it was started under and behaves as that tool. `--list` prints every tool name, one per line.

`gnux shell` is a small portable shell whose commands are these tools: it runs the same script on every platform, with no dependency on the system's own shell.

Every tool answers `--help` with its options, first examples and exit status. `gnux` help TOOL prints the tool's full manual - every option, permissions, use cases, every example for bash and PowerShell, exit status and related tools - and `gnux` help TOOL OPTION explains one option with the examples that use it.

```bash
gnux [OPTION]... [TOOL [ARGUMENT]...]
```

### Examples

```bash
  # run a tool through gnux
  gnux rmx -rf build/

  # make every tool a command of its own
  for t in $(gnux --list); do ln -s "$(command -v gnux)" "$HOME/bin/$t"; done
  rmx -rf build/

  # read a tool's full manual, or one option in depth
  gnux help rmx
  gnux help rmx --force
```

### Options

```text
      --help      display this help and exit
      --list      print the name of every tool, one per line
      --version   output version information and exit
```

### SEE ALSO

* [gnux b2sumx](b2sumx/index.md)	 - Print or check BLAKE2b (512-bit) checksums.
* [gnux base32x](base32x/index.md)	 - Encode or decode data in `base32`.
* [gnux base64x](base64x/index.md)	 - Encode or decode data in `base64`.
* [gnux basenamex](basenamex/index.md)	 - Strip the directory and suffix from file names.
* [gnux catx](catx/index.md)	 - Concatenate files and print them on standard output.
* [gnux chgrpx](chgrpx/index.md)	 - Change group ownership.
* [gnux chmodx](chmodx/index.md)	 - Change file mode bits.
* [gnux chownx](chownx/index.md)	 - Change file owner and group.
* [gnux cksumx](cksumx/index.md)	 - Print or check checksums and byte counts.
* [gnux commx](commx/index.md)	 - Compare two sorted files line by line.
* [gnux cpx](cpx/index.md)	 - Copy files and directories.
* [gnux csplitx](csplitx/index.md)	 - Split a file into sections determined by context lines.
* [gnux cutx](cutx/index.md)	 - Remove sections from each line of files.
* [gnux datex](datex/index.md)	 - Print the `date` and time.
* [gnux ddx](ddx/index.md)	 - Convert and copy a file, block by block.
* [gnux dfx](dfx/index.md)	 - Report file system space usage.
* [gnux dirnamex](dirnamex/index.md)	 - Strip the last component from file names.
* [gnux dux](dux/index.md)	 - Estimate file space usage.
* [gnux echox](echox/index.md)	 - Display a line of text.
* [gnux envx](envx/index.md)	 - Run a program in a modified environment, or print the environment.
* [gnux expandx](expandx/index.md)	 - Convert tabs to spaces.
* [gnux exprx](exprx/index.md)	 - Evaluate an expression and print its value.
* [gnux falsex](falsex/index.md)	 - Do nothing, unsuccessfully.
* [gnux fmtx](fmtx/index.md)	 - Reformat each paragraph of text.
* [gnux foldx](foldx/index.md)	 - Wrap each input line to fit in a specified width.
* [gnux headx](headx/index.md)	 - Print the first part of each file.
* [gnux hostnamex](hostnamex/index.md)	 - Print the name of the current host system.
* [gnux idx](idx/index.md)	 - Print real and effective user and group IDs.
* [gnux installx](installx/index.md)	 - Copy files into place and set their mode, owner and group.
* [gnux joinx](joinx/index.md)	 - Join lines of two files on a common field.
* [gnux killx](killx/index.md)	 - Send a signal to processes, or list signals.
* [gnux linkx](linkx/index.md)	 - Call the `link` function to create a hard `link` to a file.
* [gnux lnx](lnx/index.md)	 - Make hard or symbolic links between files.
* [gnux lsx](lsx/index.md)	 - List information about files and directory contents.
* [gnux md5sumx](md5sumx/index.md)	 - Print or check MD5 (128-bit) checksums.
* [gnux mkdirx](mkdirx/index.md)	 - Create the DIRECTORY(ies), if they do not already exist.
* [gnux mktempx](mktempx/index.md)	 - Create a temporary file or directory, safely, and print its name.
* [gnux mvx](mvx/index.md)	 - Move (rename) files and directories.
* [gnux nicex](nicex/index.md)	 - Run a program with modified scheduling priority.
* [gnux nlx](nlx/index.md)	 - Number lines of files.
* [gnux nohupx](nohupx/index.md)	 - Run a command immune to hangups, with output to a non-terminal.
* [gnux nprocx](nprocx/index.md)	 - Print the number of processing units available.
* [gnux odx](odx/index.md)	 - Dump files in octal and other formats.
* [gnux pastex](pastex/index.md)	 - Merge lines of files.
* [gnux printenvx](printenvx/index.md)	 - Print all or part of the environment.
* [gnux printfx](printfx/index.md)	 - Format and print data.
* [gnux pwdx](pwdx/index.md)	 - Print the full name of the current working directory.
* [gnux readlinkx](readlinkx/index.md)	 - Print resolved symbolic links or canonical file names.
* [gnux realpathx](realpathx/index.md)	 - Print the resolved absolute file name.
* [gnux rmdirx](rmdirx/index.md)	 - Remove the DIRECTORY(ies), if they are empty.
* [gnux rmx](rmx/index.md)	 - Remove (`unlink`) files and directories.
* [gnux seqx](seqx/index.md)	 - Print a sequence of numbers.
* [gnux sha1sumx](sha1sumx/index.md)	 - Print or check SHA-1 (160-bit) checksums.
* [gnux sha224sumx](sha224sumx/index.md)	 - Print or check SHA-224 (224-bit) checksums.
* [gnux sha256sumx](sha256sumx/index.md)	 - Print or check SHA-256 (256-bit) checksums.
* [gnux sha384sumx](sha384sumx/index.md)	 - Print or check SHA-384 (384-bit) checksums.
* [gnux sha512sumx](sha512sumx/index.md)	 - Print or check SHA-512 (512-bit) checksums.
* [gnux shell](shell/index.md)	 - Run a portable shell whose commands are the `gnux` tools.
* [gnux shufx](shufx/index.md)	 - Generate random permutations.
* [gnux sleepx](sleepx/index.md)	 - Delay for a specified amount of time.
* [gnux sortx](sortx/index.md)	 - Sort lines of text files.
* [gnux splitx](splitx/index.md)	 - Split a file into pieces.
* [gnux statx](statx/index.md)	 - Display file or file system status.
* [gnux tacx](tacx/index.md)	 - Concatenate and print files in reverse.
* [gnux tailx](tailx/index.md)	 - Print the last part of each file, optionally following it as it grows.
* [gnux teex](teex/index.md)	 - Copy standard input to each FILE, and also to standard output.
* [gnux testx](testx/index.md)	 - Check file types and compare values.
* [gnux timeoutx](timeoutx/index.md)	 - Run a command with a time limit.
* [gnux touchx](touchx/index.md)	 - Update the access and modification times of each FILE to the current time.
* [gnux truex](truex/index.md)	 - Do nothing, successfully.
* [gnux truncatex](truncatex/index.md)	 - Shrink or extend the size of each FILE to the specified size.
* [gnux trx](trx/index.md)	 - Translate, squeeze, and/or delete characters from standard input.
* [gnux ttyx](ttyx/index.md)	 - Print the file name of the terminal connected to standard input.
* [gnux unamex](unamex/index.md)	 - Print system information.
* [gnux unexpandx](unexpandx/index.md)	 - Convert spaces to tabs.
* [gnux uniqx](uniqx/index.md)	 - Report or omit repeated lines.
* [gnux unlinkx](unlinkx/index.md)	 - Call the `unlink` function to remove the specified file.
* [gnux uptimex](uptimex/index.md)	 - Tell how long the system has been running.
* [gnux wcx](wcx/index.md)	 - Print newline, word, and byte counts for each file.
* [gnux whoamix](whoamix/index.md)	 - Print the user name associated with the current effective user ID.
* [gnux whox](whox/index.md)	 - Show `who` is logged on.
* [gnux yesx](yesx/index.md)	 - Output a string repeatedly until killed.

