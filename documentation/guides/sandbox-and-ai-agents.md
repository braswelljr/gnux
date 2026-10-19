# Sandbox and AI agents

gnux can confine itself with the operating system's own sandbox, so that a script, or an AI agent working through `gnux mcp`, can read and write the directories you name and nothing else. The kernel enforces it: a symbolic link, `..`, or a program started through `envx` cannot get around it, and once entered it cannot be lifted.

## What the sandbox allows

| | Can read | Can write |
|---|---|---|
| Each `--sandbox DIR` (or each `DIR` of `gnux mcp`) | Yes | Yes |
| Each `--sandbox-read DIR` (`--read DIR` for `gnux mcp`) | Yes | No |
| The system's programs, libraries and configuration | Yes | No |
| `/dev/null` (`NUL` on Windows) | Yes | Yes |
| Everything else: your home directory, `~/.ssh`, other projects, `/tmp` | No | No |

The system directories are `/usr`, `/bin`, `/sbin`, `/lib*`, `/etc`, `/opt` and `/proc` on Linux; `/usr`, `/bin`, `/sbin`, `/System`, `/Library`, `/opt/homebrew` and `/private/etc` on macOS; on Windows, whatever Windows already opens to every AppContainer (`C:\Windows`, `C:\Program Files`); `/etc`, `/usr/share` and `/usr/local/share` on FreeBSD.

On FreeBSD the sandbox is Capsicum's capability mode, in which a process can open nothing by path, only relative to directories it already holds. gnux opens the allowed directories first, sends every tool's file access through them, and then enters capability mode, so the kernel refuses anything else. A symbolic link is followed only where it leads into an allowed directory.

## Platforms

| System | Sandbox | Needs |
|--------|---------|-------|
| Linux | Landlock | Linux 5.13 or later with Landlock enabled (every current distribution); a gnux built with `CGO_ENABLED=0`, as every release is |
| macOS | Seatbelt (`sandbox-exec`) | Nothing: `sandbox-exec` ships with every macOS |
| Windows | AppContainer | Windows 8 or later |
| FreeBSD | Capsicum (capability mode) | Nothing: Capsicum is in every FreeBSD kernel. A command can run gnux's own tools only, since capability mode starts no other program |

Where the sandbox cannot be entered, gnux says why and exits with status 125 without running anything. It never falls back to running unconfined.

On Windows the first run with a set of directories grants gnux's sandbox access to them: an access entry for the sandbox's own identity on each directory, inherited by everything inside. On a large tree that first run takes a moment; later runs with the same directories start at once. `icacls DIR` shows the entry, and `icacls DIR /remove *S-1-15-2-...` takes it away.

## A sandboxed script

```bash
# The build script may change the project and nothing else.
gnux shell --sandbox . -e ./build.gnux

# Write only in dist, read the shared configuration too.
gnux shell --sandbox dist --sandbox-read config -c 'rmx -rf dist/* && cpx config/app.json dist/'
```

The sandbox is entered before the script is read, so the script itself must lie in an allowed directory, or be given with `-c`, or piped in.

## gnux mcp: file tools for an AI agent

`gnux mcp DIR...` is a Model Context Protocol server on standard input and output. It enters the sandbox first, then offers the agent one tool, `run`, which runs a gnux shell command line in a directory of the agent's choosing inside DIR and returns its output, errors and exit status. Each call is a process of its own, stopped after `--timeout` (two minutes by default) and cut off after `--max-output` bytes of each stream (1 MiB), so neither a loop nor an endless `yesx` can hold the server up.

### Claude Code

```bash
claude mcp add gnux -- gnux mcp "$PWD"
```

### Claude Desktop, Cursor and others

The usual `mcpServers` configuration (`claude_desktop_config.json`, `.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "gnux": {
      "command": "gnux",
      "args": ["mcp", "/Users/you/src/app", "--read", "/Users/you/docs/style-guide"]
    }
  }
}
```

### VS Code

`.vscode/mcp.json`:

```json
{
  "servers": {
    "gnux": {
      "type": "stdio",
      "command": "gnux",
      "args": ["mcp", "${workspaceFolder}"]
    }
  }
}
```

An agent then runs, for example:

```text
run {"command": "findx . -name '*.go' -newer go.mod | headx -20"}
run {"command": "grepx -rn TODO src", "directory": "services/api"}
run {"command": "rmx -rf ~/.ssh"}   ->  rmx: cannot remove '/Users/you/.ssh': Permission denied
```

## Advantages and limits

What the sandbox gives you:

- **Real protection for commands you did not write.** An agent's mistake, or a hostile instruction it picked up from a web page, cannot delete or read anything outside the directories you named: not your home directory, not your SSH keys, not your other projects.
- **Kernel enforcement.** It does not rely on gnux checking paths, so there is no path trick to find, and it covers the programs a command starts.
- **Nothing to set up.** One flag; no container, no virtual machine, no daemon; the tools run at full speed.

What it does not do, or costs:

- **Files only.** On Linux and macOS it does not limit the network, CPU or memory; on Windows an AppContainer has no network access at all, so `envx curl ...` fails there.
- **Stricter than you may expect.** `/tmp` and the home directory are out of reach unless you name them, so a command that writes a temporary file fails until you allow a directory for it.
- **File names may show on macOS.** Seatbelt lets `stat` see the names and sizes of files outside the sandbox, never their contents.
- **Platform requirements.** Linux needs Landlock; macOS relies on `sandbox-exec`, which Apple marks deprecated but still ships and uses itself; on FreeBSD the commands can run gnux's own tools but no other program (`envx curl ...` fails), and `--sandbox-read` is kept read-only by gnux itself rather than by the kernel, which there confines gnux to the directories but not to reading them.
- **Within the directories, anything goes.** The agent can delete everything in the directory you give it, as you could. Give it the project it is working on, under version control, not a directory above it.
