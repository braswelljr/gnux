## recursive

Remove directories and their contents recursively (`-r`, `-R`).

### Description

By default `rmx` refuses to remove directories. `-r` (or its alias `-R`) removes each directory and everything beneath it, walking the tree post-order so children are removed before their parents. It composes with the other flags: `-ri` prompts per entry, `-rv` reports each removal, `-rf` deletes without prompting.

### Permissions

Recursive removal needs read+execute permission to list and descend into each directory, and write permission on a directory to unlink the entries inside it. A directory you cannot write to fails with 'Permission denied' and its contents are left in place. The `--preserve-root` failsafe additionally refuses to recurse on '/'.

### Use cases

- Deleting a directory and all of its contents in one command.
- Clearing build/output trees (build/, dist/, node_modules/).
- Combining with `-i`/`-I` for confirmation or `-v` to audit a large delete.
- Resetting a generated directory before regenerating it.

### Flags

```text
  -r, --recursive   remove directories and their contents recursively
```

### Examples

Remove a directory and everything under it:

```bash
rmx -r logs/
```

```powershell
rmx -r logs
```

Recursively remove a directory, printing each deletion:

```bash
rmx -rv cache/
```

```powershell
rmx -rv cache
```

Use the `-R` spelling, as with GNU `rm`:

```bash
rmx -R dist/
```

```powershell
rmx -R dist
```

Remove several trees without prompting:

```bash
rmx -rf node_modules/ coverage/
```

```powershell
rmx -rf node_modules coverage
```

Reset a directory and recreate it empty:

```bash
rmx -rf out/ && mkdirx out
```

```powershell
rmx -rf out; if ($LASTEXITCODE -eq 0) { mkdirx out }
```

Recursively clean up with the `gnux shell`:

```bash
gnux shell -c 'rmx -r tmp-*'
```

```powershell
gnux shell -c 'rmx -r tmp-*'
```
