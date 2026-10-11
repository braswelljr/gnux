## touchx

Update the access and modification times of each FILE to the current time.

Mirrors GNU `touch`. Run it as `touchx`, or as `gnux touchx`.

### Synopsis

```bash
touchx [OPTION]... FILE...
```

### Description

`touchx` sets the access and modification times of each FILE to the current time, creating an empty file for any FILE that does not exist unless `-c` is given. An existing file's contents are never changed. `-a` and `-m` (or `--time`) restrict the update to one of the two timestamps and leave the other exactly as it was. The time to apply can come from somewhere other than the clock: `-d` takes a date string (2024-01-02, '2024-01-02 15:04:05', an RFC 3339 timestamp, @SECONDS since the epoch, or now), `-t` takes a [[CC]YY]MMDDhhmm[.ss] stamp, and `-r` copies the times of another file. `-h` acts on a symbolic link itself rather than the file it points to.

### Permissions

Setting a file's times to the current time needs write permission on it; setting them to any other time (`-d`, `-t`, `-r`) needs ownership of the file. Creating a missing file needs write and search permission on its directory. Changing the times of a symbolic link itself (`-h`) is only possible on Linux; elsewhere `touchx` reports the link rather than silently touching its target.

### Use cases

- Creating an empty placeholder or marker file.
- Forcing a rebuild by making a source file newer than its outputs.
- Stamping files with a fixed time for reproducible archives.
- Copying timestamps from one file to another with `-r`.
- Refreshing the timestamps of existing files only, without creating strays, with `-c`.
- Recording when a step ran by touching a marker file that later steps compare against.
- Backdating a file with `-d` or `-t` to test retention or cleanup rules.
- Changing only the access time (`-a`) or only the modification time (`-m`).

### Flags

```text
  -a, --access-time         change only the access time
  -d, --date STRING         parse STRING and use it instead of current time
      --help                display this help and exit
  -m, --modification-time   change only the modification time
  -c, --no-create           do not create any files
  -h, --no-dereference      affect each symbolic link instead of any referenced file
  -r, --reference FILE      use this FILE's times instead of current time
  -t, --stamp STAMP         use [[CC]YY]MMDDhhmm[.ss] STAMP instead of current time
      --time WORD           change the specified time: WORD is access, atime, or use (like -a), or modify or mtime (like -m)
      --version             output version information and exit
```

### Examples

Create a file, or bump its timestamps if it exists:

```bash
touchx notes.txt
```

```powershell
touchx notes.txt
```

Create several empty files at once:

```bash
touchx a.txt b.txt c.txt
```

```powershell
touchx a.txt b.txt c.txt
```

Update timestamps without creating anything:

```bash
touchx -c build/*.o
```

```powershell
touchx -c (Get-ChildItem build\*.o)
```

Set only the modification time to a specific date:

```bash
touchx -m -d '2024-01-02 15:04:05' report.pdf
```

```powershell
touchx -m -d '2024-01-02 15:04:05' report.pdf
```

Set only the access time to now:

```bash
touchx -a app.log
```

```powershell
touchx -a app.log
```

Choose the timestamp by name with `--time` (access/atime/use or modify/mtime):

```bash
touchx --time=mtime app.log
```

```powershell
touchx --time=mtime app.log
```

Use a [[CC]YY]MMDDhhmm[.ss] stamp:

```bash
touchx -t 202401021504.05 report.pdf
```

```powershell
touchx -t 202401021504.05 report.pdf
```

Backdate only the access time with a stamp:

```bash
touchx -a -t 202001010000 app.log
```

```powershell
touchx -a -t 202001010000 app.log
```

Use an RFC 3339 timestamp, or seconds since the epoch:

```bash
touchx -d 2024-06-01T12:00:00Z release.txt && touchx -d @0 epoch.txt
```

```powershell
touchx -d 2024-06-01T12:00:00Z release.txt; touchx -d '@0' epoch.txt
```

Give one file the timestamps of another:

```bash
touchx -r original.jpg edited.jpg
```

```powershell
touchx -r original.jpg edited.jpg
```

Copy just the modification time, and only to files that already exist:

```bash
touchx -c -m -r original.jpg edited.jpg report.pdf
```

```powershell
touchx -c -m -r original.jpg edited.jpg report.pdf
```

Stamp all sources with one fixed date for a reproducible archive:

```bash
touchx -d 2024-01-01 src/*.go
```

```powershell
touchx -d 2024-01-01 (Get-ChildItem src\*.go)
```

Change the times of a symbolic link itself, not its target (Linux only):

```bash
touchx -h -d 2024-01-01 link.txt
```

```powershell
touchx -h -d 2024-01-01 link.txt
```

Create a marker file inside a new directory:

```bash
mkdirx -p cache && touchx cache/.keep
```

```powershell
mkdirx -p cache; touchx cache\.keep
```

Stamp files and list them with the `gnux shell`:

```bash
gnux shell -c 'touchx -d 2024-01-01 a.txt b.txt && lsx -l a.txt b.txt'
```

```powershell
gnux shell -c 'touchx -d 2024-01-01 a.txt b.txt && lsx -l a.txt b.txt'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
touchx --help                # options, first examples, exit status
gnux help touchx             # this page: the full manual
gnux help touchx --OPTION    # one option in depth, with its examples
```

### See also

- [lsx](../lsx/index.md)
- [mkdirx](../mkdirx/index.md)
- [catx](../catx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
