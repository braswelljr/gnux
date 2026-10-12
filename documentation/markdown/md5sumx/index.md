## md5sumx

Print or check MD5 (128-bit) checksums.

Mirrors GNU `md5sum`. Run it as `md5sumx`, or as `gnux md5sumx`.

### Synopsis

```bash
md5sumx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: md5sum invocation](https://www.gnu.org/software/coreutils/manual/html_node/md5sum-invocation.html)
- [Source: src/cksum.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/cksum.c)

### Description

`md5sumx` prints or checks MD5 message digests (RFC 1321, 128 bits). For each FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text mode, '*' with `-b`), and the file name. With no FILE, or when FILE is -, it reads standard input. Files are always read as bytes, so `-b` and `-t` change only the mark; when both are given, `-t` wins. `--tag` prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a backslash, newline or carriage return is escaped and its line starts with a backslash, unless `-z` ends lines with NUL instead. `-c` reads such lines (in either style) from the FILEs and checks each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches, unreadable files and improperly formatted lines; `--quiet` drops the OK lines, `--status` prints nothing, `--ignore-missing` skips listed files that do not exist, `-w` warns about each improperly formatted line and `--strict` fails because of them. The exit status is 0 when everything was read (and, with `-c`, matched), 1 otherwise. MD5 is broken for security purposes: use it to detect accidental corruption, not tampering, and prefer `sha256sumx` or `b2sumx` otherwise.

### Permissions

Needs read permission on each FILE (and, with `-c`, on the checksum lists and every file they name) and search permission on the directories leading to them. Nothing is ever modified.

### Use cases

- Verifying a download against the checksum its publisher lists.
- Recording digests of release artifacts so users can check them.
- Detecting whether files changed between two points in time.
- Finding duplicate files by comparing their digests.
- Checking a backup or a copy against the original.

### Flags

```text
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
```

### Examples

Print the digest of a file:

```bash
md5sumx file.iso
```

```powershell
md5sumx file.iso
```

Digest several files:

```bash
md5sumx *.tar.gz
```

```powershell
md5sumx (Get-ChildItem *.tar.gz -Name)
```

Digest standard input:

```bash
echox -n hello | md5sumx
```

```powershell
echox -n hello | md5sumx
```

Save digests to a file:

```bash
md5sumx *.iso > md5.sums
```

```powershell
md5sumx (Get-ChildItem *.iso -Name) > md5.sums
```

Verify files against saved digests:

```bash
md5sumx -c md5.sums
```

```powershell
md5sumx -c md5.sums
```

Verify quietly, showing only failures:

```bash
md5sumx -c --quiet md5.sums
```

```powershell
md5sumx -c --quiet md5.sums
```

Verify silently and branch on the result:

```bash
md5sumx -c --status md5.sums && echox verified
```

```powershell
md5sumx -c --status md5.sums; if ($LASTEXITCODE -eq 0) { 'verified' }
```

Verify only the listed files that exist:

```bash
md5sumx -c --ignore-missing md5.sums
```

```powershell
md5sumx -c --ignore-missing md5.sums
```

Fail on malformed lines in the list:

```bash
md5sumx -c --strict -w md5.sums
```

```powershell
md5sumx -c --strict -w md5.sums
```

Print BSD-style tagged lines:

```bash
md5sumx --tag file.iso
```

```powershell
md5sumx --tag file.iso
```

Mark the files as read in binary mode:

```bash
md5sumx -b file.iso
```

```powershell
md5sumx -b file.iso
```

Digest in the portable `gnux shell`:

```bash
gnux shell -c 'md5sumx file.iso > file.iso.md5sum'
```

```powershell
gnux shell -c 'md5sumx file.iso > file.iso.md5sum'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
md5sumx --help                # options, first examples, exit status
gnux help md5sumx             # this page: the full manual
gnux help md5sumx --OPTION    # one option in depth, with its examples
```

### See also

- [cksumx](../cksumx/index.md)
- [sha1sumx](../sha1sumx/index.md)
- [sha256sumx](../sha256sumx/index.md)
- [sha512sumx](../sha512sumx/index.md)
- [b2sumx](../b2sumx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
