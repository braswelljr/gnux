## trx

Translate, squeeze, and/or delete characters from standard input.

Mirrors GNU `tr`. Run it as `trx`, or as `gnux trx`.

### Synopsis

```bash
trx [OPTION]... STRING1 [STRING2]
```

### Description

`trx` copies standard input to standard output, translating each byte of SET1 into the byte at the same position in SET2. SET2 is padded with its last byte to the length of SET1, unless `-t` truncates SET1 to the length of SET2 instead; when a byte appears more than once in SET1, its last mapping wins. `-d` deletes the bytes of SET1, and `-s` squeezes each run of a repeated byte from the last SET given into one; with `-d` and `-s` together, SET1 is deleted and SET2 squeezed. `-c` (or `-C`) uses the complement of SET1: every byte not in it, in ascending order. A SET is a string of bytes with these constructs: \NNN (an octal byte), \\, \a, \b, \f, \n, \r, \t, \v; CHAR1-CHAR2 for the bytes from CHAR1 to CHAR2 in ascending order; [CHAR*] in SET2 for as many copies of CHAR as make SET2 as long as SET1, and [CHAR*REPEAT] for REPEAT copies (octal if it starts with 0); [:CLASS:] for alnum, alpha, blank, cntrl, digit, graph, lower, print, punct, space, upper or xdigit; and [=CHAR=] for the bytes equivalent to CHAR. When translating, only [:lower:] and [:upper:] may appear in SET2, and only opposite a case class in SET1, so that letters map to their other case. `trx` reads only standard input. Like GNU `tr`, it works on bytes, not characters: a multibyte UTF-8 character is several separate bytes to it, and the classes are those of the C locale (ASCII).

### Permissions

Needs no file permissions: `trx` reads standard input and writes standard output. Redirect a file into it to read one; the shell, not `trx`, opens it.

### Use cases

- Changing the case of text with '[:lower:]' '[:upper:]'.
- Converting Windows line endings by deleting carriage returns with `-d` '\r'.
- Squeezing runs of spaces or blank lines into one with `-s`.
- Splitting text into one word per line with `-cs` '[:alnum:]' '\n'.
- Removing digits, punctuation or control characters from input with `-d`.
- Turning a PATH-style list into lines by translating ':' to '\n'.
- Replacing NUL separators with newlines to read find `-print0` output.
- Applying a simple substitution cipher such as ROT13.

### Flags

```text
  -c, --complement         use the complement of SET1
  -C, --complement-upper   equivalent to -c
  -d, --delete             delete characters in SET1, do not translate
      --help               display this help and exit
  -s, --squeeze-repeats    replace each sequence of a repeated character that is listed in the last specified SET, with a single occurrence of that character
  -t, --truncate-set1      first truncate SET1 to length of SET2
      --version            output version information and exit
```

### Examples

Convert text to upper case:

```bash
echox 'hello world' | trx '[:lower:]' '[:upper:]'
```

```powershell
'hello world' | trx '[:lower:]' '[:upper:]'
```

Convert text to lower case with ranges:

```bash
echox 'Hello World' | trx A-Z a-z
```

```powershell
'Hello World' | trx A-Z a-z
```

Strip Windows carriage returns from a file:

```bash
trx -d '\r' < dos.txt > unix.txt
```

```powershell
Get-Content -Raw dos.txt | trx -d '\r' | Set-Content -NoNewline unix.txt
```

Squeeze runs of spaces into one:

```bash
echox 'a    b     c' | trx -s ' '
```

```powershell
'a    b     c' | trx -s ' '
```

Delete every digit:

```bash
echox 'r2d2 c3po' | trx -d '[:digit:]'
```

```powershell
'r2d2 c3po' | trx -d '[:digit:]'
```

Put each word on a line of its own:

```bash
catx essay.txt | trx -cs '[:alnum:]' '\n'
```

```powershell
catx essay.txt | trx -cs '[:alnum:]' '\n'
```

List the directories of PATH, one per line:

```bash
echox "$PATH" | trx : '\n'
```

```powershell
$env:PATH | trx ';' '\n'
```

Keep only the printable characters and newlines:

```bash
trx -cd '[:print:]\n' < binary.log
```

```powershell
catx binary.log | trx -cd '[:print:]\n'
```

Map a set onto a shorter one without padding it (`-t`):

```bash
echox abcde | trx -t a-e xy
```

```powershell
'abcde' | trx -t a-e xy
```

Replace every vowel with a star, using a repeat:

```bash
echox 'programming' | trx aeiou '[**]'
```

```powershell
'programming' | trx aeiou '[**]'
```

Encode text with ROT13:

```bash
echox 'Hello' | trx A-Za-z N-ZA-Mn-za-m
```

```powershell
'Hello' | trx A-Za-z N-ZA-Mn-za-m
```

Delete digits and squeeze the spaces left behind:

```bash
echox 'a 1 2 b' | trx -ds '[:digit:]' ' '
```

```powershell
'a 1 2 b' | trx -ds '[:digit:]' ' '
```

Normalize text in the portable `gnux shell`:

```bash
gnux shell -c "catx names.txt | trx '[:upper:]' '[:lower:]' > lower.txt"
```

```powershell
gnux shell -c "catx names.txt | trx '[:upper:]' '[:lower:]' > lower.txt"
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
trx --help                # options, first examples, exit status
gnux help trx             # this page: the full manual
gnux help trx --OPTION    # one option in depth, with its examples
```

### See also

- [catx](../catx/index.md)
- [wcx](../wcx/index.md)
- [echox](../echox/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
