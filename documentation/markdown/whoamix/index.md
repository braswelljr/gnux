## whoamix

Print the user name associated with the current effective user ID.

Mirrors GNU `whoami`. Run it as `whoamix`, or as `gnux whoamix`.

### Synopsis

```bash
whoamix [OPTION]...
```

### GNU reference

- [GNU coreutils manual: whoami invocation](https://www.gnu.org/software/coreutils/manual/html_node/whoami-invocation.html)
- [Source: src/whoami.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/whoami.c)

### Description

`whoamix` prints the name of the user the process runs as: the effective user, so inside sudo it prints root. It takes no operands. On Windows, where the account name is DOMAIN\user, only the user part is printed, as GNU `whoami` does under Cygwin and MSYS. If the user ID has no name, `whoamix` reports 'cannot find name for user ID N' and exits 1.

### Permissions

Needs read access to the user database (/etc/passwd or the system's directory service) to turn the user ID into a name.

### Use cases

- Refusing to run a script as root, or insisting on it.
- Naming per-user files, directories or caches in a portable way.
- Recording `who` ran a job in a log line.
- Checking which account a CI runner or container uses.
- Getting the user name on Windows without USERNAME and on Unix without USER.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Print your user name:

```bash
whoamix
```

```powershell
whoamix
```

Refuse to run as root:

```bash
[ "$(whoamix)" = root ] && { echox 'do not run as root' >&2; exit 1; }
```

```powershell
if ((whoamix) -eq 'Administrator') { throw 'do not run as Administrator' }
```

Create a per-user scratch directory:

```bash
mkdirx -p "/tmp/$(whoamix)/cache"
```

```powershell
mkdirx -p "$env:TEMP/$(whoamix)/cache"
```

Record `who` ran a job:

```bash
echox "deployed by $(whoamix) on $(hostnamex)" >> deploy.log
```

```powershell
"deployed by $(whoamix) on $(hostnamex)" >> deploy.log
```

Compare with the user sudo runs as:

```bash
whoamix; sudo whoamix
```

```powershell
whoamix
```

Print the user with a label:

```bash
printfx 'user: %s\n' "$(whoamix)"
```

```powershell
printfx 'user: %s\n' (whoamix)
```

Check which user a container runs as:

```bash
docker run --rm gnux whoamix
```

```powershell
docker run --rm gnux whoamix
```

Use the name in a file name:

```bash
cpx notes.md "notes-$(whoamix).md"
```

```powershell
cpx notes.md "notes-$(whoamix).md"
```

Fail clearly when the user has no name (exit status 1):

```bash
whoamix || echox 'no user name for this ID'
```

```powershell
whoamix; if ($LASTEXITCODE -ne 0) { 'no user name for this ID' }
```

Print the user in the portable `gnux shell`:

```bash
gnux shell -c 'whoamix'
```

```powershell
gnux shell -c 'whoamix'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
whoamix --help                # options, first examples, exit status
gnux help whoamix             # this page: the full manual
gnux help whoamix --OPTION    # one option in depth, with its examples
```

### See also

- [hostnamex](../hostnamex/index.md)
- [printenvx](../printenvx/index.md)
- [envx](../envx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
