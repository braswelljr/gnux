# SRS Workflows

The user-facing workflows of gnux as step-by-step flows: installing and linking, running a tool
both ways, writing and running a portable script, embedding an engine, running a tool by name
from Go, trying it in the playground, adding a tool, regenerating the documentation, and
releasing. Each flow names its actors, preconditions, steps, outcomes and failure paths, and the
requirement identifiers it exercises.

**This document stands on its own** as the answer to "how is this actually used". The
requirements it cites are in [functional-requirements](functional-requirements.md) and
[non-functional-requirements](non-functional-requirements.md). How the same processes work
inside the code is in [workflows](../workflows/README.md).

| ID   | Workflow                                  | Primary actor         |
|------|-------------------------------------------|-----------------------|
| WF-1 | Install gnux and link the tools           | Command-line user     |
| WF-2 | Run a tool, both ways                     | Command-line user     |
| WF-3 | Write and run a portable script           | Script or CI author   |
| WF-4 | Embed an engine                           | Go developer          |
| WF-5 | Run a tool or a script by name from Go    | Go developer          |
| WF-6 | Try gnux in the playground                | Evaluator             |
| WF-7 | Add a new tool, end to end                | Contributor           |
| WF-8 | Regenerate the documentation              | Contributor           |
| WF-9 | Release                                   | Maintainer            |

## WF-1: install gnux and link the tools

- **Actors.** A command-line user, or an image build.
- **Preconditions.** Go at the version in `go.mod`, and `git`. For the per-tool commands, a
  filesystem with symbolic links and the install directory on `PATH`.
- **Exercises.** FR-17, FR-21, FR-36, FR-37, FR-38, NFR-2, NFR-21.

**Steps.**

1. Install the binary, by one of four routes.

   ```bash
   go install github.com/braswelljr/gnux@latest   # the binary only
   ./install.sh                                    # the same, plus the per-tool links
   make install/links                              # from a checkout, with the version injected
   docker build -t gnux .                          # an image with every tool linked
   ```

2. Link the tool names to the binary, unless the route already did.

   ```bash
   bin="$(dirname "$(command -v gnux)")"
   for t in $(gnux --list); do ln -sf gnux "$bin/$t"; done
   ```

3. Check the installation.

   ```bash
   gnux --version
   rmx --version
   ```

**Outcome.** `gnux` is on `PATH`, and each of the 22 tools is a command of its own that reports
`TOOL (gnux) VERSION`.

**Failure paths.**

| What happens                                           | Why                                                              | What to do                                                        |
|--------------------------------------------------------|------------------------------------------------------------------|-------------------------------------------------------------------|
| `install.sh` stops with "the Go toolchain is required" | Go is not installed                                              | Install Go and run it again                                       |
| `rmx: command not found`                               | The links were not created, or the directory is not on `PATH`    | Run step 2; add the directory to `PATH`                           |
| `ln` fails on Windows                                  | Creating a symbolic link needs Developer Mode or an elevated shell | Copy `gnux.exe` to `rmx.exe`: a copy dispatches the same way    |
| `gnux --version` prints `gnux dev`                     | The binary has neither the version link flag nor a module version in its build info, as with `go run` or a build outside a module | Install a tagged version (`go install ...@v0.1.0`), or build with `make install` |
| `install.sh` stops with "unknown option"               | An argument other than `--from-source` or `--no-links`           | Remove it; a specific version is chosen with `VERSION=v1.2.3`     |

## WF-2: run a tool, both ways

- **Actors.** A command-line user.
- **Preconditions.** WF-1.
- **Exercises.** FR-1 to FR-8, FR-18 to FR-20, FR-73, FR-75, FR-76.

**Steps.**

1. Run a tool under its own name, exactly as the GNU tool would be run.

   ```bash
   rmx -rf build/
   ```

2. Or run it as a subcommand of the one binary. The result is the same.

   ```bash
   gnux rmx -rf build/
   ```

3. Either form also answers to the GNU name: `gnux rm -rf build/`, or a link called `rm`.
4. Read the exit status: 0 on success, 1 when an operation failed, and the tool's own values
   where GNU defines them.

**Outcome.** The operation is done, anything written to standard output is the tool's result,
and anything on standard error is a diagnostic prefixed with the tool's name.

**Failure paths.**

| What happens                                                    | Status | What it means                                                        |
|-----------------------------------------------------------------|--------|----------------------------------------------------------------------|
| `rmx: cannot remove 'x': No such file or directory`             | 1      | The operation failed. Other operands were still processed            |
| `rmx: missing operand`, then `Try 'rmx --help' ...`             | 1      | A command-line mistake. Nothing was done                             |
| `rmx: unknown flag: --no-such-flag`, then the same hint         | 1      | The option is not implemented. It is refused, never ignored          |
| `gnux: unknown tool 'frobx'`, then `Try 'gnux --help' ...`      | 1      | The first argument is not a tool                                     |
| A prompt on standard error                                      | none   | `-i`, or a write-protected file on a terminal. Answer `y` to proceed |

## WF-3: write and run a portable script

- **Actors.** A script or CI author.
- **Preconditions.** The gnux binary on the machine. Nothing else.
- **Exercises.** FR-23 to FR-29, FR-77 to FR-79, NFR-18.

**Steps.**

1. Write the script. Commands are the tools, under either name.

   ```bash
   # build.gnux
   OUT=dist
   rm -rf $OUT && mkdir -p $OUT/logs
   cp -r assets $OUT/
   ls -1 $OUT | wc -l > $OUT/count.txt
   cat $OUT/count.txt || echo "nothing built"
   ```

2. Run it, stopping at the first failure.

   ```bash
   gnux shell -e build.gnux
   ```

3. Or run a one-liner, pipe a script in, or start an interactive session.

   ```bash
   gnux shell -c 'catx *.log | wcx -l'
   echo 'pwdx' | gnux shell
   gnux shell
   ```

4. Use the same file on every developer machine and every CI runner.

**Outcome.** The script's effects, and an exit status equal to that of the last command run.

**Failure paths.**

| What happens                                          | Status | Why                                                                    |
|-------------------------------------------------------|--------|------------------------------------------------------------------------|
| `gnux: syntax error: ...`                             | 2      | An unterminated quote, a dangling `&&`, a redirection with no file name. A script stops; an interactive session carries on |
| `gnux: NAME: command not found`                       | 127    | Not a builtin or a tool. External programs are run as `envx NAME ...`  |
| `gnux: FILE: No such file or directory` at start      | 127    | The script file could not be opened                                    |
| `gnux shell: extra operand 'b'`, then the hint        | 2      | More than one script, or a script together with `-c`                   |
| `gnux: ambiguous redirect`                            | 1      | A redirection target expanded to more than one word                    |
| A tool's own diagnostic, and the script continues     | varies | Without `-e`, a failing command does not stop the script               |
| A `for` loop or a function is a syntax error or an unknown command | varies | The language has neither, by design                          |

## WF-4: embed an engine

- **Actors.** A Go developer.
- **Preconditions.** A Go module on Go 1.25 or newer.
- **Exercises.** FR-30 to FR-34, NFR-16, NFR-36, NFR-39.

**Steps.**

1. Add the module.

   ```bash
   go get github.com/braswelljr/gnux
   ```

2. Build the engine from the options the flags would set, and the streams to use. Everything
   not set is the tool's default, safeguards included.

   ```go
   engine := rm.New(rm.Options{Recursive: true, Force: true}, core.Stdio())
   ```

3. Run it against the operands and read the error.

   ```go
   err := engine.Run([]string{"build", "dist"})
   if errors.Is(err, core.ErrFailed) {
       // Some paths could not be removed; the diagnostics already went to the error stream.
   }
   ```

4. To capture output, pass buffers instead of `core.Stdio()`; to silence it, `core.Discard()`.
5. To answer prompts or control the clock, set the engine's injection fields (`PromptFn`,
   `Now`, and so on) before calling `Run`.

**Outcome.** The operation happens in the caller's process, with the caller's streams, and the
result is an error value the caller can branch on.

**Failure paths.**

| Returned error      | Meaning                                                | What the caller does                         |
|---------------------|--------------------------------------------------------|----------------------------------------------|
| `core.ErrFailed`    | Ran; one or more operands failed; already reported     | Treat as failure. Do not print the error     |
| `*core.UsageError`  | The arguments were wrong (no operands, a bad value)    | Fix the call. Its message says what was wrong |
| `*core.ExitError`   | A specific status (`lsx` 2, `envx` 125 to 127)         | Read `Code`                                  |
| A nil stream panics | `core.Streams` requires all three fields               | Start from `core.Discard()` and override     |

## WF-5: run a tool or a script by name from Go

- **Actors.** A Go developer whose program receives a command line, or a script, as data.
- **Preconditions.** As WF-4.
- **Exercises.** FR-35, FR-80, FR-81.

**Steps.**

1. Run one tool by name. Either the gnux name or the GNU name is accepted.

   ```go
   status, ok := cmd.RunTool("rmx", []string{"-rf", "build"}, core.Stdio())
   ```

2. Or run a whole script in-process.

   ```go
   sh := cmd.NewShell(shell.Options{ErrExit: true}, core.Stdio())
   sh.Set("OUT", "dist")
   err := sh.RunString("mkdirx -p $OUT && cpx -r assets $OUT/")
   ```

3. Or drive the shell one line at a time, keeping its state between calls.

   ```go
   status := sh.Eval("cd dist")
   status = sh.Eval("lsx -l")
   ```

4. Or run the entire command line and receive the status instead of exiting.

   ```go
   status := cmd.Main([]string{"gnux", "lsx", "-la"}, core.Stdio())
   ```

**Outcome.** The same result the command line would give, as a return value.

**Failure paths.**

| What happens                          | Meaning                                                                                     |
|---------------------------------------|---------------------------------------------------------------------------------------------|
| `ok` is false                         | No tool has that name. Nothing ran                                                          |
| `status` is nonzero                   | The tool failed; its diagnostics went to the error stream given                             |
| `RunString` returns `*core.ExitError` | The last command failed; `Code` carries its status                                          |
| Two scripts interfere                 | `cd` is process-wide while a script runs. Do not run two shells concurrently in one process |

## WF-6: try gnux in the playground

- **Actors.** An evaluator, with a browser and nothing installed.
- **Preconditions.** A browser with WebAssembly support.
- **Exercises.** FR-47, FR-48, FR-84, FR-85, NFR-11, NFR-19.

**Steps.**

1. Open the playground page of the website.
2. The page loads Go's loader script and `gnux.wasm` from the site, creates an in-memory
   filesystem seeded with a small home directory, and starts gnux against it.
3. Type a command, or pick one of the examples. The line runs in a persistent `gnux shell`
   session, so `cd` and variables carry over.
4. Recall earlier commands, and complete command and file names, from the keyboard.
5. Type `/help` for every playground command and tool, and `/fullscreen` to give the terminal
   the whole screen.
6. Run `/refresh` to bring every file back and start a fresh session.

**Outcome.** The visitor has run the real tools and seen their real output and diagnostics.
Nothing on their machine was read or written.

**Failure paths.**

| What happens                                         | Why                                                                             |
|------------------------------------------------------|---------------------------------------------------------------------------------|
| The page reports that WebAssembly is not supported   | The browser cannot run the build                                                |
| The page reports that it could not download the build | `gnux.wasm` or `wasm_exec.js` was not served                                   |
| `gnux: output truncated after 1048576 bytes`         | The line wrote more than the output budget, and was stopped                     |
| A command never returns                              | It waits without writing (`sleepx infinity`). Esc or Ctrl+C resets the session  |
| `envx PROGRAM` fails                                 | There are no external programs in a browser                                     |
| A write fails with "no space left on device"         | The in-memory filesystem has a size limit                                       |

The playground's interface is being built at the time of writing. The WebAssembly build, the
in-memory filesystem, completion and their tests exist; this flow describes the page they are
for.

## WF-7: add a new tool, end to end

- **Actors.** A contributor.
- **Preconditions.** A checkout, Go and golangci-lint v2 (lefthook for the Git hooks). The GNU tool's documented
  behavior, as the specification.
- **Exercises.** FR-1, FR-2, FR-39, FR-41, NFR-3, NFR-25, NFR-26, NFR-32, NFR-43.

**Steps.**

1. **Write the engine** in `core/<gnu>/`: `Name`, `Options`, `New` and `Run`, with a black-box
   `<gnu>_test.go` that drives it against `t.TempDir()` and in-memory streams. Start from
   `core/pwd` (small) or `core/rm` (full-featured).
2. **Write the adapter** in `cmd/<gnu>x/`: `Spec`, `Meta`, `Register`, `buildOptions` and `run`,
   with a table test of `buildOptions`. Start from `cmd/pwdx`.
3. **Register it**: add `<gnu>x.Spec()` to `Tools()` in `cmd/registry.go`, keeping the list
   alphabetical.
4. **Regenerate the documentation**: `make docs`.
5. **Tick it off**: add the tool to the README's table and to the list in `AGENTS.md`, add its
   requirement to [functional-requirements](functional-requirements.md), and remove it from the
   roadmap.
6. **Run the whole check**: `make ci`.
7. **Commit** the code and the regenerated files together.

**Outcome.** The tool is a subcommand, a command of its own under both names, a page in every
reference format and on the website, and a command in the shell and the playground.

**Failure paths.**

| What fails                                   | Why                                                                                 |
|----------------------------------------------|-------------------------------------------------------------------------------------|
| `TestRegistryIsConsistent`                   | The name is not the GNU name plus `x`; `Use` does not start with it; the documentation is incomplete; a name is registered twice |
| `TestEveryToolAnswersHelpAndVersion`         | The adapter defined `--help` or `--version` itself, or `Run` is missing             |
| The WebAssembly build in the Test workflow   | Platform-specific code has no fallback that compiles for `js/wasm`                  |
| The Build workflow, on one platform          | The same, for an operating system                                                   |
| The Docs workflow                            | The regenerated files were not committed                                            |
| The Lint workflow                            | A linter finding, or a misspelling                                                  |

The technical detail of each step is in [adding-a-tool](../workflows/adding-a-tool.md).

## WF-8: regenerate the documentation

- **Actors.** A contributor who changed a flag, a tool's metadata, the guide, or a hand-written
  document.
- **Preconditions.** A checkout and Go. Run from the repository root.
- **Exercises.** FR-41, FR-42, FR-46, FR-82, NFR-29.

**Steps.**

1. Make the change at its source: a `Register` function, a `Meta()`, `cmd/docs/guide.go`, or a
   file under `docs/requirements`, `docs/rules` or `docs/workflows`.
2. Regenerate.

   ```bash
   make docs            # or: go generate ./...
   ```

3. Review the difference. It should contain what the change implies and nothing else.
4. Commit the source change and the regenerated files together.

**Outcome.** `documentation/markdown`, `documentation/man`, `documentation/rest`, `documentation/yaml` and `web/content/docs` match
the code.

**Failure paths.**

| What happens                                                  | Why                                                                              |
|---------------------------------------------------------------|----------------------------------------------------------------------------------|
| The Docs workflow fails with "Generated docs are out of date" | The regenerated files were not committed                                         |
| An edit to a generated page disappears                        | Each generated directory is rebuilt from scratch. Change the source instead      |
| `make docs/check` fails locally with uncommitted hand-written edits | It reports any uncommitted change under `docs/`, hand-written files included. Commit, then check |
| A page on the website shows a brace or an angle bracket oddly | Prose outside code spans is escaped for MDX. Put such characters in a code span  |
| A link in a hand-written page goes to GitHub, not the site    | Only links to `.md` files directly inside `documentation/guides`, the one published folder, become site links |

The technical detail is in
[documentation-generation](../workflows/documentation-generation.md).

## WF-9: release

- **Actors.** The maintainer, then the Release workflow.
- **Preconditions.** `main` is green in the Test, Build, Lint, Docs and Web workflows.
- **Exercises.** FR-36, FR-38, NFR-1, NFR-2, NFR-37.

**Steps.**

1. Run the whole check locally: `make ci`.
2. Confirm the README's tool table, the roadmap, `AGENTS.md` and the requirement registers
   describe what is being released.
3. Tag the commit and push the tag. A tag with a hyphen (`v1.0.0-rc.1`) becomes a prerelease.

   ```bash
   git tag -a v0.1.0 -m "gnux v0.1.0"
   git push origin v0.1.0
   ```

4. The Release workflow runs on the tag:
   1. **verify** repeats vet, the race-enabled tests and the docs freshness check against the
      tagged commit, so a tag cannot publish what CI would reject;
   2. **build** cross-compiles `gnux` with `CGO_ENABLED=0` and the tag injected as the version,
      for linux and darwin on amd64 and arm64 and windows on amd64, and packages each with the
      `LICENSE`, the README and (on Unix) the man pages: `gnux-v0.1.0-linux-amd64.tar.gz`,
      `gnux-v0.1.0-windows-amd64.zip`, and so on;
   3. **publish**, the only job with write permission, writes `checksums.txt` (SHA-256 of every
      archive) and creates the GitHub release with generated notes and every file attached.
5. Users install that version either way:

   ```bash
   go install github.com/braswelljr/gnux@v0.1.0        # reports v0.1.0 (build info)
   curl -LO https://github.com/braswelljr/gnux/releases/download/v0.1.0/gnux-v0.1.0-linux-amd64.tar.gz
   sha256sum --check --ignore-missing checksums.txt
   ```

**Outcome.** A tag that `go install` resolves, a GitHub release with a verified archive per
platform, and binaries that report the tag as their version.

**Failure paths and known gaps.**

| What happens                                                 | Why                                                                         |
|--------------------------------------------------------------|-----------------------------------------------------------------------------|
| The release job fails at **verify**                          | The tagged commit fails a test or has stale docs. Fix on `main`, delete the tag, tag again |
| `gh release create` fails with "already exists"              | The tag was released before. Delete the release, or tag a new version       |
| The man page date is fixed                                   | `docs.ManDate` is pinned for reproducibility, and nothing sets it per release |
| No package-manager formula                                   | Out of scope (see [business-requirements](business-requirements.md))        |

The technical detail is in [ci-and-release](../workflows/ci-and-release.md).
