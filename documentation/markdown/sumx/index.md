## sumx

Print the checksum and block count of each FILE.

Mirrors GNU `sum`. Run it as `sumx`, or as `gnux sumx`.

### Synopsis

```bash
sumx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: sum invocation](https://www.gnu.org/software/coreutils/manual/html_node/sum-invocation.html)
- [Source: src/sum.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/sum.c)

### Description

`sumx` prints a 16-bit checksum of each FILE and the number of blocks it takes up, followed by its name when FILE operands are given, as GNU `sum` 9 does. With no FILE, or when FILE is -, it reads standard input. By default (and with `-r`) it computes the BSD checksum, printed with five digits, and counts 1024-byte blocks; `-s` (`--sysv`) computes the System V checksum and counts 512-byte blocks. Both are the checksums `cksumx` prints with `-a` bsd and `-a` sysv. They exist to compare with old systems' output and catch accidental corruption only: use `sha256sumx` to verify a download. GNU lets the last of `-r` and `-s` win; `gnux` cannot see their order, so `-s` wins whenever it is given. A FILE that cannot be read is reported and the rest are still summed.

### Permissions

Needs read permission on each FILE. Writes nothing.

### Use cases

- Comparing a file with a checksum published by an old Unix system or a legacy manual.
- Quickly spotting that two copies of a file differ.
- Reproducing the output of `sum` in a script that expects it.
- Checking System V checksums with `-s`.
- Getting a file's size in kilobyte blocks along with its checksum.

### Flags

```text
      --help      display this help and exit
  -s, --sysv      use the System V sum algorithm, use 512-byte blocks
      --version   output version information and exit
```

### Examples

Print the BSD checksum and 1K-block count of a file:

```bash
sumx README.md
```

```powershell
sumx README.md
```

Output:

```text
36979     1 README.md
```

Print the System V checksum and 512-byte block count:

```bash
sumx -s README.md
```

```powershell
sumx -s README.md
```

Checksum several files at once:

```bash
sumx *.txt
```

```powershell
sumx (Get-ChildItem *.txt).Name
```

Checksum standard input:

```bash
printfx 'hello\n' | sumx
```

```powershell
printfx 'hello\n' | sumx
```

Output:

```text
36979     1
```

Compare two copies of a file:

```bash
[ "$(sumx < a.bin)" = "$(sumx < b.bin)" ] && echox same
```

```powershell
if ((Get-Content a.bin -Raw | sumx) -eq (Get-Content b.bin -Raw | sumx)) { 'same' }
```

Ask for the BSD algorithm explicitly:

```bash
sumx -r archive.tar
```

```powershell
sumx -r archive.tar
```

Mix standard input with files:

```bash
catx header.txt | sumx - body.txt
```

```powershell
catx header.txt | sumx - body.txt
```

Keep a checksum list for later comparison:

```bash
sumx -s *.cfg > sums.txt
```

```powershell
sumx -s (Get-ChildItem *.cfg).Name > sums.txt
```

Get the same checksum through `cksumx`:

```bash
cksumx -a bsd README.md
```

```powershell
cksumx -a bsd README.md
```

Checksum every file under a directory in the portable `gnux shell`:

```bash
gnux shell -c 'findx . -type f | xargsx sumx'
```

```powershell
gnux shell -c 'findx . -type f | xargsx sumx'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
sumx --help                # options, first examples, exit status
gnux help sumx             # this page: the full manual
gnux help sumx --OPTION    # one option in depth, with its examples
```

### See also

- [cksumx](../cksumx/index.md)
- [md5sumx](../md5sumx/index.md)
- [sha256sumx](../sha256sumx/index.md)
- [b2sumx](../b2sumx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
