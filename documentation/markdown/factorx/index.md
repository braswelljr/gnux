## factorx

Print the prime factors of each specified integer NUMBER.

Mirrors GNU `factor`. Run it as `factorx`, or as `gnux factorx`.

### Synopsis

```bash
factorx [OPTION] [NUMBER]...
```

### GNU reference

- [GNU coreutils manual: factor invocation](https://www.gnu.org/software/coreutils/manual/html_node/factor-invocation.html)
- [Source: src/factor.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/factor.c)

### Description

`factorx` prints each NUMBER followed by a colon and its prime factors in ascending order, a repeated `factor` as many times as it divides the number. With no NUMBER, it reads numbers separated by spaces, tabs and newlines from standard input, and answers each line as soon as it is read. 0 and 1 have no prime factors and print just the colon. A NUMBER may have leading spaces and one '+', and must otherwise be decimal digits; anything else is reported as 'is not a valid positive integer', the remaining numbers are still factored, and the exit status is 1. `-h` (`--exponents`) prints a repeated `factor` once, as p^e. As in GNU `factor`, a NUMBER has no size limit. Numbers below 2^64 are factored with trial division, the Miller-Rabin `test` and Pollard's rho, and take microseconds; larger ones use arbitrary-precision arithmetic, which is fast when the number has at most one large prime `factor` and can take minutes when it is the product of two primes of more than 15 digits each.

### Permissions

Needs no permissions. Reads only its operands or standard input.

### Use cases

- Checking whether a number is prime: a prime's only `factor` is itself.
- Finding the divisors of a number in a script or a puzzle.
- Simplifying a fraction or finding a common divisor by hand.
- Teaching or checking arithmetic exercises.
- Generating `test` data with known factorizations.

### Flags

```text
  -h, --exponents   print repeated factors in form p^e unless e is 1
      --help        display this help and exit
      --version     output version information and exit
```

### Examples

Factor a number:

```bash
factorx 360
```

```powershell
factorx 360
```

Output:

```text
360: 2 2 2 3 3 5
```

Show repeated factors as powers:

```bash
factorx -h 360
```

```powershell
factorx -h 360
```

Output:

```text
360: 2^3 3^2 5
```

Factor several numbers at once:

```bash
factorx 12 97 1001
```

```powershell
factorx 12 97 1001
```

Output:

```text
12: 2 2 3
97: 97
1001: 7 11 13
```

Check whether a number is prime:

```bash
[ "$(factorx 1000000007 | cutx -d' ' -f2-)" = 1000000007 ] && echox prime
```

```powershell
if ((factorx 1000000007) -eq '1000000007: 1000000007') { 'prime' }
```

Factor numbers read from standard input:

```bash
seqx 10 15 | factorx
```

```powershell
seqx 10 15 | factorx
```

List the primes up to 50:

```bash
seqx 2 50 | factorx | grepx -E '^([0-9]+): \1$' | cutx -d: -f1
```

```powershell
seqx 2 50 | factorx | grepx -E '^([0-9]+): \1$' | cutx -d: -f1
```

Factor the largest 64-bit unsigned number:

```bash
factorx 18446744073709551615
```

```powershell
factorx 18446744073709551615
```

Output:

```text
18446744073709551615: 3 5 17 257 641 65537 6700417
```

Factor a number larger than 64 bits:

```bash
factorx 1234567890123456789012345678901234567890
```

```powershell
factorx 1234567890123456789012345678901234567890
```

Count the prime factors of a number:

```bash
factorx 1024 | trx ' ' '\n' | tailx -n +2 | wcx -l
```

```powershell
factorx 1024 | trx ' ' '\n' | tailx -n +2 | wcx -l
```

Factor numbers in the portable `gnux shell`:

```bash
gnux shell -c 'for n in 6 28 496; do factorx -h $n; done'
```

```powershell
gnux shell -c 'for n in 6 28 496; do factorx -h $n; done'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
factorx --help                # options, first examples, exit status
gnux help factorx             # this page: the full manual
gnux help factorx --OPTION    # one option in depth, with its examples
```

### See also

- [seqx](../seqx/index.md)
- [exprx](../exprx/index.md)
- [xargsx](../xargsx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
