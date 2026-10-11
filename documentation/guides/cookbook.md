# Cookbook

A task-oriented recipe book for the gnux command-line tools: over 90 short recipes grouped by what you are trying to do, from inspecting and listing files through copying, linking, safe deletion and backups to reading logs, manipulating paths, controlling the environment, setting timestamps, generating data and building pipelines. Each recipe has a one-line reason, a bash form, and a PowerShell form where it differs, and the outputs shown were captured from a real gnux build.

## How to read the recipes

- Commands use the per-tool names (`rmx`, `cpx`, ...). If you did not link them, prefix each with `gnux`: `gnux rmx -rf build`.
- **PowerShell does not expand wildcards** for native programs, and gnux does not expand them itself. Where a recipe uses `*.log`, the PowerShell form either lists the files with `Get-ChildItem` or runs the line through `gnux shell -c`, which globs the same way on every platform.
- **Redirection** with `>` in Windows PowerShell 5.1 writes UTF-16. Where a recipe writes a file, the PowerShell form uses `gnux shell -c` so the bytes are exactly what the tool printed. PowerShell 7 writes UTF-8 and either form works there.
- When there is no PowerShell block, the bash command runs unchanged in PowerShell.

The tool reference pages hold every flag: start at the [reference index](../markdown/index.md).

## Contents

1. [Inspecting files](#inspecting-files)
2. [Listing directories](#listing-directories)
3. [Creating files and directories](#creating-files-and-directories)
4. [Copying](#copying)
5. [Moving and renaming](#moving-and-renaming)
6. [Linking](#linking)
7. [Safe deletion](#safe-deletion)
8. [Backups](#backups)
9. [Logs](#logs)
10. [Paths](#paths)
11. [Environment](#environment)
12. [Timing and timestamps](#timing-and-timestamps)
13. [Generating data](#generating-data)
14. [Pipelines](#pipelines)

## Inspecting files

### 1. Print a file with line numbers

Why: quote a line by number in a review or bug report.

```bash
catx -n notes.txt
```

```text
     1	line one
     2	line two
     3	
     4	
     5		indented	tab  
```

### 2. Number only the non-empty lines

Why: blank lines should not shift the numbering of real content.

```bash
catx -b notes.txt
```

```text
     1	line one
     2	line two


     3		indented	tab  
```

### 3. Reveal tabs, trailing spaces and line endings

Why: find the stray tab, trailing whitespace or CRLF that breaks a YAML or Makefile.

```bash
catx -A notes.txt
```

```text
line one$
line two$
$
$
^Iindented^Itab  $
```

A Windows line ending shows up as `^M$`. Use `-E` for the `$` markers alone, `-T` for tabs alone.

### 4. Squeeze runs of blank lines

Why: tidy a generated file before reading it.

```bash
catx -s notes.txt
```

### 5. Peek at the start of a large file

Why: `headx` stops reading as soon as it has enough, so it is instant on a multi-gigabyte file.

```bash
headx data.csv            # first 10 lines
headx -n 3 data.csv       # first 3
headx -3 data.csv         # the historical spelling of -n 3
headx -c 64 image.png     # first 64 bytes
```

### 6. Drop a trailer

Why: everything except the last lines, such as a summary footer.

```bash
headx -n -2 report.txt    # all but the last 2 lines
headx -c -1 file.txt      # all but the final byte
```

### 7. Skip a header line

Why: process a CSV without its column names.

```bash
tailx -n +2 data.csv
```

### 8. Count lines, words and bytes

Why: the size of a file in every unit at once.

```bash
wcx notes.txt
wcx -l notes.txt
wcx -w -c notes.txt
```

```text
 5  6 36 notes.txt
5 notes.txt
 6 36 notes.txt
```

The counts always come in the order lines, words, characters, bytes, longest line, whatever the order of the flags.

### 9. Count characters, not bytes

Why: UTF-8 text has more bytes than characters.

```bash
echox 'héllo wörld' | wcx -m -c
```

```text
     12      14
```

### 10. Find the widest line

Why: check a file against a line-length limit before formatting it.

```bash
wcx -L notes.txt
```

```text
29 notes.txt
```

Tabs count to the next multiple of 8, as on a terminal.

### 11. Total several files, or only the total

Why: the size of a codebase in one number.

```bash
wcx -l a.txt b.txt app.log
wcx -l --total=only a.txt b.txt app.log
```

```text
 1 a.txt
 1 b.txt
 5 app.log
 7 total
7
```

```powershell
wcx -l (Get-ChildItem *.txt, *.log).Name
gnux shell -c 'wcx -l --total=only *.txt *.log'
```

### 12. Join files into one

Why: concatenate parts, or prepend a header to a stream.

```bash
catx part1.txt part2.txt part3.txt > whole.txt
echox 'header' | catx - body.txt > with-header.txt   # - is standard input
```

```powershell
gnux shell -c 'catx part1.txt part2.txt part3.txt > whole.txt'
gnux shell -c "echox header | catx - body.txt > with-header.txt"
```

## Listing directories

### 13. Long listing with hidden files

Why: permissions, owners, sizes and times, dotfiles included.

```bash
lsx -la
```

```text
total 4
drwx------ 4 alice staff 128 Oct 11 18:36 .
drwxr-xr-x 6 alice staff 192 Oct 11 18:36 ..
-rw-r--r-- 1 alice staff  36 Oct 11 18:36 notes.txt
drwxr-xr-x 4 alice staff 128 Oct 11 18:36 project
```

Use `-A` instead of `-a` to leave out `.` and `..`.

### 14. Largest files first, in human units

Why: find what is eating the disk.

```bash
lsx -lhS
```

```text
total 8.0K
-rw-r--r-- 1 alice staff 4.9K Oct 11 18:55 big.bin
drwxr-xr-x 2 alice staff   64 Oct 11 18:55 d
...
```

`--si` uses powers of 1000 instead of 1024.

### 15. Newest files first

Why: what changed most recently.

```bash
lsx -lt | headx -n 5
lsx -ltr                  # oldest first: the newest end up next to the prompt
```

### 16. Natural version order

Why: `file2` before `file10`.

```bash
lsx -v
```

```text
a.txt
b.o
big.bin
d
file1.txt
file2.txt
file10.txt
```

### 17. Directories first, then files

Why: a tidier view of a project root.

```bash
lsx --group-directories-first
```

### 18. Hide build artifacts from a listing

Why: skip object files and editor backups.

```bash
lsx -I '*.o'              # never list *.o
lsx -B                    # never list names ending in ~
lsx --hide='*.txt'        # hide *.txt unless -a or -A is given
```

### 19. Mark file types

Why: tell directories, links and executables apart without color.

```bash
lsx -F                    # / for directories, @ for links, * for executables
lsx -p                    # only / after directories
```

### 20. One name per line, for scripts

Why: feed names to another command.

```bash
lsx -1
lsx -1 | wcx -l           # how many entries
```

When output is piped or redirected, `lsx` prints one name per line anyway.

### 21. Walk a tree

Why: see a whole directory hierarchy.

```bash
lsx -R project
```

```text
project:
docs
src

project/docs:

project/src:
app

project/src/app:
```

### 22. Exact timestamps

Why: a sortable, unambiguous date in a listing or a manifest.

```bash
lsx -l --time-style=long-iso
lsx -l --full-time
lsx -l --time-style=+%Y-%m-%d
```

```text
-rw-r--r-- 1 alice staff 0 2024-01-02 15:04 old.txt
-rw-r--r-- 1 alice staff 0 2024-01-02 15:04:05.000000000 +0000 old.txt
```

### 23. Show a directory itself, not its contents

Why: check a directory's own permissions.

```bash
lsx -ld secrets
```

```text
drwx------ 2 alice staff 64 Oct 11 18:36 secrets
```

### 24. Hard-link counts and inode numbers

Why: prove two names are the same file.

```bash
lsx -li orig.txt hard.txt
```

```text
67066675 -rw-r--r-- 2 alice staff 5 Oct 11 18:36 hard.txt
67066675 -rw-r--r-- 2 alice staff 5 Oct 11 18:36 orig.txt
```

## Creating files and directories

### 25. Create a nested tree in one step

Why: `-p` creates missing parents and is happy if the directory already exists, so it is safe to repeat.

```bash
mkdirx -p app/src app/tests app/docs
mkdirx -pv app/src app/tests app/docs     # report what was created
```

```text
mkdirx: created directory 'app'
mkdirx: created directory 'app/src'
mkdirx: created directory 'app/tests'
mkdirx: created directory 'app/docs'
```

### 26. A private directory, whatever the umask

Why: secrets should not be world-readable.

```bash
mkdirx -m 700 secrets
mkdirx -m u=rwx,go= secrets2     # symbolic form
```

On Windows only the read-only attribute is representable, so the mode has little effect there.

### 27. Create empty placeholder files

Why: marker files, `.keep` files and stubs.

```bash
touchx app/docs/.keep app/src/main.go
```

### 28. Touch only files that already exist

Why: refresh timestamps without creating anything by accident.

```bash
touchx -c maybe-there.txt
```

### 29. Write a small file

Why: a version stamp, a config line or a `.gitignore`.

```bash
echox '1.4.0' > VERSION
echox -e 'bin/\ndist/\n*.log' > .gitignore
```

```powershell
gnux shell -c "echox 1.4.0 > VERSION"
gnux shell -c "echox -e 'bin/\ndist/\n*.log' > .gitignore"
```

`echox -e` interprets `\n`, `\t` and the other escapes identically on every platform.

## Copying

### 30. Copy a file, showing what happened

```bash
cpx -v app/README.md app/README.md.bak
```

```text
'app/README.md' -> 'app/README.md.bak'
```

### 31. Copy a directory tree

```bash
cpx -r app app-copy
```

Without `-r` a directory is skipped with `cpx: -r not specified; omitting directory 'app'`.

### 32. Copy several files into a directory

Why: `-t` names the destination first, which is convenient when the list of sources is long.

```bash
cpx -t app/docs app/README.md app/src/main.go -v
```

```text
'app/README.md' -> 'app/docs/README.md'
'app/src/main.go' -> 'app/docs/main.go'
```

### 33. Never overwrite

Why: a script that must not clobber an existing file.

```bash
cpx -n defaults.conf app.conf
```

### 34. Copy only what is newer

Why: refresh a mirror without rewriting unchanged files.

```bash
cpx -ruv src/ mirror/
```

As with GNU `cp`, when `mirror/` already exists the tree lands in `mirror/src`; when it does not, `mirror` becomes the copy.

### 35. Ask before overwriting

```bash
cpx -i template.md README.md
```

```text
cpx: overwrite 'README.md'?
```

Answer `y` to proceed; anything else, or end of input, keeps the file.

### 36. Recreate the source path under a directory

Why: copy a few deep files while keeping their layout.

```bash
cpx --parents app/src/main.go dist
lsx -R dist
```

```text
dist:
app

dist/app:
src

dist/app/src:
main.go
```

### 37. A faithful archive copy

Why: modes, timestamps, symbolic links and hard links preserved.

```bash
cpx -a project project.backup
```

`-a` is `-dR --preserve=all`. On Windows only the read-only attribute and the modification time carry over.

### 38. A tree of links instead of copies

Why: a cheap snapshot that shares data with the original.

```bash
cpx -al project project.snapshot     # hard links, same filesystem only
cpx -s "$PWD/a.txt" a-link.txt       # a symbolic link
```

## Moving and renaming

### 39. Rename

```bash
mvx -v draft.md final.md
```

```text
renamed 'draft.md' -> 'final.md'
```

### 40. Move many files into a directory

```bash
mvx -t archive/ *.log -v
```

```powershell
mvx -t archive/ -v (Get-ChildItem *.log).Name
gnux shell -c 'mvx -t archive/ -v *.log'
```

### 41. Move without clobbering

```bash
mvx -n incoming/report.pdf reports/
```

An existing `reports/report.pdf` is left alone and the exit status is still 0.

### 42. Move only if newer

```bash
mvx -u build/app.js public/app.js
```

### 43. Move across disks safely

Why: `mvx` copies across filesystems with modes, times and links intact, and removes the original only after the copy fully succeeded.

```bash
mvx -v /mnt/scratch/dataset /data/
```

## Linking

### 44. Point a stable name at a release, atomically

Why: switch `current` to a new version with no window where it is missing.

```bash
mkdirx -p releases/v1.0.0 releases/v1.1.0
lnx -sfn releases/v1.1.0 current
lsx -l current
```

```text
lrwxr-xr-x 1 alice staff 15 Oct 11 18:36 current -> releases/v1.1.0
```

Roll back the same way: `lnx -sfnv releases/v1.0.0 current`. The new link is created beside the old one and renamed into place, so a failure leaves the old link intact.

### 45. A relative link that survives moving the tree

```bash
lnx -sr app/src/main.go app/docs/main-link.go
lsx -l app/docs/main-link.go
```

```text
lrwxr-xr-x 1 alice staff 14 Oct 11 18:36 app/docs/main-link.go -> ../src/main.go
```

### 46. A second name for a file

Why: a hard link shares the data, so nothing is duplicated.

```bash
lnx orig.txt hard.txt
```

Running it again fails with `lnx: failed to create hard link 'hard.txt': File exists`; add `-f` to replace.

### 47. Link several targets into a directory

```bash
lnx -s -t bin/ ../tools/fmt ../tools/lint
```

On Windows, symbolic links need Developer Mode or an elevated prompt.

## Safe deletion

### 48. Remove files, and see the error for a missing one

```bash
rmx missing.txt; echo "status $?"
```

```text
rmx: cannot remove 'missing.txt': No such file or directory
status 1
```

### 49. Remove without complaint

Why: cleanup that must succeed whether or not the files exist.

```bash
rmx -f missing.txt; echo "status $?"
```

```text
status 0
```

### 50. Remove a tree and audit every entry

```bash
rmx -rv build
```

```text
removed 'build/app'
removed 'build/obj/x/1.o'
removed directory 'build/obj/x'
removed directory 'build/obj'
removed directory 'build'
```

### 51. One confirmation for a big delete

Why: `-I` asks once before removing more than three files or recursing, instead of once per file.

```bash
rmx -I *.tmp
rmx -rI build
```

```text
rmx: remove 4 arguments?
rmx: remove 1 argument recursively?
```

```powershell
rmx -I (Get-ChildItem *.tmp).Name
```

### 52. Confirm every single entry

```bash
rmx -ri dir2
```

```text
rmx: descend into directory 'dir2'?
rmx: remove regular empty file 'dir2/g'?
rmx: descend into directory 'dir2/s'?
rmx: remove regular empty file 'dir2/s/f'?
rmx: remove directory 'dir2/s'?
rmx: remove directory 'dir2'?
```

Removal is post-order: children first, then the directory.

### 53. Remove only empty directories

Why: `rmdirx` can never destroy file contents.

```bash
rmdirx old-dir                         # fails if anything is inside
rmx -d old-dir                         # the same with rmx
rmdirx -pv empty/a                     # and each empty parent
rmdirx --ignore-fail-on-non-empty tmp  # best effort, quiet if not empty
```

```text
rmdirx: removing directory, 'empty/a'
rmdirx: removing directory, 'empty'
```

### 54. Remove a file whose name starts with a dash

```bash
rmx -- -weird
rmx ./-weird
```

### 55. Rely on the failsafes

Why: some mistakes are refused outright.

```bash
rmx -r .
rmx -rf /
```

```text
rmx: refusing to remove '.' or '..' directory: skipping '.'
rmx: it is dangerous to operate recursively on '/'
rmx: use --no-preserve-root to override this failsafe
```

## Backups

### 56. A quick `.bak` before editing

```bash
cpx -v config.yaml config.yaml.bak
```

### 57. A dated backup directory

```bash
mkdirx -p backups/2026-10-11
cpx -a project backups/2026-10-11/
```

### 58. Back up only what changed since last time

```bash
mkdirx -p backups/latest
cpx -ruv project backups/latest/
```

```text
'project' -> 'backups/latest/project'
'project/a' -> 'backups/latest/project/a'
'project/sub' -> 'backups/latest/project/sub'
'project/sub/b' -> 'backups/latest/project/sub/b'
```

A second run copies only the files that changed since.

### 59. Restore without clobbering newer work

```bash
cpx -rn backups/2026-10-11/project/. project/
```

`-n` leaves every file that already exists alone.

## Logs

### 60. The last lines of a log

```bash
tailx app.log             # last 10
tailx -n 50 app.log
tailx -c 1K app.log       # last kilobyte
```

### 61. Follow a log live

```bash
tailx -f app.log
```

Stop it with Ctrl+C. `-s 0.2` polls five times a second instead of once.

### 62. Follow across log rotation

Why: `-F` follows the file name, reopens it after rotation and waits for it to reappear.

```bash
tailx -F /var/log/app/current.log
```

When the file is missing at first, `tailx` reports it and then announces `tailx: 'current.log' has appeared;  following new file`.

### 63. Follow only while a process runs

Why: watch a build log and stop when the build ends.

```bash
make build > build.log 2>&1 &
tailx -f --pid $! build.log
```

### 64. Headers for several logs at once

```bash
headx -n 1 a.log b.log
tailx -n 3 -f a.log b.log
```

```text
==> a.log <==
one

==> b.log <==
two
```

`-q` removes the headers, `-v` adds one even for a single file.

### 65. Count the lines of a day's logs

Why: how big is today's log, in lines.

```bash
wcx -l app.log
catx app-*.log | wcx -l
```

```powershell
gnux shell -c 'catx app-*.log | wcx -l'
```

## Paths

### 66. The file name from a path

```bash
basenamex /usr/local/bin/gnux
basenamex src/main.go .go
basenamex -s .go a/x.go b/y.go
```

```text
gnux
main
x
y
```

### 67. The directory from a path

```bash
dirnamex /usr/local/bin/gnux a/b/ file
```

```text
/usr/local/bin
a
.
```

### 68. The absolute, canonical path

```bash
realpathx p/q/../q/f.txt
realpathx -m does/not/exist/yet     # nothing needs to exist
realpathx -e must/exist             # everything must exist
```

### 69. A path relative to a project root

```bash
realpathx --relative-to=p p/q/f.txt
realpathx --relative-base="$PWD" p/q/f.txt /usr
```

```text
q/f.txt
p/q/f.txt
/usr
```

```powershell
realpathx --relative-base="$PWD" p/q/f.txt C:\Windows
```

### 70. Where am I, really

```bash
pwdx          # every symbolic link resolved
pwdx -L       # as the shell walked in, symlinks kept
```

## Environment

### 71. Print the environment

```bash
envx
envx -0 | wcx -c     # NUL-separated, for safe parsing
```

### 72. Run a command with a clean environment

```bash
envx -i PATH=/usr/bin:/bin HOME="$HOME" make test
```

```powershell
envx -i "PATH=C:\Windows\System32" "SystemRoot=C:\Windows" cmd /c set
```

### 73. Set a variable for one command, in any shell

Why: `NAME=value cmd` is bash syntax; `envx` works in PowerShell and `cmd.exe` too.

```bash
envx GREETING=hi sh -c 'echo $GREETING'
```

```powershell
envx GOOS=linux GOARCH=arm64 go build -o bin/app-linux-arm64 .
```

### 74. Remove one variable

```bash
envx -u GOFLAGS go build ./...
```

### 75. Run in another directory

```bash
envx -C web pnpm install
```

### 76. See what envx does

```bash
envx -v -i A=1
```

```text
envx: cleaning environ
envx: setenv:   A=1
A=1
```

### 77. Know why a command did not run

```bash
envx nosuchcommand; echo "status $?"
```

```text
envx: 'nosuchcommand': No such file or directory
status 127
```

126 means the command was found but could not be run; 125 means `envx` itself failed.

## Timing and timestamps

### 78. Pause, with fractions and units

```bash
sleepx 0.5
sleepx 2m
sleepx 1m 30s       # arguments add up: 90 seconds
sleepx infinity     # until interrupted
```

### 79. A simple retry loop in bash

```bash
until curl -fsS http://localhost:8080/health; do sleepx 2; done
```

```powershell
while (-not (Test-NetConnection localhost -Port 8080).TcpTestSucceeded) { sleepx 2 }
```

### 80. Set a file's time to a fixed date

Why: reproducible archives and builds.

```bash
touchx -d 2024-01-02 old.txt
touchx -d '2024-01-02 15:04:05' old.txt
touchx -d 2024-01-02T15:04:05Z old.txt     # RFC 3339
touchx -d @0 epoch.txt                     # seconds since the epoch
touchx -t 202312312359.30 stamp.txt        # [[CC]YY]MMDDhhmm[.ss]
lsx -l --time-style=long-iso
```

```text
-rw-r--r-- 1 alice staff 0 2024-01-02 15:04 old.txt
-rw-r--r-- 1 alice staff 0 2023-12-31 23:59 stamp.txt
-rw-r--r-- 1 alice staff 0 1970-01-01 00:00 epoch.txt
```

`touchx -d` takes those forms and `now`, not GNU's free-form phrases: `touchx -d yesterday` is a usage error.

### 81. Copy timestamps from another file

```bash
touchx -r reference.txt target.txt
```

### 82. Force a rebuild

Why: make a source newer than its outputs.

```bash
touchx src/main.c
```

### 83. Change only one timestamp

```bash
touchx -m -d 2024-06-01 file.txt     # modification time only
touchx -a file.txt                   # access time only
```

## Generating data

### 84. A fixed number of lines

```bash
yesx | headx -n 3
yesx 'log entry' | headx -n 1000 > app.log
yesx a b c | headx -n 2
```

```text
y
y
y
a b c
a b c
```

```powershell
gnux shell -c "yesx 'log entry' | headx -n 1000 > app.log"
```

`yesx` stops quietly when `headx` has had enough.

### 85. A test file of an exact size

```bash
yesx 0123456789 | headx -c 1M > one-meg.bin
wcx -c one-meg.bin
```

```text
1048576 one-meg.bin
```

`headx` sizes accept `K`, `M`, `G` (powers of 1024), `KB`, `MB` (powers of 1000) and `b` (512).

### 86. Answer every prompt automatically

```bash
yesx | rmx -ri old-cache
```

### 87. Text with escapes, identical everywhere

```bash
echox -e 'name\tvalue\nport\t8080'
echox -n 'no newline'
```

## Pipelines

### 88. Count the entries in a directory

```bash
lsx -1 | wcx -l
```

```powershell
lsx -1 | wcx -l
```

### 89. The newest file's name

```bash
lsx -t | headx -n 1
```

### 90. The five largest files

```bash
lsx -lS | headx -n 6     # the "total" line plus five entries
```

### 91. Number the lines of any output

```bash
lsx -1 | catx -n
tailx -n 3 app.log | catx -n
```

### 92. A whole pipeline the same way on every OS

Why: `gnux shell -c` gives Windows the same globbing, quoting and redirection as Linux.

```bash
gnux shell -c 'catx *.log | wcx -l > line-count.txt'
gnux shell -c 'lsx -1 | wcx -l > count; catx count'
```

```powershell
gnux shell -c 'catx *.log | wcx -l > line-count.txt'
```

In the second line the redirection is opened before `lsx` runs, so `count` is counted too, exactly as in a POSIX shell.

## See also

- [Portable scripting](portable-scripting.md): turn these recipes into scripts.
- [Windows and PowerShell](windows-and-powershell.md): more on quoting and globbing in PowerShell.
- [Troubleshooting](troubleshooting.md): what each error in these recipes means.
- The tool references: [catx](../markdown/catx/index.md), [cpx](../markdown/cpx/index.md), [headx](../markdown/headx/index.md), [lnx](../markdown/lnx/index.md), [lsx](../markdown/lsx/index.md), [mvx](../markdown/mvx/index.md), [rmx](../markdown/rmx/index.md), [tailx](../markdown/tailx/index.md), [touchx](../markdown/touchx/index.md), [wcx](../markdown/wcx/index.md).
