## sha512sumx

Print or check SHA-512 (512-bit) checksums.

Mirrors GNU `sha512sum`. Run it as `sha512sumx`, or as `gnux sha512sumx`.

### Synopsis

```bash
sha512sumx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: sha512sum invocation](https://www.gnu.org/software/coreutils/manual/html_node/sha2-utilities.html)
- [Source: src/cksum.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/cksum.c)

### Description

`sha512sumx` prints or checks SHA-512 message digests (FIPS 180-4, 512 bits). For each FILE it prints one line: the digest in hexadecimal, a space, a mode mark (a space for text mode, '*' with `-b`), and the file name. With no FILE, or when FILE is -, it reads standard input. Files are always read as bytes, so `-b` and `-t` change only the mark; when both are given, `-t` wins. `--tag` prints BSD-style lines instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a backslash, newline or carriage return is escaped and its line starts with a backslash, unless `-z` ends lines with NUL instead. `-c` reads such lines (in either style) from the FILEs and checks each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning of mismatches, unreadable files and improperly formatted lines; `--quiet` drops the OK lines, `--status` prints nothing, `--ignore-missing` skips listed files that do not exist, `-w` warns about each improperly formatted line and `--strict` fails because of them. The exit status is 0 when everything was read (and, with `-c`, matched), 1 otherwise.

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
sha512sumx file.iso
```

```powershell
sha512sumx file.iso
```

Digest several files:

```bash
sha512sumx *.tar.gz
```

```powershell
sha512sumx (Get-ChildItem *.tar.gz -Name)
```

Digest standard input:

```bash
echox -n hello | sha512sumx
```

```powershell
echox -n hello | sha512sumx
```

Save digests to a file:

```bash
sha512sumx *.iso > sha512.sums
```

```powershell
sha512sumx (Get-ChildItem *.iso -Name) > sha512.sums
```

Verify files against saved digests:

```bash
sha512sumx -c sha512.sums
```

```powershell
sha512sumx -c sha512.sums
```

Verify quietly, showing only failures:

```bash
sha512sumx -c --quiet sha512.sums
```

```powershell
sha512sumx -c --quiet sha512.sums
```

Verify silently and branch on the result:

```bash
sha512sumx -c --status sha512.sums && echox verified
```

```powershell
sha512sumx -c --status sha512.sums; if ($LASTEXITCODE -eq 0) { 'verified' }
```

Verify only the listed files that exist:

```bash
sha512sumx -c --ignore-missing sha512.sums
```

```powershell
sha512sumx -c --ignore-missing sha512.sums
```

Fail on malformed lines in the list:

```bash
sha512sumx -c --strict -w sha512.sums
```

```powershell
sha512sumx -c --strict -w sha512.sums
```

Print BSD-style tagged lines:

```bash
sha512sumx --tag file.iso
```

```powershell
sha512sumx --tag file.iso
```

Mark the files as read in binary mode:

```bash
sha512sumx -b file.iso
```

```powershell
sha512sumx -b file.iso
```

Digest in the portable `gnux shell`:

```bash
gnux shell -c 'sha512sumx file.iso > file.iso.sha512sum'
```

```powershell
gnux shell -c 'sha512sumx file.iso > file.iso.sha512sum'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
sha512sumx --help                # options, first examples, exit status
gnux help sha512sumx             # this page: the full manual
gnux help sha512sumx --OPTION    # one option in depth, with its examples
```

### See also

- [cksumx](../cksumx/index.md)
- [md5sumx](../md5sumx/index.md)
- [sha1sumx](../sha1sumx/index.md)
- [sha256sumx](../sha256sumx/index.md)
- [b2sumx](../b2sumx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
