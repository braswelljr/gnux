## nohupx

Run a command immune to hangups, with output to a non-terminal.

Mirrors GNU `nohup`. Run it as `nohupx`, or as `gnux nohupx`.

### Synopsis

```bash
nohupx COMMAND [ARG]...
```

### GNU reference

- [GNU coreutils manual: nohup invocation](https://www.gnu.org/software/coreutils/manual/html_node/nohup-invocation.html)
- [Source: src/nohup.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/nohup.c)

### Description

`nohupx` runs COMMAND so that it keeps running after you log out or close the terminal. On Unix it ignores the hangup signal, as GNU `nohup` does. Any of the standard streams that is a terminal is replaced: standard input by an unusable /dev/null, standard output by the file `nohup`.out, appended to and created readable only by you, in the current directory or, if that fails, in $HOME, and standard error by standard output; `nohupx` says which on standard error. Streams that are already redirected are left alone. The exit status is COMMAND's, or 125 when `nohupx` itself failed, 126 when COMMAND was found but could not be run and 127 when it was not found. GNU `nohup` does nothing useful on Windows; `nohupx` starts COMMAND detached from the console and in a process group of its own, so closing the console window or pressing Ctrl-C in it no longer ends COMMAND. `nohupx` runs COMMAND as a child and waits for it, where GNU `nohup` becomes COMMAND; to get your prompt back, start it in the background with your shell.

### Permissions

Needs execute permission on COMMAND, and write permission on the current directory or $HOME when `nohup`.out has to be created.

### Use cases

- Keeping a long download, build or migration running after an SSH session drops.
- Starting a server from a terminal that will be closed.
- Capturing the output of a background job in a file automatically.
- Running a job on Windows that must survive the console window closing.
- Combining with `timeoutx` and `nicex` for a bounded, low-priority background job.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Keep a long rsync running after your SSH session ends:

```bash
nohupx rsync -a /data backup:/data &
```

```powershell
Start-Job { nohupx robocopy C:\data \\backup\data /E }
```

Start a server in the background and log its output:

```bash
nohupx ./server > server.log 2>&1 &
```

```powershell
Start-Process nohupx -ArgumentList '.\server.exe' -RedirectStandardOutput server.log -NoNewWindow
```

Let output go to `nohup`.out and follow it:

```bash
nohupx ./train.py & tailx -f nohup.out
```

```powershell
Start-Job { nohupx python train.py }; tailx -f nohup.out
```

Remember the process ID to stop it later:

```bash
nohupx ./worker & echox $! > worker.pid
```

```powershell
(Start-Process nohupx -ArgumentList '.\worker.exe' -PassThru).Id > worker.pid
```

Stop that job when it is no longer needed:

```bash
killx "$(catx worker.pid)"
```

```powershell
killx (catx worker.pid)
```

Run a migration that must not be cut short by a closed laptop:

```bash
nohupx ./migrate.sh > migrate.log 2>&1 &
```

```powershell
Start-Job { nohupx .\migrate.ps1 > migrate.log }
```

Run a nightly build at low priority, immune to hangups:

```bash
nohupx nicex make release &
```

```powershell
Start-Job { nohupx nicex make release }
```

Give a background job at most two hours:

```bash
nohupx timeoutx 2h ./crawl.sh &
```

```powershell
Start-Job { nohupx timeoutx 2h .\crawl.ps1 }
```

Keep standard error separate from the output:

```bash
nohupx ./job 2> job.err &
```

```powershell
Start-Job { nohupx .\job.exe 2> job.err }
```

Check how the job ended:

```bash
nohupx ./job; echox "exit status $?"
```

```powershell
nohupx .\job.exe; "exit status $LASTEXITCODE"
```

Run an external program immune to hangups from the portable `gnux shell`:

```bash
gnux shell -c 'nohupx envx ./server > server.log'
```

```powershell
gnux shell -c 'nohupx envx ./server > server.log'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 125 | `nohupx` itself failed |
| 126 | COMMAND was found but could not be run |
| 127 | COMMAND was not found |
| other | the exit status of COMMAND |

### Help in the terminal

```bash
nohupx --help                # options, first examples, exit status
gnux help nohupx             # this page: the full manual
gnux help nohupx --OPTION    # one option in depth, with its examples
```

### See also

- [timeoutx](../timeoutx/index.md)
- [nicex](../nicex/index.md)
- [killx](../killx/index.md)
- [teex](../teex/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
