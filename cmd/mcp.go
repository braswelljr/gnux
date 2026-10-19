package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/mcp"
	"github.com/braswelljr/gnux/internal/sandbox"
	"github.com/braswelljr/gnux/internal/tool"
)

// mcpName is the command's name, the prefix of its diagnostics.
const mcpName = "gnux mcp"

// MCPSpec describes gnux mcp, the Model Context Protocol server. Like the
// shell it is a gnux command, not a tool: no program name selects it.
func MCPSpec() tool.Spec {
	return tool.Spec{
		Meta: docmeta.Meta{
			Name:    "mcp",
			SeeAlso: []string{"shell"},
			ExitStatus: []docmeta.Status{
				{Code: "0", Meaning: "the client closed the connection"},
				{Code: "2", Meaning: "a command-line mistake"},
				{Code: "125", Meaning: "the sandbox could not be entered (a Linux kernel without Landlock, a build with cgo); nothing was served"},
			},
			Use:   "mcp [OPTION]... [DIR]...",
			Short: "Serve the gnux tools to an AI agent over MCP, sandboxed to DIR.",
			Long: "gnux mcp is a Model Context Protocol server on standard input and output, " +
				"for an AI agent (Claude Code, Claude Desktop, Cursor, VS Code, ...) to manage " +
				"files with. It offers one tool, run, which runs a gnux shell command line " +
				"(pipelines, globs, if, for, every gnux tool under either name) and returns " +
				"its output, errors and exit status.\n\n" +
				"Before it answers anything the server confines itself with the operating " +
				"system's sandbox, as gnux shell --sandbox does: the commands may read and " +
				"write each DIR (the current directory when none is given) and the --read " +
				"directories may only be read; the system's own directories stay readable, " +
				"and everything else, the home directory included, can be neither read nor " +
				"written. The kernel enforces it, on the tools and on any program a command " +
				"starts, so no command line can get around it: not a symbolic link, not .., " +
				"not envx. The sandbox is Landlock on Linux (5.13 or later), Seatbelt on macOS, " +
				"an AppContainer on Windows and Capsicum on FreeBSD (where each call enters it, " +
				"and runs gnux's own tools only); where there is none gnux mcp refuses to start " +
				"rather than serve unconfined.\n\n" +
				"Each call runs in a process of its own, in the first DIR or in the " +
				"directory the call names beneath one, with no standard input. It is " +
				"stopped after --timeout (two minutes by default), and its output is cut " +
				"off after --max-output bytes of each stream, so that neither a loop nor " +
				"yesx can hold the server up.",
			Permissions: "The commands can do in each DIR whatever the user running the " +
				"server may do there, removing files included; give it the project it is " +
				"meant to work on, not a directory above it.",
			UseCases: []string{
				"Letting an AI agent list, search, copy, move and remove files in one project, and nowhere else.",
				"The same file tools for an agent on Linux, macOS, Windows and FreeBSD, with GNU's flags and messages.",
				"Giving an agent read-only access to reference material next to a writable project.",
			},
			Examples: []docmeta.Example{
				{
					Description: "Serve the current project to Claude Code:",
					Bash:        "claude mcp add gnux -- gnux mcp \"$PWD\"",
					PowerShell:  "claude mcp add gnux -- gnux mcp $PWD",
				},
				{
					Description: "Let the agent write in the project but only read the shared docs:",
					Bash:        "gnux mcp --read ~/docs/style-guide ~/src/app",
					PowerShell:  "gnux mcp --read $HOME\\docs\\style-guide $HOME\\src\\app",
				},
				{
					Description: "Allow long builds and more output:",
					Bash:        "gnux mcp --timeout 10m --max-output 4194304 .",
					PowerShell:  "gnux mcp --timeout 10m --max-output 4194304 .",
				},
			},
		},
		Flags: func(fs *pflag.FlagSet) {
			fs.StringArray("read", nil, "also let the commands read `DIR`; may be repeated")
			fs.Duration("timeout", 2*time.Minute, "stop a call after `DURATION` (as 30s, 5m)")
			fs.Int("max-output", 1<<20, "keep at most `BYTES` of each stream of a call")
		},
		Run:       runMCP,
		UsageExit: 2,
	}
}

func runMCP(fs *pflag.FlagSet, args []string, s core.Streams) error {
	read, _ := fs.GetStringArray("read")
	timeout, _ := fs.GetDuration("timeout")
	maxOutput, _ := fs.GetInt("max-output")
	if timeout <= 0 {
		return core.Usagef("invalid timeout: '%s'", fs.Lookup("timeout").Value)
	}
	if maxOutput <= 0 {
		return core.Usagef("invalid output limit: '%d'", maxOutput)
	}
	dirs := args
	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	diag := core.NewDiag(mcpName, s)
	exe, err := os.Executable()
	if err != nil {
		diag.Errorf("%v", err)
		return core.ErrFailed
	}
	runner := &mcpRunner{exe: exe, timeout: timeout, maxOutput: maxOutput}
	if sandboxEachCall {
		// A process in the sandbox cannot start another here (Capsicum), so
		// the server stays outside it and each call enters it on its own;
		// one call now tells whether that works at all.
		runner.sandboxArgs = sandboxFlags(dirs, read)
		if out, err := exec.Command(exe, append(append([]string{"shell"}, runner.sandboxArgs...), "-c", "truex")...).CombinedOutput(); err != nil {
			diag.Errorf("%s", strings.TrimSpace(strings.TrimPrefix(string(out), "shell: ")))
			return &core.ExitError{Code: sandboxRefused}
		}
	} else if err := sandbox.Enter(sandbox.Policy{Write: dirs, Read: read}); err != nil {
		diag.Errorf("%v", err)
		return &core.ExitError{Code: sandboxRefused}
	}
	roots := make([]string, 0, len(dirs))
	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			diag.Errorf("%s: %v", d, err)
			return core.ErrFailed
		}
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			abs = real
		}
		roots = append(roots, abs)
	}
	runner.roots = roots
	server := mcp.Server{
		Name:    common.AppName,
		Version: common.Version,
		Instructions: "gnux runs GNU-compatible file tools (ls, cat, cp, mv, rm, mkdir, find, grep, sed, " +
			"wc, sort, ...) in a sandbox. Use the run tool with a gnux shell command line. Files can be " +
			"read and written only in: " + strings.Join(roots, ", ") + ".",
		Tools: []mcp.Tool{runner.tool()},
	}
	if err := server.Serve(context.Background(), s.In, s.Out); err != nil {
		diag.Errorf("%v", err)
		return core.ErrFailed
	}
	return nil
}

// mcpRunner runs the commands of the run tool.
type mcpRunner struct {
	exe       string
	roots     []string
	timeout   time.Duration
	maxOutput int
	// sandboxArgs, where each call enters the sandbox itself, are the
	// shell's --sandbox and --sandbox-read options.
	sandboxArgs []string
}

// sandboxFlags are the gnux shell options that sandbox a call to write and
// read.
func sandboxFlags(write, read []string) []string {
	var args []string
	for _, d := range write {
		args = append(args, "--sandbox", d)
	}
	for _, d := range read {
		args = append(args, "--sandbox-read", d)
	}
	return args
}

// runResult is the run tool's structured result.
type runResult struct {
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Status    int    `json:"status"`
	TimedOut  bool   `json:"timedOut"`
	Truncated bool   `json:"truncated"`
}

func (r *mcpRunner) tool() mcp.Tool {
	return mcp.Tool{
		Name:  "run",
		Title: "Run a gnux shell command",
		Description: "Run a command line in the gnux shell and return its standard output, standard " +
			"error and exit status. The language is a portable subset of sh: pipelines, &&, ||, ;, " +
			"redirection, variables, $(...), globbing, if, for, while and case. Every command is a " +
			"gnux tool, under its GNU name or with an x appended (rm or rmx, ls or lsx), with GNU's " +
			"options; 'gnux --list' lists them and 'gnux help TOOL' shows a manual. Other programs run " +
			"only through 'envx PROGRAM ARG...'. Files can be read and written only in " +
			strings.Join(r.roots, ", ") + "; the command runs in " + r.roots[0] +
			" unless directory names another directory there. There is no standard input.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "command": {"type": "string", "description": "The gnux shell command line, as 'lsx -la src' or 'grepx -rn TODO . | headx -20'."},
    "directory": {"type": "string", "description": "Where to run it: a path relative to the first allowed directory, or an absolute path inside an allowed directory."}
  },
  "required": ["command"],
  "additionalProperties": false
}`),
		OutputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "stdout": {"type": "string"},
    "stderr": {"type": "string"},
    "status": {"type": "integer", "description": "The exit status; -1 when the command was stopped, by the time limit or by its output reaching the limit."},
    "timedOut": {"type": "boolean"},
    "truncated": {"type": "boolean", "description": "An output stream was longer than the limit and was cut off."}
  },
  "required": ["stdout", "stderr", "status", "timedOut", "truncated"]
}`),
		Call: r.call,
	}
}

// call runs one command line.
func (r *mcpRunner) call(ctx context.Context, arguments json.RawMessage) mcp.Result {
	var in struct {
		Command   string `json:"command"`
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal(arguments, &in); err != nil || strings.TrimSpace(in.Command) == "" {
		return mcp.Result{Text: "run needs a command, as {\"command\": \"lsx -l\"}", IsError: true}
	}
	dir, err := r.directory(in.Directory)
	if err != nil {
		return mcp.Result{Text: err.Error(), IsError: true}
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	args := append(append([]string{"shell"}, r.sandboxArgs...), "-c", in.Command)
	c := exec.CommandContext(ctx, r.exe, args...)
	c.Dir = dir
	stdout, stderr := &cappedBuffer{limit: r.maxOutput}, &cappedBuffer{limit: r.maxOutput}
	c.Stdout, c.Stderr = stdout, stderr
	stopGroup(c)
	err = c.Run()

	res := runResult{Stdout: stdout.String(), Stderr: stderr.String(), Truncated: stdout.cut || stderr.cut}
	var exit *exec.ExitError
	switch {
	case ctx.Err() != nil:
		res.Status, res.TimedOut = -1, true
	case errors.As(err, &exit):
		res.Status = exit.ExitCode()
	case err != nil:
		return mcp.Result{Text: "could not run the command: " + err.Error(), IsError: true}
	}
	return mcp.Result{Text: res.text(r.timeout), Structured: res, IsError: res.Status != 0}
}

// directory resolves where a call runs, refusing anywhere outside the roots.
func (r *mcpRunner) directory(dir string) (string, error) {
	if dir == "" {
		return r.roots[0], nil
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(r.roots[0], dir)
	}
	dir = filepath.Clean(dir)
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	for _, root := range r.roots {
		if rel, err := filepath.Rel(root, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return dir, nil
		}
	}
	return "", fmt.Errorf("%s is outside the allowed directories (%s)", dir, strings.Join(r.roots, ", "))
}

// text is the result as the model reads it.
func (res runResult) text(timeout time.Duration) string {
	var b strings.Builder
	switch {
	case res.TimedOut:
		fmt.Fprintf(&b, "stopped after %s, the time limit\n", timeout)
	case res.Status == -1 && res.Truncated:
		// Its output had nowhere to go, as when a pipe's reader quits.
		b.WriteString("stopped when its output reached the limit\n")
	default:
		fmt.Fprintf(&b, "exit status %d\n", res.Status)
	}
	if res.Stdout != "" {
		b.WriteString("--- stdout ---\n")
		b.WriteString(res.Stdout)
		if !strings.HasSuffix(res.Stdout, "\n") {
			b.WriteByte('\n')
		}
	}
	if res.Stderr != "" {
		b.WriteString("--- stderr ---\n")
		b.WriteString(res.Stderr)
		if !strings.HasSuffix(res.Stderr, "\n") {
			b.WriteByte('\n')
		}
	}
	if res.Truncated {
		b.WriteString("(output cut off at the limit)\n")
	}
	return b.String()
}

// cappedBuffer keeps the first limit bytes written to it and then refuses
// more, which stops a tool writing endlessly as a closed pipe would. The
// buffer is a field, not embedded: an embedded bytes.Buffer would bring its
// ReadFrom, which io.Copy prefers to Write, and the limit would never apply.
type cappedBuffer struct {
	buf   bytes.Buffer
	limit int
	cut   bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	room := b.limit - b.buf.Len()
	if len(p) <= room {
		return b.buf.Write(p)
	}
	if room > 0 {
		b.buf.Write(p[:room])
	}
	b.cut = true
	return room, errors.New("output limit reached")
}

func (b *cappedBuffer) String() string { return b.buf.String() }
