## sleepx

Delay for a specified amount of time.

Mirrors GNU `sleep`. Run it as `sleepx`, or as `gnux sleepx`.

### Synopsis

```bash
sleepx NUMBER[SUFFIX]...
```

### GNU reference

- [GNU coreutils manual: sleep invocation](https://www.gnu.org/software/coreutils/manual/html_node/sleep-invocation.html)
- [Source: src/sleep.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/sleep.c)

### Description

`sleepx` pauses for NUMBER seconds. SUFFIX may be 's' for seconds (the default), 'm' for minutes, 'h' for hours or 'd' for days. NUMBER need not be an integer: fractions (0.5), exponents (1e3) and hexadecimal floating-point values are accepted, as is 'infinity' (or 'inf') to pause until interrupted. Given two or more arguments, `sleepx` pauses for the `sum` of their values. Every operand is checked before the pause begins, so an invalid interval fails immediately.

### Permissions

Needs no permissions; it touches nothing.

### Use cases

- Pacing a retry or polling loop in a script.
- Giving a service time to start before the next step runs.
- Writing delays that behave the same in every shell, fractional seconds included.
- Rate-limiting a loop that calls an API or writes to a slow device.
- Waiting for a file or flag to appear before continuing.
- Keeping a container or background job alive with `sleepx` infinity.
- Spacing out progress messages or demo output so it can be read.
- Simulating a slow command when testing timeouts, spinners or pipelines.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Pause for half a second:

```bash
sleepx 0.5
```

```powershell
sleepx 0.5
```

Pause for one and a half minutes:

```bash
sleepx 1m 30s
```

```powershell
sleepx 1m 30s
```

Pause for two hours:

```bash
sleepx 2h
```

```powershell
sleepx 2h
```

Use a fraction of a larger unit (6 minutes):

```bash
sleepx 0.1h
```

```powershell
sleepx 0.1h
```

Write the interval in scientific or hexadecimal notation:

```bash
sleepx 1e-1 0x0.1p0
```

```powershell
sleepx 1e-1 0x0.1p0
```

Run the next command only after the pause:

```bash
sleepx 2 && echox ready
```

```powershell
sleepx 2; if ($LASTEXITCODE -eq 0) { echox ready }
```

Print a line every second, three times:

```bash
for i in 1 2 3; do echox "tick $i"; sleepx 1; done
```

```powershell
foreach ($i in 1..3) { echox "tick $i"; sleepx 1 }
```

Retry a command every two seconds until it succeeds:

```bash
until curl -fs localhost:8080/health; do sleepx 2; done
```

```powershell
while (-not (Test-NetConnection localhost -Port 8080 -InformationLevel Quiet)) { sleepx 2 }
```

Wait until a file appears:

```bash
until [ -e ready.flag ]; do sleepx 0.5; done; echox found
```

```powershell
while (-not (Test-Path ready.flag)) { sleepx 0.5 }; echox found
```

Run a slow stand-in command in the background:

```bash
sleepx 30 & echox "started $!"
```

```powershell
Start-Job { sleepx 30 }
```

See an invalid interval rejected before any pause:

```bash
sleepx 10x; echox "status: $?"
```

```powershell
sleepx 10x; echox "status: $LASTEXITCODE"
```

Space out the steps of a `gnux shell` script:

```bash
gnux shell -c 'echox start; sleepx 1; echox done'
```

```powershell
gnux shell -c 'echox start; sleepx 1; echox done'
```

Block until interrupted:

```bash
sleepx infinity
```

```powershell
sleepx infinity
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | the full interval elapsed |
| 1 | an invalid interval or a command-line mistake |

### Help in the terminal

```bash
sleepx --help                # options, first examples, exit status
gnux help sleepx             # this page: the full manual
gnux help sleepx --OPTION    # one option in depth, with its examples
```

### See also

- [envx](../envx/index.md)
- [yesx](../yesx/index.md)
- [gnux shell](../shell/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
