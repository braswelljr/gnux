## sha256sumx

Print or check SHA-256 (256-bit) checksums.

Mirrors GNU `sha256sum`. Run it as `sha256sumx`, or as `gnux sha256sumx`.

### Synopsis

```bash
sha256sumx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: sha256sum invocation](https://www.gnu.org/software/coreutils/manual/html_node/sha2-utilities.html)
- [Source: src/cksum.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/cksum.c)

### Description

`sha256sumx` prints or checks SHA-256 message digests (FIPS 180-4, 256 bits). For each FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text mode, '*' with `-b`), and the file name. With no FILE, or when FILE is -, it reads standard input. Files are always read as bytes, so `-b` and `-t` change only the mark; when both are given, `-t` wins. `--tag` prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a backslash, newline or carriage return is escaped and its line starts with a backslash, unless `-z` ends lines with NUL instead. `-c` reads such lines (in either style) from the FILEs and checks each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches, unreadable files and improperly formatted lines; `--quiet` drops the OK lines, `--status` prints nothing, `--ignore-missing` skips listed files that do not exist, `-w` warns about each improperly formatted line and `--strict` fails because of them. The exit status is 0 when everything was read (and, with `-c`, matched), 1 otherwise.

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
sha256sumx file.iso
```

```powershell
sha256sumx file.iso
```

Digest several files:

```bash
sha256sumx *.tar.gz
```

```powershell
sha256sumx (Get-ChildItem *.tar.gz -Name)
```

Digest standard input:

```bash
echox -n hello | sha256sumx
```

```powershell
echox -n hello | sha256sumx
```

Save digests to a file:

```bash
sha256sumx *.iso > sha256.sums
```

```powershell
sha256sumx (Get-ChildItem *.iso -Name) > sha256.sums
```

Verify files against saved digests:

```bash
sha256sumx -c sha256.sums
```

```powershell
sha256sumx -c sha256.sums
```

Verify quietly, showing only failures:

```bash
sha256sumx -c --quiet sha256.sums
```

```powershell
sha256sumx -c --quiet sha256.sums
```

Verify silently and branch on the result:

```bash
sha256sumx -c --status sha256.sums && echox verified
```

```powershell
sha256sumx -c --status sha256.sums; if ($LASTEXITCODE -eq 0) { 'verified' }
```

Verify only the listed files that exist:

```bash
sha256sumx -c --ignore-missing sha256.sums
```

```powershell
sha256sumx -c --ignore-missing sha256.sums
```

Fail on malformed lines in the list:

```bash
sha256sumx -c --strict -w sha256.sums
```

```powershell
sha256sumx -c --strict -w sha256.sums
```

Print BSD-style tagged lines:

```bash
sha256sumx --tag file.iso
```

```powershell
sha256sumx --tag file.iso
```

Mark the files as read in binary mode:

```bash
sha256sumx -b file.iso
```

```powershell
sha256sumx -b file.iso
```

Digest in the portable `gnux shell`:

```bash
gnux shell -c 'sha256sumx file.iso > file.iso.sha256sum'
```

```powershell
gnux shell -c 'sha256sumx file.iso > file.iso.sha256sum'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
sha256sumx --help                # options, first examples, exit status
gnux help sha256sumx             # this page: the full manual
gnux help sha256sumx --OPTION    # one option in depth, with its examples
```

### See also

- [cksumx](../cksumx/index.md)
- [md5sumx](../md5sumx/index.md)
- [sha1sumx](../sha1sumx/index.md)
- [sha512sumx](../sha512sumx/index.md)
- [b2sumx](../b2sumx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
