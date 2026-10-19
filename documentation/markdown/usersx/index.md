## usersx

Print the user names of users currently logged in to the current host.

Mirrors GNU `users`. Run it as `usersx`, or as `gnux usersx`.

### Synopsis

```bash
usersx [OPTION]... [FILE]
```

### GNU reference

- [GNU coreutils manual: users invocation](https://www.gnu.org/software/coreutils/manual/html_node/users-invocation.html)
- [Source: src/users.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/users.c)

### Description

`usersx` prints the login name of each session on this machine, sorted and separated by spaces on one line; a user logged in more than once appears once per session. With nobody logged in it prints nothing. It reads the same records as `whox`: the utmp file on Linux and the BSDs (sessions whose process has ended are left out) and utmpx on macOS. FILE reads another file in that format instead, such as /var/log/wtmp, without leaving any session out. On Windows the sessions are the signed-in console and Remote Desktop sessions, and FILE is not supported; in the browser playground there are no records.

### Permissions

Needs read permission on the login records (normally readable by everyone) or on FILE. Writes nothing.

### Use cases

- Seeing `who` else is logged in before rebooting a shared machine.
- Counting the sessions on a server.
- Checking whether a particular user is logged in.
- Recording `who` was logged in when a job ran.
- Listing the users recorded in an old wtmp file.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

List the users logged in:

```bash
usersx
```

```powershell
usersx
```

Output:

```text
alice alice bob
```

Count the sessions:

```bash
usersx | wcx -w
```

```powershell
usersx | wcx -w
```

List each user once:

```bash
usersx | trx ' ' '\n' | uniqx
```

```powershell
usersx | trx ' ' '\n' | uniqx
```

Check whether a user is logged in:

```bash
usersx | grepx -qw alice && echox 'alice is on'
```

```powershell
usersx | grepx -qw alice; if ($LASTEXITCODE -eq 0) { 'alice is on' }
```

Count the sessions of each user:

```bash
usersx | trx ' ' '\n' | uniqx -c
```

```powershell
usersx | trx ' ' '\n' | uniqx -c
```

Read the users from a wtmp file:

```bash
usersx /var/log/wtmp
```

```powershell
usersx /var/log/wtmp
```

See the same users with their terminals and times:

```bash
whox
```

```powershell
whox
```

Warn before a reboot when anyone else is on:

```bash
[ "$(usersx | trx ' ' '\n' | grepx -vx "$(whoamix)")" ] && echox 'others are logged in'
```

```powershell
if (usersx | trx ' ' '\n' | grepx -vx (whoamix)) { 'others are logged in' }
```

Log `who` was on when a job ran:

```bash
echox "$(datex): $(usersx)" >> jobs.log
```

```powershell
"$(datex): $(usersx)" >> jobs.log
```

List the users in the portable `gnux shell`:

```bash
gnux shell -c 'usersx'
```

```powershell
gnux shell -c 'usersx'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
usersx --help                # options, first examples, exit status
gnux help usersx             # this page: the full manual
gnux help usersx --OPTION    # one option in depth, with its examples
```

### See also

- [whox](../whox/index.md)
- [whoamix](../whoamix/index.md)
- [lognamex](../lognamex/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
