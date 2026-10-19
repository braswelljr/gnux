## groupsx

Print the groups a user is in.

Mirrors GNU `groups`. Run it as `groupsx`, or as `gnux groupsx`.

### Synopsis

```bash
groupsx [OPTION]... [USERNAME]...
```

### GNU reference

- [GNU coreutils manual: groups invocation](https://www.gnu.org/software/coreutils/manual/html_node/groups-invocation.html)
- [Source: src/groups.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/groups.c)

### Description

`groupsx` prints the names of the groups the current process is in: its real group, its effective group when that differs, then its supplementary groups, each once, as `idx -Gn` does. For each USERNAME it prints 'USERNAME : ' and the groups the account database lists for that user instead, which can differ from a running process's groups until the user logs in again. An unknown user is reported as 'no such user' and the remaining names are still printed; a group with no name is reported and printed as its number. On Windows the groups are the account's local groups, which is what Go's account lookup reports.

### Permissions

Needs no permissions. Reads the process's credentials and the account database.

### Use cases

- Checking that you are in the docker, wheel or sudo group after adding yourself.
- Seeing which groups another user belongs to.
- Debugging a 'Permission denied' on a group-writable file.
- Recording the CI user's groups in a build log.
- Testing group membership in a script with `grepx -w`.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Print your groups:

```bash
groupsx
```

```powershell
groupsx
```

Output:

```text
staff admin docker
```

Print another user's groups:

```bash
groupsx root
```

```powershell
groupsx Administrator
```

Output:

```text
root : root wheel
```

Print several users' groups:

```bash
groupsx alice bob
```

```powershell
groupsx alice bob
```

Check membership of one group:

```bash
groupsx | grepx -qw docker && echox 'in docker'
```

```powershell
groupsx | grepx -qw docker; if ($LASTEXITCODE -eq 0) { 'in docker' }
```

Print one group per line:

```bash
groupsx | trx ' ' '\n'
```

```powershell
groupsx | trx ' ' '\n'
```

Count your groups:

```bash
groupsx | wcx -w
```

```powershell
groupsx | wcx -w
```

Print the numeric group IDs instead:

```bash
idx -G
```

```powershell
idx -G
```

Compare the account database with this process:

```bash
groupsx "$(whoamix)"; groupsx
```

```powershell
groupsx (whoamix); groupsx
```

Log the groups of the build user:

```bash
echox "groups: $(groupsx)" >> build.log
```

```powershell
"groups: $(groupsx)" >> build.log
```

Check a group in the portable `gnux shell`:

```bash
gnux shell -c 'if groupsx | grepx -qw wheel; then echox admin; fi'
```

```powershell
gnux shell -c 'if groupsx | grepx -qw wheel; then echox admin; fi'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
groupsx --help                # options, first examples, exit status
gnux help groupsx             # this page: the full manual
gnux help groupsx --OPTION    # one option in depth, with its examples
```

### See also

- [idx](../idx/index.md)
- [whoamix](../whoamix/index.md)
- [chgrpx](../chgrpx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
