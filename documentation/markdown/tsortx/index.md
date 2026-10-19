## tsortx

Write a totally ordered list consistent with the partial ordering in FILE.

Mirrors GNU `tsort`. Run it as `tsortx`, or as `gnux tsortx`.

### Synopsis

```bash
tsortx [OPTION] [FILE]
```

### GNU reference

- [GNU coreutils manual: tsort invocation](https://www.gnu.org/software/coreutils/manual/html_node/tsort-invocation.html)
- [Source: src/tsort.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/tsort.c)

### Description

`tsortx` reads pairs of strings, separated by spaces, tabs and newlines, from FILE (standard input when there is none, or when FILE is -). Each pair 'A B' says that A comes before B; a pair of the same string twice just names that string. It prints every string once, each after every string it was paired before. It follows GNU's algorithm, so the order is GNU's: the strings that nothing comes before are taken in byte order, then strings in the order the ones before them are printed. When the pairs contain a loop, `tsortx` reports 'input contains a loop:' and the strings of the loop on standard error, drops one pair of it and carries on, so every string is still printed, and exits 1. An odd number of strings is an error and prints nothing.

### Permissions

Needs read permission on FILE. Writes nothing.

### Use cases

- Ordering build steps, packages or modules so each comes after what it depends on.
- Finding a dependency loop: `tsortx` names the strings in it.
- Ordering database migrations or initialization scripts by their prerequisites.
- Scheduling tasks from a list of 'before after' pairs.
- Ordering object files for a linker from their symbol dependencies.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Order tasks from 'before after' pairs:

```bash
printfx 'fetch build\nbuild test\nbuild package\n' | tsortx
```

```powershell
printfx 'fetch build\nbuild test\nbuild package\n' | tsortx
```

Output:

```text
fetch
build
package
test
```

Order the pairs listed in a file:

```bash
tsortx deps.txt
```

```powershell
tsortx deps.txt
```

Put several pairs on one line:

```bash
echox 'a b b c c d' | tsortx
```

```powershell
echox 'a b b c c d' | tsortx
```

Output:

```text
a
b
c
d
```

Name a string that has no relations by pairing it with itself:

```bash
printfx 'a b\nlonely lonely\n' | tsortx
```

```powershell
printfx 'a b\nlonely lonely\n' | tsortx
```

Find a dependency loop:

```bash
printfx 'a b\nb c\nc a\n' | tsortx
```

```powershell
printfx 'a b\nb c\nc a\n' | tsortx
```

Output:

```text
tsortx: -: input contains a loop:
tsortx: a
tsortx: b
tsortx: c
a
b
c
```

Fail a build script when the dependencies contain a loop:

```bash
tsortx deps.txt > order.txt || echox 'dependency loop' >&2
```

```powershell
tsortx deps.txt > order.txt; if ($LASTEXITCODE -ne 0) { 'dependency loop' }
```

Print the order reversed, dependents first:

```bash
tsortx deps.txt | tacx
```

```powershell
tsortx deps.txt | tacx
```

Number the steps of the order:

```bash
tsortx deps.txt | nlx
```

```powershell
tsortx deps.txt | nlx
```

Read the pairs from standard input given as -:

```bash
catx deps.txt | tsortx -
```

```powershell
catx deps.txt | tsortx -
```

Run steps in dependency order in the portable `gnux shell`:

```bash
gnux shell -c 'for step in $(tsortx deps.txt); do echox "running $step"; done'
```

```powershell
gnux shell -c 'for step in $(tsortx deps.txt); do echox "running $step"; done'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | every string was printed in an order consistent with every pair |
| 1 | the input contains a loop (every string is still printed), an odd number of strings, or FILE cannot be read |

### Help in the terminal

```bash
tsortx --help                # options, first examples, exit status
gnux help tsortx             # this page: the full manual
gnux help tsortx --OPTION    # one option in depth, with its examples
```

### See also

- [sortx](../sortx/index.md)
- [uniqx](../uniqx/index.md)
- [joinx](../joinx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
