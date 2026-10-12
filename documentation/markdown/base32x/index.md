## base32x

Encode or decode data in `base32`.

Mirrors GNU `base32`. Run it as `base32x`, or as `gnux base32x`.

### Synopsis

```bash
base32x [OPTION]... [FILE]
```

### GNU reference

- [GNU coreutils manual: base32 invocation](https://www.gnu.org/software/coreutils/manual/html_node/base32-invocation.html)
- [Source: src/basenc.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/basenc.c)

### Description

`base32x` encodes FILE in `base32` (RFC 4648, with padding) and writes the result to standard output, or with `-d` decodes it. With no FILE, or when FILE is -, it reads standard input; at most one FILE is accepted. Encoded output is wrapped every 76 characters and ends with a newline; `-w` COLS wraps at COLS instead, and `-w` 0 writes one line with no newline at all. When decoding, newlines are ignored, and `-i` also ignores every other character outside the alphabet; any other stray character, or a truncated input, stops decoding with 'invalid input' and exit status 1, after the data decoded so far has been written.

### Permissions

Needs read permission on FILE and search permission on the directories leading to it. Nothing is ever modified.

### Use cases

- Embedding binary data such as images or keys in text formats like JSON or YAML.
- Decoding `base32` found in configuration, tokens or email attachments.
- Passing binary data through channels that only carry text.
- Producing single-line encodings for environment variables with `-w` 0.
- Inspecting encoded payloads in a pipeline.

### Flags

```text
  -d, --decode           decode data
      --help             display this help and exit
  -i, --ignore-garbage   when decoding, ignore non-alphabet characters
      --version          output version information and exit
  -w, --wrap COLS        wrap encoded lines after COLS character (default 76); use 0 to disable line wrapping
```

### Examples

Encode a file:

```bash
base32x image.png
```

```powershell
base32x image.png
```

Encode a string:

```bash
printfx 'hello' | base32x
```

```powershell
printfx 'hello' | base32x
```

Decode a string:

```bash
echox NBSWY3DP | base32x -d
```

```powershell
echox NBSWY3DP | base32x -d
```

Decode a file back to binary:

```bash
base32x -d image.txt > image.png
```

```powershell
base32x -d image.txt > image.png
```

Encode on a single line, without a newline:

```bash
base32x -w 0 key.bin
```

```powershell
base32x -w 0 key.bin
```

Wrap encoded lines at 64 characters:

```bash
base32x -w 64 cert.der
```

```powershell
base32x -w 64 cert.der
```

Decode, skipping characters outside the alphabet:

```bash
base32x -d -i messy.txt
```

```powershell
base32x -d -i messy.txt
```

Store an encoded file in a variable:

```bash
data=$(base32x -w 0 key.bin)
```

```powershell
$data = base32x -w 0 key.bin
```

Round-trip a file and compare the checksums:

```bash
base32x file.bin | base32x -d > copy.bin && cksumx file.bin copy.bin
```

```powershell
base32x file.bin | base32x -d > copy.bin; cksumx file.bin copy.bin
```

Check that input decodes cleanly:

```bash
base32x -d payload.txt > /dev/null && echox valid
```

```powershell
base32x -d payload.txt > $null; if ($LASTEXITCODE -eq 0) { 'valid' }
```

Encode in the portable `gnux shell`:

```bash
gnux shell -c 'base32x -w 0 key.bin > key.txt'
```

```powershell
gnux shell -c 'base32x -w 0 key.bin > key.txt'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
base32x --help                # options, first examples, exit status
gnux help base32x             # this page: the full manual
gnux help base32x --OPTION    # one option in depth, with its examples
```

### See also

- [base64x](../base64x/index.md)
- [odx](../odx/index.md)
- [cksumx](../cksumx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
