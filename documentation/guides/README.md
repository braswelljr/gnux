# Guides

Task-oriented guides for gnux, a portable, GNU-compatible implementation of the GNU core utilities' interface in Go: one statically linked binary that carries 108 tools, a Go library with an importable engine for every tool, and a portable shell that runs the same script on Linux, macOS and Windows. The reference pages describe each tool's flags one by one; these guides show how to put the tools to work, with hundreds of runnable examples, complete scripts, CI configurations and Go programs. The gnux command lines and shell scripts were run against a real gnux build, the outputs shown are captured from it, and every Go program was compiled and run.

## Pick a guide

| Guide | For | What you get |
|-------|-----|--------------|
| [Getting started](getting-started.md) | Everyone | Every way to install, linking the tool names, the two invocation forms, GNU-name aliases, `--help`, `--version` and `--list` |
| [Cookbook](cookbook.md) | Command-line users | Over 90 recipes grouped by task: inspecting files, copying and linking, safe deletion, backups, logs, paths, environment, timestamps, generated data and pipelines |
| [Portable scripting](portable-scripting.md) | Script and build authors | The `gnux shell` language in depth and four complete scripts (scaffold, clean, log rotation, release staging) that behave identically on every platform |
| [Windows and PowerShell](windows-and-powershell.md) | Windows users | Installing on Windows, links versus copies of `gnux.exe`, PowerShell's colliding aliases, quoting, and what degrades on NTFS |
| [Sandbox and AI agents](sandbox-and-ai-agents.md) | Anyone running scripts or agents they do not fully trust | `gnux shell --sandbox`, `gnux mcp` for Claude Code, Claude Desktop, Cursor and VS Code, what is reachable on each platform, and the limits |
| [Node and npm](node-and-npm.md) | JavaScript and TypeScript developers | Installing from npm, replacing rimraf, mkdirp, cpy-cli, shx and cross-env in package scripts, globs on Windows, and the JavaScript API |
| [CI and containers](ci-and-containers.md) | CI and DevOps engineers | GitHub Actions, GitLab CI and Docker recipes, and one `gnux shell` script as a cross-OS build step |
| [Go library](go-library.md) | Go developers | Embedding the engines: options, streams, captured output, injected prompts and clocks, errors, `cmd.RunTool`, `cmd.NewShell`, and seventeen complete programs |
| [Terminal view](terminal-view.md) | Everyone at a terminal | The terminal view: colored columns, trees, icons, git status, usage bars, grep headings, colored checks and reports, dark and light themes, and making your choices the default |
| [Adoption](adoption.md) | Users of GNU coreutils | All 108 tools side by side with their GNU counterparts, supported flags, every documented deviation and what is not implemented yet |
| [Troubleshooting](troubleshooting.md) | Everyone | What each error and exit status means, prompts you did not expect, refusals, and a FAQ |

## By task

| I want to... | Read |
|--------------|------|
| Install gnux and run my first command | [Getting started](getting-started.md) |
| Use `rmx`, `cpx` and `lsx` on Windows | [Windows and PowerShell](windows-and-powershell.md) |
| Find the right flags for an everyday job | [Cookbook](cookbook.md) |
| Replace a pile of `.sh` and `.ps1` files with one script | [Portable scripting](portable-scripting.md) |
| Replace rimraf, mkdirp and shx in my package scripts | [Node and npm](node-and-npm.md) |
| Let an AI agent manage files in one project and nowhere else | [Sandbox and AI agents](sandbox-and-ai-agents.md) |
| Run the same build step on Linux, macOS and Windows runners | [CI and containers](ci-and-containers.md) |
| Delete, copy or list files from a Go program without spawning a process | [Go library](go-library.md) |
| See a colored tree of a project, with git status, and pick a theme | [Terminal view](terminal-view.md) |
| Adopt gnux next to the GNU tools I already use, and know where it differs | [Adoption](adoption.md) |
| Understand an error message or an exit status | [Troubleshooting](troubleshooting.md) |

## Elsewhere in the documentation

- [The reference](https://gnux.vercel.app/docs): one page per tool, with every flag, the permissions it needs, use cases and examples in bash and PowerShell.
- [The shell reference](https://gnux.vercel.app/docs/shell) and the [installation and usage guide](https://gnux.vercel.app/docs/guide).
- [Requirements](../../docs/requirements/functional-requirements.md): what each tool must do, and the [user workflows](../../docs/requirements/srs-workflows.md) step by step.
- [Workflows](../../docs/workflows/README.md): how the same processes work inside the code.
- [Rules](../../docs/rules/README.md): the conventions every tool follows.
