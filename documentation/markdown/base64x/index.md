## base64x

Encode or decode data in `base64`.

Mirrors GNU `base64`. Run it as `base64x`, or as `gnux base64x`.

### Synopsis

```bash
base64x [OPTION]... [FILE]
```

### GNU reference

- [GNU coreutils manual: base64 invocation](https://www.gnu.org/software/coreutils/manual/html_node/base64-invocation.html)
- [Source: src/basenc.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/basenc.c)

### Description

`base64x` encodes FILE in `base64` (RFC 4648, with padding) and writes the result to standard output, or with `-d` decodes it. With no FILE, or when FILE is -, it reads standard input; at most one FILE is accepted. Encoded output is wrapped every 76 characters and ends with a newline; `-w` COLS wraps at COLS instead, and `-w` 0 writes one line with no newline at all. When decoding, newlines are ignored, and `-i` also ignores every other character outside the alphabet; any other stray character, or a truncated input, stops decoding with 'invalid input' and exit status 1, after the data decoded so far has been written.

### Permissions

Needs read permission on FILE and search permission on the directories leading to it. Nothing is ever modified.

### Use cases

- Embedding binary data such as images or keys in text formats like JSON or YAML.
- Decoding `base64` found in configuration, tokens or email attachments.
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
base64x image.png
```

```powershell
base64x image.png
```

Encode a string:

```bash
printfx 'hello' | base64x
```

```powershell
printfx 'hello' | base64x
```

Decode a string:

```bash
echox aGVsbG8= | base64x -d
```

```powershell
echox aGVsbG8= | base64x -d
```

Decode a file back to binary:

```bash
base64x -d image.txt > image.png
```

```powershell
base64x -d image.txt > image.png
```

Encode on a single line, without a newline:

```bash
base64x -w 0 key.bin
```

```powershell
base64x -w 0 key.bin
```

Wrap encoded lines at 64 characters:

```bash
base64x -w 64 cert.der
```

```powershell
base64x -w 64 cert.der
```

Decode, skipping characters outside the alphabet:

```bash
base64x -d -i messy.txt
```

```powershell
base64x -d -i messy.txt
```

Store an encoded file in a variable:

```bash
data=$(base64x -w 0 key.bin)
```

```powershell
$data = base64x -w 0 key.bin
```

Round-trip a file and compare the checksums:

```bash
base64x file.bin | base64x -d > copy.bin && cksumx file.bin copy.bin
```

```powershell
base64x file.bin | base64x -d > copy.bin; cksumx file.bin copy.bin
```

Check that input decodes cleanly:

```bash
base64x -d payload.txt > /dev/null && echox valid
```

```powershell
base64x -d payload.txt > $null; if ($LASTEXITCODE -eq 0) { 'valid' }
```

Encode in the portable `gnux shell`:

```bash
gnux shell -c 'base64x -w 0 key.bin > key.txt'
```

```powershell
gnux shell -c 'base64x -w 0 key.bin > key.txt'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
base64x --help                # options, first examples, exit status
gnux help base64x             # this page: the full manual
gnux help base64x --OPTION    # one option in depth, with its examples
```

### See also

- [base32x](../base32x/index.md)
- [odx](../odx/index.md)
- [cksumx](../cksumx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
