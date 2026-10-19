# GNU Conformance

How gnux is checked against the GNU tools themselves: the harness in `cmd/conformance_test.go` runs each command line in `cmd/testdata/conformance/` once through GNU coreutils, grep, sed and findutils and once through gnux, in two identical fresh directories, and requires the same standard output, standard error and exit status. It is how "GNU is the specification" (FR-166) is enforced rather than remembered, and how a new GNU release that changes a message is noticed.

## Running it

```bash
make conformance           # Linux, with the GNU tools installed (the system's ls must be GNU's)
make conformance/docker    # anywhere with Docker: inside golang:1.26-trixie
go test -tags conformance -run 'TestConformance/text/' ./cmd/   # one file of cases
```

The test is behind the `conformance` build tag, so `go test ./...` never runs it, and it skips itself when `ls --version` does not name GNU coreutils. CI runs it in the `conformance` job of the Test workflow, inside the same Debian image, as an unprivileged user: root ignores permissions, and a case about a refused write would pass for the wrong reason.

## What one case is

A line of a `.txt` file in `cmd/testdata/conformance/` is a command for `sh -c`; `#` starts a comment line, and a line ending in `\` continues on the next. The file name groups the cases (`text`, `files`, `checksums`, `misc`, `grep-sed-find`, `options`, `gnu-options`) and is the first part of the subtest's name.

For each case the harness:

1. builds gnux once and links it under every GNU name, and `[`, in a temporary `bin` directory;
2. creates two directories with the same fixture (`fixture` in the test: text files, a CSV, binary bytes, a file without a final newline, a name with a space, a small tree, a symbolic link), all with the same fixed time;
3. runs the line in the first with the system's `PATH`, and in the second with the gnux `bin` first, both with `LC_ALL=C`, `TZ=UTC`, `COLUMNS=80` and `HOME` set to the directory;
4. rewrites gnux's tool names in the second run's output to the GNU names (`rmx:` becomes `rm:`, `Try 'rmx --help'` becomes `Try 'rm --help'`), and each run's own directory to `$DIR`;
5. compares standard output, standard error and the exit status, and on a difference prints all three side by side.

Shell builtins (`echo`, `printf`, `test`, `[`, `true`, `false`, `pwd`, `kill`) would be the shell's own in both runs, so a case reaches the real program through `env`: `env printf '%s\n' a`.

## Writing cases

- Write the command line the way a GNU user would, with GNU names: the same line runs in both worlds.
- Cover an option's output, its diagnostics and its exit status. `cmd; echo $?` shows a status that the pipeline would hide.
- A documented deviation (a tool's `Meta().Long`) is not a case: the harness is for what must be equal. When a case finds a difference, either fix gnux or, if the difference is deliberate, document it and drop the case.
- Use the fixture; a case that needs another file creates it in its own command line (`cp a.txt e.txt && sed -i ... e.txt`).

## What it has found

The first run found, among about 270 cases: `sort -g` not reading hexadecimal numbers as `strtold` does; `wc` dropping its column-width estimate after a missing first operand, a behavior GNU 9 no longer has; `rm .` refused by name without `-r`; `findx` adding a `--help` hint GNU find never prints; GNU grep 3.11's wording for a trailing `[`; and, for every tool, pflag's option errors where glibc's getopt has its own words and accepts abbreviated long options (FR-165).

## Related

- [command-line-to-tool-run](command-line-to-tool-run.md) for `tool.ParseFlags`, where options are parsed as getopt parses them.
- [benchmarking-and-robustness](benchmarking-and-robustness.md) for the other checks that run the real tools.
- The rule [GNU is the specification](../rules/gnu-is-the-specification-and-differences-are-documented.md).
