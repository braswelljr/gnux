## lognamex

Print the user's login name.

Mirrors GNU `logname`. Run it as `lognamex`, or as `gnux lognamex`.

### Synopsis

```bash
lognamex [OPTION]...
```

### GNU reference

- [GNU coreutils manual: logname invocation](https://www.gnu.org/software/coreutils/manual/html_node/logname-invocation.html)
- [Source: src/logname.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/logname.c)

### Description

`lognamex` prints the name the user logged in as, which stays the same after su or sudo, where `whoamix` prints the user the process now runs as. It takes no operands. When there is no login name, as in a cron job, a container or a service, it reports 'no login name' and exits 1. GNU `logname` asks the C library's getlogin; `gnux` has no cgo and does what glibc does instead: on Linux it reads the login user the kernel records for the process (/proc/self/loginuid), and failing that, on every Unix, the login record of the terminal on standard input. macOS and the BSDs also keep the name in the kernel session, which only the C library can read, so there a process with no terminal on standard input has no login name for `lognamex` even when GNU `logname` would print one. On Windows it prints the user the session was signed in as, without the domain.

### Permissions

Needs no permissions. Reads the process's login user and the login records.

### Use cases

- Finding out `who` really logged in inside a sudo or su session.
- Addressing the human user in a script that runs as root.
- Detecting that a script runs without a login (cron, a service) by its exit status.
- Recording `who` started a deployment in its log.
- Choosing a per-user directory in an administration script.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Print your login name:

```bash
lognamex
```

```powershell
lognamex
```

Output:

```text
alice
```

See the difference from `whoamix` under sudo:

```bash
sudo sh -c 'lognamex; whoamix'
```

```powershell
lognamex; whoamix
```

Output:

```text
alice
root
```

Record `who` ran a deployment:

```bash
echox "deployed by $(lognamex)" >> deploy.log
```

```powershell
"deployed by $(lognamex)" >> deploy.log
```

Fall back to the effective user when there is no login:

```bash
name=$(lognamex 2>/dev/null || whoamix)
```

```powershell
$name = lognamex 2>$null; if ($LASTEXITCODE -ne 0) { $name = whoamix }
```

Detect a run without a login:

```bash
lognamex >/dev/null 2>&1 || echox 'no login session'
```

```powershell
lognamex *> $null; if ($LASTEXITCODE -ne 0) { 'no login session' }
```

Use the login name in a path:

```bash
mkdirx -p "/srv/scratch/$(lognamex)"
```

```powershell
mkdirx -p "scratch/$(lognamex)"
```

Save the login name in a variable:

```bash
me=$(lognamex)
```

```powershell
$me = lognamex
```

Greet the user:

```bash
printfx 'hello, %s\n' "$(lognamex)"
```

```powershell
printfx 'hello, %s\n' (lognamex)
```

List the other sessions of the same user:

```bash
whox | grepx "^$(lognamex) "
```

```powershell
whox | grepx "^$(lognamex) "
```

Print the login name in the portable `gnux shell`:

```bash
gnux shell -c 'lognamex'
```

```powershell
gnux shell -c 'lognamex'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
lognamex --help                # options, first examples, exit status
gnux help lognamex             # this page: the full manual
gnux help lognamex --OPTION    # one option in depth, with its examples
```

### See also

- [whoamix](../whoamix/index.md)
- [usersx](../usersx/index.md)
- [whox](../whox/index.md)
- [idx](../idx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
