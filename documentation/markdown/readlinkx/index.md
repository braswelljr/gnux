## readlinkx

Print resolved symbolic links or canonical file names.

Mirrors GNU `readlink`. Run it as `readlinkx`, or as `gnux readlinkx`.

### Synopsis

```bash
readlinkx [OPTION]... FILE...
```

### GNU reference

- [GNU coreutils manual: readlink invocation](https://www.gnu.org/software/coreutils/manual/html_node/readlink-invocation.html)
- [Source: src/readlink.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/readlink.c)

### Description

`readlinkx` prints the value of each FILE, which must be a symbolic `link`: the target exactly as it was stored, not resolved. With `-f`, `-e` or `-m` it prints the canonical name instead: absolute, with every symbolic `link` and every '.' and '..' component resolved. `-f` requires every component but the last to exist, `-e` requires all of them, and `-m` requires none. As in GNU `readlink`, failures are silent by default (`-q` and `-s` say so explicitly) and only the exit status reports them; `-v` prints a diagnostic for each. `-n` drops the trailing newline, and is ignored with a warning when there is more than one FILE; `-z` ends each name with NUL. GNU `readlink` lets the last of `-f`, `-e` and `-m` win; `readlinkx` cannot see option order, so it picks the most demanding (`-e`, then `-f`, then `-m`), and `-v` wins over `-q` and `-s`.

### Permissions

Needs search permission on every directory leading to FILE, and on every directory a `link` leads through. Reading a `link`'s value needs no permission on the `link` itself. Nothing is ever modified.

### Use cases

- Finding where a symbolic `link` points, exactly as it was created.
- Getting the absolute, `link``-free` path of a script or binary with `-f`.
- Checking in a script that a path fully exists after resolution with `-e`.
- Normalizing a path that does not exist yet with `-m`.
- Detecting whether a file is a symbolic `link` from the exit status.
- Comparing two paths for identity after resolving their links.
- Producing NUL-separated output for names with newlines with `-z`.

### Flags

```text
  -f, --canonicalize            canonicalize by following every symlink in every component recursively; all but the last component must exist
  -e, --canonicalize-existing   canonicalize by following every symlink in every component recursively; all components must exist
  -m, --canonicalize-missing    canonicalize by following every symlink in every component recursively, without requirements on existence
      --help                    display this help and exit
  -n, --no-newline              do not output the trailing delimiter
  -q, --quiet                   suppress most error messages (on by default)
  -s, --silent                  suppress most error messages (on by default)
  -v, --verbose                 report error messages
      --version                 output version information and exit
  -z, --zero                    end each output line with NUL, not newline
```

### Examples

Print where a `link` points:

```bash
readlinkx current
```

```powershell
readlinkx current
```

Print the canonical, absolute name of a file:

```bash
readlinkx -f ./bin/../bin/tool
```

```powershell
readlinkx -f ./bin/../bin/tool
```

Require the whole path to exist:

```bash
readlinkx -e config/app.yaml
```

```powershell
readlinkx -e config/app.yaml
```

Normalize a path that may not exist yet:

```bash
readlinkx -m build/../dist/out
```

```powershell
readlinkx -m build/../dist/out
```

Explain why a name cannot be resolved:

```bash
readlinkx -v -e missing/file
```

```powershell
readlinkx -v -e missing/file
```

Test whether a file is a symbolic `link`:

```bash
readlinkx current > /dev/null && echox 'a link'
```

```powershell
readlinkx current > $null; if ($LASTEXITCODE -eq 0) { 'a link' }
```

Capture a resolved path without a trailing newline:

```bash
printfx 'cd %s\n' "$(readlinkx -n -f link)"
```

```powershell
$p = readlinkx -n -f link; "cd $p"
```

Resolve several links at once, NUL-separated:

```bash
readlinkx -z -f a b c | trx '\0' '\n'
```

```powershell
readlinkx -z -f a b c | trx '\0' '\n'
```

Find the directory a script really lives in:

```bash
dirnamex "$(readlinkx -f "$0")"
```

```powershell
dirnamex (readlinkx -f $PSCommandPath)
```

Check that two names reach the same file:

```bash
[ "$(readlinkx -f a)" = "$(readlinkx -f b)" ] && echox same
```

```powershell
if ((readlinkx -f a) -eq (readlinkx -f b)) { 'same' }
```

Resolve a `link` in the portable `gnux shell`:

```bash
gnux shell -c 'readlinkx -f current'
```

```powershell
gnux shell -c 'readlinkx -f current'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
readlinkx --help                # options, first examples, exit status
gnux help readlinkx             # this page: the full manual
gnux help readlinkx --OPTION    # one option in depth, with its examples
```

### See also

- [realpathx](../realpathx/index.md)
- [lnx](../lnx/index.md)
- [lsx](../lsx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
