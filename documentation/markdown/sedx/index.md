## sedx

Filter and transform text with a stream editor.

Mirrors GNU `sed`. Run it as `sedx`, or as `gnux sedx`.

### Synopsis

```bash
sedx [OPTION]... {script-only-if-no-other-script} [input-file]...
```

### GNU reference

- [GNU coreutils manual: sed invocation](https://www.gnu.org/software/coreutils/manual/html_node/sed-invocation.html)
- [Source: src/sed.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/sed.c)

### Description

`sedx` reads its input one line at a time into the pattern space, runs the script on it and, unless `-n` is given, prints what is left. The input is the input-files in order as one stream, or standard input when there is none or the file is -. The script is the first operand, or every `-e` SCRIPT and `-f` SCRIPT-FILE in command-line order when at least one is given. It is compiled in full before anything is read, so a mistake in it changes no file.

A script is a list of commands separated by newlines or semicolons, each optionally preceded by an address: a line number N, $ for the last line, /REGEX/ or \cREGEXc (with I to ignore case and M to let ^ and $ match at every line of the pattern space), FIRST~STEP, or a range ADDR1,ADDR2 where ADDR2 may also be +N (N more lines) or ~N (up to the next multiple of N), and 0,/REGEX/ lets the range end on the first line. A ! after the address selects the other lines. An empty regular expression stands for the last one used.

The commands are those of GNU `sed`: { } to group; s/REGEX/REPLACEMENT/FLAGS with any delimiter, the flags g, p, a number, i or I, m or M and w FILE, and in the replacement &, \0 to \9, \n, and the case conversions \L \U \l \u \E; y/SOURCE/DEST/; d D p P n N g G h H x; a, i and c in both the one-line form (a TEXT) and the POSIX form (a\ followed by the text on the next lines); = l q Q with an optional exit status; r FILE, R FILE, w FILE and W FILE, where /dev/stdin, /dev/stdout and /dev/stderr are the standard streams; b, t and T with a : LABEL; z; F; v; and # comments. A script whose first line is #n runs as with `-n`.

Regular expressions are POSIX basic ones with the GNU extensions \+ \? \| or, with `-E`, extended ones; both take \w \W \s \S \b \B \< \> \` \' and the character escapes \n \t \f \v \a \r \cX \dNNN \oNNN \xHH, and both find the leftmost longest match.

`-i` edits each file in place: the output is written to a temporary file in the same directory, which takes the file's permissions and then its name; with a SUFFIX the original is kept under that name (a * in SUFFIX stands for the file's base name, and a SUFFIX containing a slash names another directory). The suffix must be attached: `-i`.bak or `--in-place`=.bak. A missing newline at the end of the last line is missing from the output too.

Differences from GNU `sed`: the e command and the e flag of s are refused, as `sedx` never runs another program. A back reference inside a regular expression (\(a\)\1) is refused, because the linear-time matcher cannot express one; \1 in a replacement works. \< and \> both match at any word boundary, like \b, and word characters are ASCII letters, digits and the underscore. A repetition count may be at most 1000. Text is taken as UTF-8 whatever the locale, a regular expression must itself be valid UTF-8, and l shows every non-ASCII byte in octal. `--debug` is not supported, and long options cannot be abbreviated. `--posix` only changes what n and N do at the end of the input (nothing is printed); the GNU extensions stay available. `-u` flushes the output after every line but still reads the input in blocks. `-i` does not carry over a file's owner, group or extended attributes. A value of `-l` that is not a number is an error rather than 0, and running with no script is a one-line error rather than the full usage text.

### Permissions

Needs read permission on each input-file, script file and file read by r or R. w, W and the w flag of s create or `truncate` the file they name as soon as the script has been compiled, which needs write permission on it or on its directory. `-i` needs write permission on the directory of each file, since the file is replaced by a new one, not rewritten: the new file has the old one's permission bits but belongs to the user running `sedx`, any hard `link` keeps the old contents, and a symbolic `link` is replaced by a regular file unless `--follow-symlinks` is given. `--sandbox` refuses any script that reads or writes a file of its own (r, R, w, W).

### Use cases

- Replacing a string or a pattern in a stream or, with `-i`, in a set of files.
- Printing a range of lines of a file, by number or between two patterns.
- Deleting or commenting out the lines of a configuration file that match a pattern.
- Inserting, appending or replacing lines at a known place in a generated file.
- Rewriting fields with groups, such as swapping two columns or reformatting dates.
- Running one edit script the same way on Linux, macOS and Windows, where the system `sed` differs or is missing.

### Flags

```text
  -b, --binary              open files in binary mode (always the case here)
  -e, --expression SCRIPT   add the SCRIPT to the commands to be executed
  -f, --file SCRIPT-FILE    add the contents of SCRIPT-FILE to the commands to be executed
      --follow-symlinks     follow symlinks when processing in place
      --help                display this help and exit
  -i, --in-place            edit files in place; -iSUFFIX or --in-place=SUFFIX also keeps each original under that suffix
  -l, --line-length N       specify the desired line-wrap length N for the 'l' command (0 never wraps; default 70)
  -z, --null-data           separate lines by NUL characters
      --posix               print nothing when n or N finds no next line, as POSIX has it
  -n, --quiet               suppress automatic printing of pattern space
  -E, --regexp-extended     use extended regular expressions in the script (-r is accepted too)
      --sandbox             operate in sandbox mode (refuse the r, R, w and W commands)
  -s, --separate            consider files as separate rather than as a single continuous long stream
      --silent              same as --quiet
  -u, --unbuffered          flush the output after every line
      --version             output version information and exit
```

### Examples

Replace every occurrence of a word in a stream:

```bash
sedx 's/staging/production/g' deploy.yaml
```

```powershell
sedx 's/staging/production/g' deploy.yaml
```

Point a config file at a new host, keeping the original as config.ini.bak:

```bash
sedx -i.bak 's/^host = .*/host = db2.internal/' config.ini
```

```powershell
sedx '-i.bak' 's/^host = .*/host = db2.internal/' config.ini
```

Print only lines 10 to 20 of a log:

```bash
sedx -n '10,20p' app.log
```

```powershell
sedx -n '10,20p' app.log
```

Show the block between two markers in a file:

```bash
sedx -n '/BEGIN CERTIFICATE/,/END CERTIFICATE/p' bundle.pem
```

```powershell
sedx -n '/BEGIN CERTIFICATE/,/END CERTIFICATE/p' bundle.pem
```

Strip comments and blank lines from a config file:

```bash
sedx -e 's/#.*//' -e '/^[[:space:]]*$/d' nginx.conf
```

```powershell
sedx -e 's/#.*//' -e '/^[[:space:]]*$/d' nginx.conf
```

Swap the first two comma-separated fields with extended regular expressions:

```bash
sedx -E 's/^([^,]*),([^,]*)/\2,\1/' people.csv
```

```powershell
sedx -E 's/^([^,]*),([^,]*)/\2,\1/' people.csv
```

Output:

```text
Lovelace,Ada,1815
Hopper,Grace,1906
```

Add a line after the one that matches, in every YAML file of a directory:

```bash
sedx -i '/^dependencies:/a\  - gnux' *.yaml
```

```powershell
sedx -i '/^dependencies:/a\  - gnux' (Get-ChildItem *.yaml).Name
```

Join all the lines of a file with commas:

```bash
sedx ':a;N;$!ba;s/\n/,/g' ids.txt
```

```powershell
sedx ':a;N;$!ba;s/\n/,/g' ids.txt
```

Output:

```text
17,23,42
```

Print the first three lines and stop reading, like `headx -n` 3:

```bash
sedx 3q huge.log
```

```powershell
sedx 3q huge.log
```

Capitalize the first letter of every word:

```bash
sedx 's/\b\w/\u&/g' title.txt
```

```powershell
sedx 's/\b\w/\u&/g' title.txt
```

Output:

```text
The Quick Brown Fox
```

Remove trailing white space from every source file, in place:

```bash
sedx -i 's/[[:space:]]*$//' main.go util.go
```

```powershell
sedx -i 's/[[:space:]]*$//' main.go util.go
```

Run an edit script kept in a file:

```bash
sedx -f cleanup.sed report.txt > report.clean.txt
```

```powershell
sedx -f cleanup.sed report.txt > report.clean.txt
```

Rewrite a path in the portable `gnux shell`:

```bash
gnux shell -c "echox /usr/local/bin | sedx 's|/usr/local|/opt|'"
```

```powershell
gnux shell -c "echox /usr/local/bin | sedx 's|/usr/local|/opt|'"
```

Output:

```text
/opt/bin
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success |
| 1 | an invalid command, syntax, regular expression or option |
| 2 | an input file could not be opened; the others were still processed |
| 4 | an I/O error while running: `sedx` stopped at once |
| N | the exit status given to the q or Q command |

### Help in the terminal

```bash
sedx --help                # options, first examples, exit status
gnux help sedx             # this page: the full manual
gnux help sedx --OPTION    # one option in depth, with its examples
```

### See also

- [trx](../trx/index.md)
- [cutx](../cutx/index.md)
- [headx](../headx/index.md)
- [tailx](../tailx/index.md)
- [csplitx](../csplitx/index.md)
- [gnux shell](../shell/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
