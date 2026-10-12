## odx

Dump files in octal and other formats.

Mirrors GNU `od`. Run it as `odx`, or as `gnux odx`.

### Synopsis

```bash
odx [OPTION]... [FILE]...
```

### GNU reference

- [GNU coreutils manual: od invocation](https://www.gnu.org/software/coreutils/manual/html_node/od-invocation.html)
- [Source: src/od.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/od.c)

### Description

`odx` writes an unambiguous representation of its input, by default as two-byte octal words, sixteen bytes per line, each line starting with the input offset in octal. The FILEs are read one after another as one input; with no FILE, or when FILE is -, it reads standard input. `-t` TYPE selects the format and may be given more than once, each format getting a line of its own per block, in the order given: a (named characters), c (printable characters or backslash escapes), d, o, u and x (signed decimal, octal, unsigned decimal and hexadecimal) with an optional size in bytes (1, 2, 4, 8, or C, S, I, L), and f (floating point, size 4 or 8, or F, D, L); a trailing z adds the printable characters of the line between > and <. The traditional letters `-a`, `-b`, `-c`, `-d`, `-f`, `-i`, `-l`, `-o`, `-s` and `-x` are shorthands for `-t` a, o1, c, u2, fF, dI, dL, o2, d2 and x2. `-A` RADIX picks the offset base (d, o, x, or n for none), `-j` BYTES skips input, `-N` BYTES limits it, `-w` BYTES sets the bytes per line, and `--endian` chooses the byte order of multi-byte values. Unless `-v` is given, a run of lines identical to the one before is shown as a single '*'. `-S` N prints the NUL-terminated strings of at least N printable characters instead of a dump. BYTES may be decimal, octal with a leading 0, or hexadecimal with 0x, followed by a multiplier: b 512, k or K 1024, kB 1000, M, MB, G and so on. Differences from GNU `od`: characters are interpreted in the C locale, so `-c` shows every byte above 127 in octal; long double (fL) is read as an eight-byte double; `-S` and `-w` need a value (GNU's bare `-S` means `-S3` and bare `-w` means `-w32`), because an optional value could not also be attached as in `-w8`; the traditional offset operand ([+]OFFSET[.][b]) and `--traditional` are not implemented.

### Permissions

Needs read permission on each FILE and search permission on the directories leading to it. Nothing is ever modified.

### Use cases

- Seeing exactly which bytes a file holds, including invisible ones.
- Finding stray carriage returns, tabs or NUL bytes with `-c`.
- Reading binary headers as hexadecimal bytes or words with `-t` x1 or `-t` x2.
- Checking a file's byte order or a magic number with `--endian` and `-N`.
- Decoding integers or floating-point values stored in a binary file.
- Getting a hexdump-style view with `-A` x `-t` x1z.
- Extracting printable strings from a binary with `-S`.
- Dumping just a slice of a large file with `-j` and `-N`.

### Flags

```text
  -A, --address-radix RADIX                        output format for file offsets; RADIX is one of [doxn], for Decimal, Octal, Hex or None (default "o")
  -c, --characters string[="true"]                 same as -t c, select printable characters or backslash escapes
  -s, --decimal-2-bytes string[="true"]            same as -t d2, select decimal 2-byte units
  -i, --decimal-ints string[="true"]               same as -t dI, select decimal ints
  -l, --decimal-longs string[="true"]              same as -t dL, select decimal longs
      --endian ORDER                               swap input bytes according the specified ORDER: little or big
  -f, --floats string[="true"]                     same as -t fF, select floats
  -t, --format TYPE                                select output format or formats; TYPE is a, c, d, f, o, u or x with an optional size, and z
      --help                                       display this help and exit
  -x, --hex-2-bytes string[="true"]                same as -t x2, select hexadecimal 2-byte units
  -a, --named-characters string[="true"]           same as -t a, select named characters, ignoring high-order bit
  -o, --octal-2-bytes string[="true"]              same as -t o2, select octal 2-byte units
  -b, --octal-bytes string[="true"]                same as -t o1, select octal bytes
  -v, --output-duplicates                          do not use * to mark line suppression
  -N, --read-bytes BYTES                           limit dump to BYTES input bytes
  -j, --skip-bytes BYTES                           skip BYTES input bytes first
  -S, --strings BYTES                              show only NUL terminated strings of at least BYTES printable characters
  -d, --unsigned-decimal-2-bytes string[="true"]   same as -t u2, select unsigned decimal 2-byte units
      --version                                    output version information and exit
  -w, --width BYTES                                output BYTES bytes per output line
```

### Examples

Dump a file as octal words:

```bash
odx data.bin
```

```powershell
odx data.bin
```

Show every character, with escapes for invisible ones:

```bash
odx -c notes.txt
```

```powershell
odx -c notes.txt
```

A hexdump with offsets in hex and the text alongside:

```bash
odx -A x -t x1z image.png
```

```powershell
odx -A x -t x1z image.png
```

Read the first four bytes (a magic number):

```bash
odx -A n -t x1 -N 4 image.png
```

```powershell
odx -A n -t x1 -N 4 image.png
```

Skip a 512-byte header and dump the next 64 bytes:

```bash
odx -j 512 -N 64 -t x1 disk.img
```

```powershell
odx -j 512 -N 64 -t x1 disk.img
```

Decode big-endian 32-bit integers:

```bash
odx --endian=big -t u4 counts.bin
```

```powershell
odx --endian=big -t u4 counts.bin
```

Decode floating-point values:

```bash
odx -t f8 samples.bin
```

```powershell
odx -t f8 samples.bin
```

Two formats per line, hex bytes over characters:

```bash
odx -t x1 -t c file.txt
```

```powershell
odx -t x1 -t c file.txt
```

Look for Windows line endings in standard input:

```bash
printfx 'a\r\nb\n' | odx -c
```

```powershell
'a' | odx -c
```

Print every line, even repeated ones:

```bash
odx -v -t x1 zeros.bin
```

```powershell
odx -v -t x1 zeros.bin
```

Eight bytes per line:

```bash
odx -w8 -t x1 data.bin
```

```powershell
odx -w8 -t x1 data.bin
```

List the printable strings of at least 6 characters in a binary:

```bash
odx -S 6 program.bin
```

```powershell
odx -S 6 program.exe
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
odx --help                # options, first examples, exit status
gnux help odx             # this page: the full manual
gnux help odx --OPTION    # one option in depth, with its examples
```

### See also

- [catx](../catx/index.md)
- [headx](../headx/index.md)
- [wcx](../wcx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
