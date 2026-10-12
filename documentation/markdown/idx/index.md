## idx

Print real and effective user and group IDs.

Mirrors GNU `id`. Run it as `idx`, or as `gnux idx`.

### Synopsis

```bash
idx [OPTION]... [USER]...
```

### GNU reference

- [GNU coreutils manual: id invocation](https://www.gnu.org/software/coreutils/manual/html_node/id-invocation.html)
- [Source: src/id.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/id.c)

### Description

`idx` prints the user and group IDs of the current process, or of each USER (a name or a numeric ID), in GNU's default format: 'uid=1000(alice) gid=1000(alice) groups=1000(alice),27(sudo)', with euid= and egid= added when they differ from the real IDs. `-u`, `-g` and `-G` print only the effective user ID, the effective group ID or every group ID; `-n` prints names instead of numbers and `-r` the real IDs instead of the effective ones, with one of those three; `-z` ends each entry with NUL. `-a` is accepted and ignored, as in GNU. `-Z` (SELinux context) is not supported and fails, as GNU `id` does on a kernel without SELinux. An unknown USER is reported ('no such user') and the remaining USERs are still printed. On Windows the IDs are security identifiers (SIDs) and the names come from the account database.

### Permissions

Needs no permissions; names are read from the system's user and group databases.

### Use cases

- Checking which user and groups a script or service runs as.
- Getting the numeric UID and GID to pass to `chownx` or a container.
- Checking whether a user belongs to a group, such as docker or sudo.
- Refusing to run a script as root by testing `idx -u`.
- Listing a user's groups by name.

### Flags

```text
  -a, --all       ignore, for compatibility with other versions
  -Z, --context   print only the security context of the process (not supported)
  -g, --group     print only the effective group ID
  -G, --groups    print all group IDs
      --help      display this help and exit
  -n, --name      print a name instead of a number, for -u,-g,-G
  -r, --real      print the real ID instead of the effective ID, with -u,-g,-G
  -u, --user      print only the effective user ID
      --version   output version information and exit
  -z, --zero      delimit entries with NUL characters, not whitespace; not permitted in default format
```

### Examples

Print your user and group IDs:

```bash
idx
```

```powershell
idx
```

Print another user's IDs:

```bash
idx alice
```

```powershell
idx alice
```

Print only the effective user ID:

```bash
idx -u
```

```powershell
idx -u
```

Print your user name:

```bash
idx -un
```

```powershell
idx -un
```

Print the effective group ID:

```bash
idx -g
```

```powershell
idx -g
```

List your groups by name:

```bash
idx -Gn
```

```powershell
idx -Gn
```

Print the real user ID rather than the effective one:

```bash
idx -ru
```

```powershell
idx -ru
```

Refuse to run as root:

```bash
[ "$(idx -u)" -eq 0 ] && { echox 'do not run as root' >&2; exit 1; }
```

```powershell
if ((idx -u) -eq '0') { Write-Error 'do not run as root'; exit 1 }
```

Check membership of a group:

```bash
idx -Gn | trx ' ' '\n' | grep -qx docker && echox member
```

```powershell
if ((idx -Gn) -split ' ' -contains 'docker') { 'member' }
```

Pass your IDs to a container:

```bash
docker run --user "$(idx -u):$(idx -g)" image
```

```powershell
docker run --user "$(idx -u):$(idx -g)" image
```

Print IDs in the portable `gnux shell`:

```bash
gnux shell -c 'idx -un && idx -Gn'
```

```powershell
gnux shell -c 'idx -un && idx -Gn'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
idx --help                # options, first examples, exit status
gnux help idx             # this page: the full manual
gnux help idx --OPTION    # one option in depth, with its examples
```

### See also

- [whoamix](../whoamix/index.md)
- [chownx](../chownx/index.md)
- [chgrpx](../chgrpx/index.md)
- [whox](../whox/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
