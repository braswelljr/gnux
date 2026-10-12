## unlinkx

Call the `unlink` function to remove the specified file.

Mirrors GNU `unlink`. Run it as `unlinkx`, or as `gnux unlinkx`.

### Synopsis

```bash
unlinkx FILE
```

### GNU reference

- [GNU coreutils manual: unlink invocation](https://www.gnu.org/software/coreutils/manual/html_node/unlink-invocation.html)
- [Source: src/unlink.c in coreutils/coreutils](https://github.com/coreutils/coreutils/blob/master/src/unlink.c)

### Description

`unlinkx` removes the single name FILE with the `unlink`(2) system call. It takes exactly one operand and no options: it never recurses, never prompts, never ignores a missing file, and refuses a directory, even an empty one, with 'Is a directory' (use `rmdirx` for that). A symbolic `link` is removed, not its target. Removing the last name of a file frees its data once no process has it open.

### Permissions

Needs write and search permission on the directory containing FILE; permission on FILE itself does not matter. In a directory with the sticky bit set (such as /tmp), only the file's owner, the directory's owner or root may remove it. On Windows, a read-only file or one open in another process cannot be removed.

### Use cases

- Removing exactly one file in a script, failing loudly if anything is unexpected.
- Removing a symbolic `link` without any risk of touching what it points to.
- Releasing a lock file created with `linkx`.
- Removing a file whose name could be mistaken for an option, with --.
- Removing one of several hard links while keeping the others.

### Flags

```text
      --help      display this help and exit
      --version   output version information and exit
```

### Examples

Remove one file:

```bash
unlinkx old.log
```

```powershell
unlinkx old.log
```

Remove a symbolic `link`, leaving its target alone:

```bash
unlinkx current
```

```powershell
unlinkx current
```

A directory is refused (use `rmdirx`):

```bash
unlinkx build || rmdirx build
```

```powershell
unlinkx build; if ($LASTEXITCODE -ne 0) { rmdirx build }
```

Release a lock file:

```bash
unlinkx app.lock
```

```powershell
unlinkx app.lock
```

Remove a file whose name starts with a dash:

```bash
unlinkx -- -oops
```

```powershell
unlinkx -- -oops
```

Drop one hard `link` and keep the other name:

```bash
linkx data data.bak && unlinkx data
```

```powershell
linkx data data.bak; unlinkx data
```

Fail when the file is already gone (`rmx -f` would not):

```bash
unlinkx tmp.txt || echox 'nothing to remove'
```

```powershell
unlinkx tmp.txt; if ($LASTEXITCODE -ne 0) { 'nothing to remove' }
```

Remove a file with spaces in its name:

```bash
unlinkx 'my file.txt'
```

```powershell
unlinkx 'my file.txt'
```

Remove a temporary file at the end of a script:

```bash
tmp=$(mktempx) && echox data > "$tmp" && unlinkx "$tmp"
```

```powershell
$tmp = mktempx; 'data' > $tmp; unlinkx $tmp
```

Remove a file in the portable `gnux shell`:

```bash
gnux shell -c 'unlinkx out.txt'
```

```powershell
gnux shell -c 'unlinkx out.txt'
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | success: every operand was processed |
| 1 | failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake |

### Help in the terminal

```bash
unlinkx --help                # options, first examples, exit status
gnux help unlinkx             # this page: the full manual
gnux help unlinkx --OPTION    # one option in depth, with its examples
```

### See also

- [rmx](../rmx/index.md)
- [rmdirx](../rmdirx/index.md)
- [linkx](../linkx/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
