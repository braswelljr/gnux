## whox

Show `who` is logged on.

Mirrors GNU `who`. Run it as `whox`, or as `gnux whox`.

### Synopsis

```bash
whox [OPTION]... [ FILE | ARG1 ARG2 ]
```

### GNU reference

- [GNU coreutils manual: who invocation](https://www.gnu.org/software/coreutils/manual/html_node/who-invocation.html)
- [Source: src/who.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/who.c)

### Description

`whox` lists the users logged in: name, terminal, login time and, for a remote session, the host in parentheses. `-u` adds how long each terminal has been idle ('.' under a minute, 'old' over a day) and the session's process ID, `-T` (or `-w`) whether the user accepts messages (+, - or ?), and `-H` a line of headings. `-q` prints just the names and their count. `-b` shows the last boot, `-r` the run level, `-l` login processes, `-p` processes started by init, `-d` dead processes and `-t` clock changes; `-a` shows all of them. `-m` (or any two operands, as in '`whox` am i') shows only the terminal on standard input. `--lookup` turns host addresses into names. With FILE, the records are read from that utmp file (on Linux, /var/log/wtmp lists past logins too) instead of the current ones, and in the current ones, logins whose process has ended are left out. Times are in the C locale's 'Oct 12 09:30' format. The records are read without the C library, in each system's own format: Linux utmp, macOS utmpx and FreeBSD utx.active. On Windows, which keeps sessions rather than a utmp file, `whox` lists the signed-in console and Remote Desktop sessions, so the same command answers '`who` is on this machine' everywhere; there FILE is not supported.

### Permissions

Needs read permission on the login records, which every user has.

### Use cases

- Checking `who` else is logged in to a shared server before rebooting it.
- Seeing which remote hosts users are connected from.
- Finding idle sessions to clean up with `-u`.
- Checking when the machine last booted with `-b`.
- Counting logged-in users in a monitoring script with `-q`.

### Flags

```text
  -a, --all        same as -b -d --login -p -r -t -T -u
  -b, --boot       time of last system boot
  -q, --count      all login names and number of users logged on
  -d, --dead       print dead processes
  -H, --heading    print line of column headings
      --help       display this help and exit
  -l, --login      print system login processes
      --lookup     attempt to canonicalize hostnames via DNS
  -T, --mesg       add user's message status as +, - or ?
      --message    same as -T
  -m, --my-line    only hostname and user associated with stdin
  -p, --process    print active processes spawned by init
  -r, --runlevel   print current runlevel
  -s, --short      print only name, line, and time (default)
  -t, --time       print last system clock change
  -u, --users      list users logged in
      --version    output version information and exit
  -w, --writable   same as -T
```

### Examples

Before rebooting a shared server, see `who` is still logged in:

```bash
whox
```

```powershell
whox
```

Output:

```text
alice    pts/0        Oct 12 09:30 (10.0.0.5)
bob      pts/1        Oct 12 13:02 (laptop.lan)
```

See how long each session has been idle, with headings:

```bash
whox -uH
```

```powershell
whox -uH
```

Output:

```text
NAME     LINE         TIME         IDLE          PID COMMENT
alice    pts/0        Oct 12 09:30 01:30        1234 (10.0.0.5)
bob      pts/1        Oct 12 13:02   .          1300 (laptop.lan)
```

Check when the machine last booted:

```bash
whox -b
```

```powershell
whox -b
```

Output:

```text
         system boot  Oct  9 10:44
```

Count the logged-in users for a monitoring check:

```bash
whox -q
```

```powershell
whox -q
```

Output:

```text
alice bob
# users=2
```

Find out which terminal and login you are on:

```bash
whox am i
```

```powershell
whox am i
```

Output:

```text
alice    pts/0        Oct 12 09:30 (10.0.0.5)
```

See `who` will receive a 'wall' broadcast (+ accepts messages):

```bash
whox -T
```

```powershell
whox -T
```

Show every record, including dead processes and the run level:

```bash
whox -a
```

```powershell
whox -a
```

Show the remote hosts by name instead of address:

```bash
whox --lookup
```

```powershell
whox --lookup
```

Review past logins from the Linux login history:

```bash
whox /var/log/wtmp | tailx
```

```powershell
whox
```

Warn in a script when anyone besides you is logged in:

```bash
[ "$(whox -q | tailx -n 1 | cutx -d= -f2)" -gt 1 ] && echox 'others are logged in'
```

```powershell
if ([int]((whox -q | tailx -n 1) -split '=')[1] -gt 1) { 'others are logged in' }
```

List logged-in users in the portable `gnux shell`, the same on every OS:

```bash
gnux shell -c 'whox -q && uptimex'
```

```powershell
gnux shell -c 'whox -q && uptimex'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
whox --help                # options, first examples, exit status
gnux help whox             # this page: the full manual
gnux help whox --OPTION    # one option in depth, with its examples
```

### See also

- [whoamix](../whoamix/index.md)
- [idx](../idx/index.md)
- [uptimex](../uptimex/index.md)
- [ttyx](../ttyx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
