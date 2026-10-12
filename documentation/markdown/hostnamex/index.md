## hostnamex

Print the name of the current host system.

Mirrors GNU `hostname`. Run it as `hostnamex`, or as `gnux hostnamex`.

### Synopsis

```bash
hostnamex [NAME]
```

### GNU reference

- [GNU coreutils manual: hostname invocation](https://www.gnu.org/software/coreutils/manual/html_node/hostname-invocation.html)
- [Source: src/hostname.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/hostname.c)

### Description

`hostnamex` prints the name of the current host, as the operating system reports it. GNU coreutils `hostname` also sets the name when given NAME (with sethostname(2), as root); `gnux` never changes system settings, so `hostnamex` refuses NAME with 'cannot set name to NAME: Operation not supported' and exits 1. Use the system's own tool (hostnamectl, scutil, Rename-Computer) to set it. Note that the `hostname` program shipped by many Linux distributions is not the coreutils one and has options such as `-f` and `-I`; `hostnamex` has none of them.

### Permissions

Needs no permissions to print the name.

### Use cases

- Recording which machine produced a log line or a build artifact.
- Naming per-host files or directories.
- Branching a script on the machine it runs on.
- Checking a container's or a CI runner's host name.
- Getting the host name the same way on Linux, macOS and Windows.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Print the host name:

```bash
hostnamex
```

```powershell
hostnamex
```

Tag a log line with the host:

```bash
echox "$(hostnamex): backup done" >> backup.log
```

```powershell
"$(hostnamex): backup done" >> backup.log
```

Name an archive after the machine:

```bash
cpx backup.tar "backup-$(hostnamex).tar"
```

```powershell
cpx backup.tar "backup-$(hostnamex).tar"
```

Run a step only on one machine:

```bash
[ "$(hostnamex)" = build-01 ] && make release
```

```powershell
if ((hostnamex) -eq 'build-01') { make release }
```

Print the user and the host together:

```bash
echox "$(whoamix)@$(hostnamex)"
```

```powershell
"$(whoamix)@$(hostnamex)"
```

Compare with the network node name `unamex` reports:

```bash
hostnamex; unamex -n
```

```powershell
hostnamex; unamex -n
```

Setting the name is refused (exit status 1):

```bash
hostnamex new-name || echox 'use hostnamectl instead'
```

```powershell
hostnamex new-name; if ($LASTEXITCODE -ne 0) { 'use Rename-Computer instead' }
```

Print the host name in upper case:

```bash
hostnamex | trx a-z A-Z
```

```powershell
hostnamex | trx a-z A-Z
```

Create a per-host directory:

```bash
mkdirx -p "logs/$(hostnamex)"
```

```powershell
mkdirx -p "logs/$(hostnamex)"
```

Print the host name in the portable `gnux shell`:

```bash
gnux shell -c 'hostnamex'
```

```powershell
gnux shell -c 'hostnamex'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
hostnamex --help                # options, first examples, exit status
gnux help hostnamex             # this page: the full manual
gnux help hostnamex --OPTION    # one option in depth, with its examples
```

### See also

- [unamex](../unamex/index.md)
- [whoamix](../whoamix/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
