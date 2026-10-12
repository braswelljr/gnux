## unamex

Print system information.

Mirrors GNU `uname`. Run it as `unamex`, or as `gnux unamex`.

### Synopsis

```bash
unamex [OPTION]...
```

### GNU reference

- [GNU coreutils manual: uname invocation](https://www.gnu.org/software/coreutils/manual/html_node/uname-invocation.html)
- [Source: src/uname.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/uname.c)

### Description

`unamex` prints information about the machine and operating system. With no option it prints the kernel name, as `-s` does. The selected fields are printed on one line, separated by spaces, in a fixed order whatever the order of the options: kernel name (`-s`), network node host name (`-n`), kernel release (`-r`), kernel version (`-v`), machine hardware name (`-m`), processor type (`-p`), hardware platform (`-i`) and operating system (`-o`). `-a` prints them all, leaving out `-p` and `-i` when they are unknown, as GNU does. The values come from `uname`(2) on Linux, sysctl on macOS and FreeBSD, and the registry on Windows, so `unamex` prints the same fields on every platform: on Windows the kernel name is 'Windows_NT' and the operating system 'MS/Windows'; in the browser playground the fields describe the WebAssembly runtime.

### Permissions

Needs no permissions.

### Use cases

- Branching a portable script on the operating system or architecture.
- Recording the platform in bug reports and build logs.
- Picking the right download for the machine's architecture with `-m`.
- Checking the kernel release before relying on a kernel feature.
- Getting the same answer on Linux, macOS, FreeBSD and Windows.

### Flags

```text
  -a, --all                 print all information, in the following order, except omit -p and -i if unknown
  -i, --hardware-platform   print the hardware platform (non-portable)
      --help                display this help and exit
  -s, --kernel-name         print the kernel name
  -r, --kernel-release      print the kernel release
  -v, --kernel-version      print the kernel version
  -m, --machine             print the machine hardware name
  -n, --nodename            print the network node hostname
  -o, --operating-system    print the operating system
  -p, --processor           print the processor type (non-portable)
      --version             output version information and exit
```

### Examples

Print the kernel name:

```bash
unamex
```

```powershell
unamex
```

Print everything:

```bash
unamex -a
```

```powershell
unamex -a
```

Print the machine architecture:

```bash
unamex -m
```

```powershell
unamex -m
```

Print the kernel release:

```bash
unamex -r
```

```powershell
unamex -r
```

Print the operating system:

```bash
unamex -o
```

```powershell
unamex -o
```

Print the kernel name and release:

```bash
unamex -sr
```

```powershell
unamex -sr
```

Print the host name:

```bash
unamex -n
```

```powershell
unamex -n
```

Branch a script on the operating system:

```bash
case "$(unamex -s)" in Linux) echox linux ;; Darwin) echox macos ;; esac
```

```powershell
switch (unamex -s) { 'Linux' { 'linux' } 'Darwin' { 'macos' } 'Windows_NT' { 'windows' } }
```

Pick a download for this machine:

```bash
curl -LO "https://example.com/tool-$(unamex -s)-$(unamex -m).tar.gz"
```

```powershell
Invoke-WebRequest "https://example.com/tool-$(unamex -s)-$(unamex -m).zip" -OutFile tool.zip
```

Record the platform in a log:

```bash
unamex -a >> build.log
```

```powershell
unamex -a >> build.log
```

Print system information in the portable `gnux shell`:

```bash
gnux shell -c 'unamex -srm'
```

```powershell
gnux shell -c 'unamex -srm'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
unamex --help                # options, first examples, exit status
gnux help unamex             # this page: the full manual
gnux help unamex --OPTION    # one option in depth, with its examples
```

### See also

- [hostnamex](../hostnamex/index.md)
- [nprocx](../nprocx/index.md)
- [uptimex](../uptimex/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
