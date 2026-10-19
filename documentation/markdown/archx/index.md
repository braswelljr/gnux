## archx

Print machine architecture.

Mirrors GNU `arch`. Run it as `archx`, or as `gnux archx`.

### Synopsis

```bash
archx [OPTION]...
```

### GNU reference

- [GNU coreutils manual: arch invocation](https://www.gnu.org/software/coreutils/manual/html_node/arch-invocation.html)
- [Source: src/arch.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/arch.c)

### Description

`archx` prints the machine hardware name, exactly as `unamex -m` does, and takes no operands. The name comes from the operating system: x86_64 or aarch64 on Linux, x86_64 or arm64 on macOS, amd64 or arm64 on FreeBSD. On Windows, and in the browser playground, `gnux` names the architecture it was built for the way Linux does (x86_64, aarch64, wasm32).

### Permissions

Needs no permissions. Reads only system information.

### Use cases

- Picking the right binary to download in an `install` script.
- Recording the machine's architecture in a CI log or a bug report.
- Skipping a step that only works on one architecture.
- Naming build output after the architecture it was built on.
- Checking whether a shell runs under emulation (Rosetta, QEMU).

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Print the machine architecture:

```bash
archx
```

```powershell
archx
```

Output:

```text
x86_64
```

Print the same name through `unamex`:

```bash
unamex -m
```

```powershell
unamex -m
```

Pick a download for this machine:

```bash
curl -LO "https://example.com/tool-$(archx).tar.gz"
```

```powershell
curl.exe -LO "https://example.com/tool-$(archx).zip"
```

Run a step only on 64-bit ARM:

```bash
case "$(archx)" in aarch64|arm64) echox 'ARM build' ;; esac
```

```powershell
if ((archx) -in 'aarch64','arm64') { 'ARM build' }
```

Name an output directory after the architecture:

```bash
mkdirx -p "build/$(archx)"
```

```powershell
mkdirx -p "build/$(archx)"
```

Record the system in a log:

```bash
echox "$(unamex -s) $(archx)" >> build.log
```

```powershell
"$(unamex -s) $(archx)" >> build.log
```

Save the architecture in a variable:

```bash
arch=$(archx)
```

```powershell
$arch = archx
```

Print the architecture and the number of processors:

```bash
echox "$(archx), $(nprocx) cores"
```

```powershell
"$(archx), $(nprocx) cores"
```

Compare with the expected architecture:

```bash
[ "$(archx)" = x86_64 ] || echox 'not x86_64'
```

```powershell
if ((archx) -ne 'x86_64') { 'not x86_64' }
```

Branch on the architecture in the portable `gnux shell`:

```bash
gnux shell -c 'case $(archx) in x86_64) echox intel ;; *) echox other ;; esac'
```

```powershell
gnux shell -c 'case $(archx) in x86_64) echox intel ;; *) echox other ;; esac'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
archx --help                # options, first examples, exit status
gnux help archx             # this page: the full manual
gnux help archx --OPTION    # one option in depth, with its examples
```

### See also

- [unamex](../unamex/index.md)
- [nprocx](../nprocx/index.md)
- [hostnamex](../hostnamex/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
