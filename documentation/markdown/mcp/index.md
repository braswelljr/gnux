## mcp

Serve the `gnux` tools to an AI agent over MCP, sandboxed to DIR.

### Synopsis

```bash
gnux mcp [OPTION]... [DIR]...
```

### Description

`gnux` mcp is a Model Context Protocol server on standard input and output, for an AI agent (Claude Code, Claude Desktop, Cursor, VS Code, ...) to manage files with. It offers one tool, run, which runs a `gnux shell` command line (pipelines, globs, if, for, every `gnux` tool under either name) and returns its output, errors and exit status.

Before it answers anything the server confines itself with the operating system's sandbox, as `gnux shell --sandbox` does: the commands may read and write each DIR (the current directory when none is given) and the `--read` directories may only be read; the system's own directories stay readable, and everything else, the home directory included, can be neither read nor written. The kernel enforces it, on the tools and on any program a command starts, so no command line can get around it: not a symbolic `link`, not .., not `envx`. The sandbox is Landlock on Linux (5.13 or later), Seatbelt on macOS, an AppContainer on Windows and Capsicum on FreeBSD (where each call enters it, and runs `gnux`'s own tools only); where there is none `gnux` mcp refuses to start rather than serve unconfined.

Each call runs in a process of its own, in the first DIR or in the directory the call names beneath one, with no standard input. It is stopped after --`timeout` (two minutes by default), and its output is cut off after `--max-output` bytes of each stream, so that neither a loop nor `yesx` can hold the server up.

### Permissions

The commands can do in each DIR whatever the user running the server may do there, removing files included; give it the project it is meant to work on, not a directory above it.

### Use cases

- Letting an AI agent list, search, copy, move and remove files in one project, and nowhere else.
- The same file tools for an agent on Linux, macOS, Windows and FreeBSD, with GNU's flags and messages.
- Giving an agent read-only access to reference material next to a writable project.

### Flags

```text
      --help               display this help and exit
      --max-output BYTES   keep at most BYTES of each stream of a call (default 1048576)
      --read DIR           also let the commands read DIR; may be repeated
      --timeout DURATION   stop a call after DURATION (as 30s, 5m) (default 2m0s)
      --version            output version information and exit
```

### Examples

Serve the current project to Claude Code:

```bash
claude mcp add gnux -- gnux mcp "$PWD"
```

```powershell
claude mcp add gnux -- gnux mcp $PWD
```

Let the agent write in the project but only read the shared docs:

```bash
gnux mcp --read ~/docs/style-guide ~/src/app
```

```powershell
gnux mcp --read $HOME\docs\style-guide $HOME\src\app
```

Allow long builds and more output:

```bash
gnux mcp --timeout 10m --max-output 4194304 .
```

```powershell
gnux mcp --timeout 10m --max-output 4194304 .
```

### Exit status

| Status | Meaning |
|--------|---------|
| 0 | the client closed the connection |
| 2 | a command-line mistake |
| 125 | the sandbox could not be entered (a Linux kernel without Landlock, a build with cgo); nothing was served |

### Help in the terminal

```bash
gnux help mcp             # this page: the full manual
gnux help mcp --OPTION    # one option in depth, with its examples
```

### See also

- [shell](../shell/index.md)
- [gnux](../index.md) - every tool in the bundle
- [guide](../guide.md) - installation and usage
