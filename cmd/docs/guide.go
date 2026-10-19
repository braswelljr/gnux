package docs

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/braswelljr/gnux/internal/common"
)

// section is one titled part of the guide. The same sections feed every
// format: cobra-driven ones get "TITLE. body" paragraphs (plain text, so they
// render cleanly in man, yaml and reST), the website gets real headings.
type section struct {
	title string
	body  []string // paragraphs
}

// guideIntro opens the guide, before the first section.
const guideIntro = `gnux brings the interface of the GNU core utilities to every platform: one binary of ` +
	`GNU-compatible tools, written in Go. Every tool is named after its GNU counterpart with an x appended - rm is rmx, ` +
	`cp is cpx, ls is lsx - and mirrors its flags, semantics, diagnostics and exit statuses, on ` +
	`Linux, macOS and Windows alike. It can be used three ways: as commands, as a Go library, and ` +
	`through gnux shell, a portable script interpreter built on the tools. gnux is an independent ` +
	`project, not affiliated with or endorsed by the GNU Project or the FSF; it complements GNU ` +
	`coreutils rather than replacing them.`

// guideSections is the body of the guide, in reading order.
var guideSections = []section{
	{"Requirements", []string{
		`Running gnux needs nothing but the binary: it is statically linked, has no runtime ` +
			`dependencies, and supports Linux on amd64, arm64 and riscv64, macOS and FreeBSD on amd64 ` +
			`and arm64, and Windows on amd64 and arm64.`,
		`Installing with go install, building from source, or importing the engines as a library ` +
			`needs Go 1.26 or newer, plus git.`,
		`The per-tool commands are links to the one binary, so they need a filesystem with symbolic ` +
			`links and the install directory on PATH; on Windows, where creating a symlink needs ` +
			`Developer Mode or an elevated shell, copy gnux.exe to the tool's name instead.`,
		`make, bash and Docker are optional conveniences. Working on the website additionally needs ` +
			`Node.js 22.18 or newer and pnpm.`,
	}},
	{"Installation", []string{
		`Install the latest version with go install, or build from source with make build (the ` +
			`binary lands in ./bin) or make install. The version string is injected at build time and ` +
			`reported by --version.`,
		`To make each tool a command of its own, link the binary to the tool names: make links does ` +
			`this for a source build, make install/links and install.sh do it next to the installed ` +
			`binary, and the examples show the loop to run by hand.`,
		`Each release is also published as an archive per platform with a checksums file, as ` +
			`.deb, .rpm and .apk packages for Linux, and to Homebrew (brew tap braswelljr/gnux ` +
			`https://github.com/braswelljr/gnux, then brew install --cask braswelljr/gnux/gnux), Scoop ` +
			`(scoop bucket add gnux https://github.com/braswelljr/gnux) and winget (winget install braswelljr.gnux), the Snap Store ` +
			`(snap install gnux) and the AUR (gnux-bin). In GitHub Actions, the step uses: ` +
			`braswelljr/gnux@v0 installs a release on any runner and puts every tool on PATH.`,
		`To install one particular version, name it: go install github.com/braswelljr/gnux@v0.0.2, ` +
			`winget install braswelljr.gnux --version 0.0.2, scoop install gnux/gnux@0.0.2, or ` +
			`the archive or package of that release. gnux --version reports the version installed.`,
		`gnux completion SHELL prints a tab-completion script for bash, zsh, fish or powershell; ` +
			`the release archives carry them pre-generated.`,
	}},
	{"Usage", []string{
		`There are two ways to run a tool. As a subcommand, pass the tool name first: gnux rmx -rf ` +
			`build. As a command of its own, invoke the binary under the tool's name: gnux looks at the ` +
			`name it was started with, so a link (or copy) of gnux called rmx behaves as rmx.`,
		`Both forms also answer to the GNU name, so scripts written for the GNU tools run unchanged through a link called rm. Every ` +
			`tool accepts --help and --version; gnux --list prints the name of every tool.`,
		`gnux help TOOL prints a tool's full manual in the terminal: synopsis, description, every ` +
			`option, permissions, use cases, every example for bash and PowerShell, exit status and ` +
			`related tools. gnux help TOOL OPTION (gnux help rmx force, gnux help lsx -l) explains one ` +
			`option with the examples that use it, and gnux help on its own lists every command. Help ` +
			`is colored on a terminal and plain when piped or when NO_COLOR is set.`,
	}},
	{"Shell", []string{
		`gnux shell is a small interpreter whose commands are the gnux tools, run inside the gnux ` +
			`process. It understands pipelines, the &&, || and ; connectors, redirection, variables, ` +
			`quoting, globbing and comments - a deliberately small subset of POSIX sh - and nothing in ` +
			`it depends on the system's own shell, so one script runs unchanged on every platform.`,
		`Start it with no arguments for an interactive session, pass a script file, or give a ` +
			`one-liner with -c. Add -e to stop at the first failing command.`,
	}},
	{"Library", []string{
		`gnux is also a Go module. Each tool's engine is an importable package under core/ - ` +
			`core/rm, core/cp, core/ls and so on - taking an Options struct and a set of streams, so an ` +
			`application can remove, copy or list files with GNU semantics without spawning a process.`,
		`To run a tool by name, call cmd.RunTool; to run a whole portable script, build a shell with ` +
			`cmd.NewShell. Add the module with go get.`,
	}},
	{"Exit status", []string{
		`Tools exit 0 on success and 1 when an operation fails, with the diagnostic on standard ` +
			`error prefixed by the tool name. A command-line mistake prints a pointer to --help.`,
		`Tools whose status carries more meaning keep it: lsx exits 2 on serious trouble and envx ` +
			`uses 125 to 127. gnux shell exits with the status of the last command, 2 for a syntax ` +
			`error and 127 for an unknown command.`,
	}},
	{"Workflows", []string{
		`Everyday use: install once, link the tool names, then use rmx, cpx, lsx and the rest ` +
			`exactly as you would the GNU originals - including on Windows, where they are otherwise ` +
			`missing.`,
		`Cross-platform scripting: put the steps of a build, clean or packaging job in a script and ` +
			`run it with gnux shell -e. The only thing a machine needs is the gnux binary, so the same ` +
			`script serves a developer laptop and every CI runner.`,
		`Embedding: import the engine you need from core/ and call it with your own streams; check ` +
			`the returned error against core.ErrFailed to tell "ran but something failed" from success.`,
		`Contributing: add an engine under core/ and its adapter under cmd/, register it, then run ` +
			`make ci. That vets, lints and tests the code and regenerates this reference; CI runs the ` +
			`same steps on Linux, macOS and Windows and fails if the committed docs are stale.`,
	}},
}

// guideLong renders the guide as plain prose: the intro, then each section as
// "TITLE. paragraph paragraph".
func guideLong() string {
	var b strings.Builder
	b.WriteString(guideIntro)
	for _, s := range guideSections {
		b.WriteString("\n\n")
		b.WriteString(strings.ToUpper(s.title))
		b.WriteString(". ")
		b.WriteString(strings.Join(s.body, " "))
	}
	return b.String()
}

// guideExample is the worked-examples block for the guide.
const guideExample = `  # install the latest gnux
  go install github.com/braswelljr/gnux@latest

  # run a tool as a subcommand
  gnux lsx -la
  gnux rmx -rf build/

  # make every tool a command of its own, then use it directly
  bin="$(dirname "$(command -v gnux)")"
  for t in $(gnux --list); do ln -sf gnux "$bin/$t"; done
  rmx -rf build/

  # run a portable script, stopping at the first failure
  gnux shell -e build.gnux

  # use the engines from Go
  go get github.com/braswelljr/gnux`

// guideCommand builds the synthetic command backing the guide page. It carries
// gnux's own flags so the guide's options section matches the entry page.
func guideCommand() *cobra.Command {
	c := &cobra.Command{
		Use:                   common.AppName + " [OPTION]... [TOOL [ARGUMENT]...]",
		Short:                 "Requirements, installation and usage guide for gnux, the GNU core utilities in one cross-platform binary.",
		Long:                  guideLong(),
		Example:               guideExample,
		DisableFlagsInUseLine: true,
		Run:                   func(*cobra.Command, []string) {},
	}
	c.DisableAutoGenTag = true
	c.Flags().Bool("help", false, "display this help and exit")
	c.Flags().Bool("list", false, "print the name of every tool, one per line")
	c.Flags().Bool("version", false, "output version information and exit")
	return c
}
