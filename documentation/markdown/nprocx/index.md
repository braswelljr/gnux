## nprocx

Print the number of processing units available.

Mirrors GNU `nproc`. Run it as `nprocx`, or as `gnux nprocx`.

### Synopsis

```bash
nprocx [OPTION]...
```

### GNU reference

- [GNU coreutils manual: nproc invocation](https://www.gnu.org/software/coreutils/manual/html_node/nproc-invocation.html)
- [Source: src/nproc.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/nproc.c)

### Description

`nprocx` prints the number of processing units available to the current process. As in GNU `nproc`, the OMP_NUM_THREADS environment variable (its first value, when it is a comma-separated list) replaces that count when it is a positive number, and OMP_THREAD_LIMIT caps the result. `--all` prints the number of installed processors instead and ignores both variables. `--ignore`=N subtracts N, but the result is never less than 1. `gnux` has no cgo, so both counts come from Go's runtime.NumCPU: the processors this process may run on. A machine with processors taken offline, or a process restricted to fewer of them, prints the same number with and without `--all`.

### Permissions

Needs no permissions. Reads only the process's CPU affinity and its environment.

### Use cases

- Choosing how many parallel jobs a build should run.
- Leaving a core free for the rest of the system with `--ignore`=1.
- Respecting OMP_NUM_THREADS when a job scheduler has set it.
- Reporting the machine's size in a CI log or a bug report.
- Sizing a worker pool in a portable script.

### Flags

```text
      --all        print the number of installed processors
      --help       display this help and exit
      --ignore N   if possible, exclude N processing units (default "0")
      --version    output version information and exit
```

### Examples

Print the number of processing units:

```bash
nprocx
```

```powershell
nprocx
```

Print the number of installed processors:

```bash
nprocx --all
```

```powershell
nprocx --all
```

Leave one core free:

```bash
nprocx --ignore=1
```

```powershell
nprocx --ignore=1
```

Run a build with one job per core:

```bash
make -j"$(nprocx)"
```

```powershell
make -j(nprocx)
```

Respect an OpenMP thread count:

```bash
OMP_NUM_THREADS=4 nprocx
```

```powershell
$env:OMP_NUM_THREADS = 4; nprocx
```

Cap the count with OMP_THREAD_LIMIT:

```bash
OMP_THREAD_LIMIT=2 nprocx
```

```powershell
$env:OMP_THREAD_LIMIT = 2; nprocx
```

Set the count through `envx`, on any platform:

```bash
envx OMP_NUM_THREADS=3 nprocx
```

```powershell
envx OMP_NUM_THREADS=3 nprocx
```

Save the count in a variable:

```bash
jobs=$(nprocx --ignore=2); echox "using $jobs jobs"
```

```powershell
$jobs = nprocx --ignore=2; "using $jobs jobs"
```

Print the count with a label:

```bash
printfx 'cores: %d\n' "$(nprocx)"
```

```powershell
printfx 'cores: %d\n' (nprocx)
```

Show the count in the portable `gnux shell`:

```bash
gnux shell -c 'nprocx --all'
```

```powershell
gnux shell -c 'nprocx --all'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
nprocx --help                # options, first examples, exit status
gnux help nprocx             # this page: the full manual
gnux help nprocx --OPTION    # one option in depth, with its examples
```

### See also

- [unamex](../unamex/index.md)
- [envx](../envx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
