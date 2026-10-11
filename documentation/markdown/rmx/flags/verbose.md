## verbose

Explain what is being removed (`-v`/`--verbose`).

### Description

`-v`/`--verbose` prints a line for each item removed, e.g. removed 'notes.txt'. To keep the report in the order you listed the arguments, verbose removal runs sequentially rather than concurrently.

### Permissions

`-v` does not change permission handling or what may be removed; it only reports the items that were successfully removed. Files it could not remove are reported as errors on stderr as usual.

### Use cases

- Auditing exactly what a recursive or wildcard delete removed.
- Confirming a script deleted the files you expected.
- Troubleshooting by pairing `-v` with `-i` to see and approve each removal.
- Keeping a log of a cleanup by saving the report to a file.

### Flags

```text
  -v, --verbose   explain what is being done
```

### Examples

Remove several files, printing a line per file:

```bash
rmx -v a.txt b.txt
```

```powershell
rmx -v a.txt b.txt
```

Recursively remove a directory, reporting each removal:

```bash
rmx -rv dist/
```

```powershell
rmx -rv dist
```

Save the removal report to a log file:

```bash
rmx -rv cache/ > cleanup.log
```

```powershell
rmx -rv cache | Out-File cleanup.log
```

Count the entries removed:

```bash
rmx -rv tmp/ | wcx -l
```

```powershell
rmx -rv tmp | wcx -l
```

Report removals of files that may not exist, with the `gnux shell`:

```bash
gnux shell -c 'rmx -fv *.bak'
```

```powershell
gnux shell -c 'rmx -fv *.bak'
```
