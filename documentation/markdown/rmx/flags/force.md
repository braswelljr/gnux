## force

Ignore nonexistent files and arguments, and never prompt.

### Description

The `-f`/`--force` flag makes `rmx` ignore files that do not exist (exiting successfully instead of reporting an error) and suppresses every confirmation prompt, including the write-protected-file prompt. It is what turns `rmx` into an unattended, script-safe delete. `-f` only affects prompting and missing-file errors; it does not enable directory removal - combine it with `-r` for that.

### Permissions

`-f` skips the write-protected-file prompt, so read-only files are removed without asking. Deletion is still governed by the parent directory's permissions, not the file's own mode: if you lack write+execute on the containing directory the removal fails with 'Permission denied' even under `-f`.

### Use cases

- Unattended cleanup in CI pipelines and shell scripts where prompts would hang.
- Deleting paths that may or may not exist without treating absence as an error.
- Removing read-only or write-protected files without per-file confirmation.
- Making cleanup steps idempotent so a script can be re-run safely.

### Flags

```text
  -f, --force   ignore nonexistent files and arguments, never prompt
```

### Examples

Remove a file, succeeding silently if it is already gone:

```bash
rmx -f maybe-missing.log
```

```powershell
rmx -f maybe-missing.log
```

Recursively delete a build directory with no prompts:

```bash
rmx -rf build/
```

```powershell
rmx -rf build
```

Remove a write-protected file without being asked:

```bash
rmx -f readonly.cfg
```

```powershell
rmx -f readonly.cfg
```

Run with no operands at all, which `-f` also tolerates:

```bash
rmx -f && echox "nothing to do"
```

```powershell
rmx -f; if ($LASTEXITCODE -eq 0) { echox "nothing to do" }
```

Make a cleanup step idempotent in the `gnux shell`:

```bash
gnux shell -c 'rmx -fv out.log err.log; echox $?'
```

```powershell
gnux shell -c 'rmx -fv out.log err.log; echox $?'
```
