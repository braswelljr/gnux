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

You downloaded an installer: print its digest to compare with the one on the website:

```bash
sha512sumx ubuntu-24.04-desktop-amd64.iso
```

```powershell
sha512sumx ubuntu-24.04-desktop-amd64.iso
```

Verify the download against the publisher's checksum file, which lists other files too:

```bash
sha512sumx -c --ignore-missing sha512.sums
```

```powershell
sha512sumx -c --ignore-missing sha512.sums
```

Output:

```text
ubuntu-24.04-desktop-amd64.iso: OK
```

Publish checksums next to your release archives:

```bash
sha512sumx dist/*.tar.gz > dist/sha512.sums
```

```powershell
sha512sumx (Get-ChildItem dist\*.zip -Name) > dist\sha512.sums
```

Check that a backup copied to a USB drive still matches the original:

```bash
sha512sumx photos.tar /media/usb/photos.tar
```

```powershell
sha512sumx photos.tar E:\photos.tar
```

Find duplicate photos in a folder: files with the same digest have the same content:

```bash
sha512sumx *.jpg | sortx | uniqx -w 128 -D
```

```powershell
sha512sumx (Get-ChildItem *.jpg -Name) | sortx | uniqx -w 128 -D
```

In CI, fail the build when a vendored file was changed by hand:

```bash
sha512sumx -c --status vendor.sha512sums || { echox 'vendored files changed' >&2; exit 1; }
```

```powershell
sha512sumx -c --status vendor.sha512sums; if ($LASTEXITCODE -ne 0) { Write-Error 'vendored files changed'; exit 1 }
```

Re-check a large archive set and show only what no longer matches:

```bash
sha512sumx -c --quiet sha512.sums
```

```powershell
sha512sumx -c --quiet sha512.sums
```

Output:

```text
notes.txt: FAILED
sha512sumx: WARNING: 1 computed checksum did NOT match
```

Hash a token from a variable, without the newline echo would add:

```bash
printfx '%s' "$TOKEN" | sha512sumx
```

```powershell
printfx '%s' $env:TOKEN | sha512sumx
```

Write lines that name the algorithm, so a mixed list can be checked with `cksumx -c`:

```bash
sha512sumx --tag release.tar.gz >> CHECKSUMS
```

```powershell
sha512sumx --tag release.zip >> CHECKSUMS
```

Refuse a checksum list with malformed lines instead of skipping them:

```bash
sha512sumx -c --strict -w sha512.sums
```

```powershell
sha512sumx -c --strict -w sha512.sums
```

Match the '*' binary-mode lines produced by tools on Windows:

```bash
sha512sumx -b setup.exe
```

```powershell
sha512sumx -b setup.exe
```

Record a digest next to the file in the portable `gnux shell`, the same on every OS:

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
