// Package shell implements the gnux shell: a small, portable command
// interpreter whose commands are the gnux tools, run in-process. A script
// written for it behaves the same on Linux, macOS and Windows, because nothing
// in it depends on the system's own shell or utilities.
//
// The language is a deliberately small subset of POSIX sh:
//
//	command arg...            run a tool (by its gnux or GNU name) or a builtin
//	a | b                     pipe a's standard output into b
//	a && b    a || b    a; b  run b on success / on failure / always
//	cmd < in > out >> log     redirect standard input and output
//	cmd 2> errs 2>> errlog    redirect standard error
//	cmd > /dev/null           discard: /dev/null works on every platform
//	NAME=value                set a shell variable
//	$NAME  ${NAME}  $?        expand a variable / the last exit status
//	$PWD                      the working directory
//	'literal'  "with $vars"   quoting; \ escapes a special character
//	*.go  file?.txt  [a-c]*   globbing
//	~  ~/dir                  the home directory
//	# comment
//
// The builtins are cd, exit and help. There are no functions, loops or
// conditionals beyond && and ||, and no external programs: to run one, go
// through the env tool (envx program arg...).
package shell

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/braswelljr/gnux/core"
)

// Name is the program name diagnostics are prefixed with.
const Name = "gnux"

// devNull is the one device name the shell understands, on every platform.
const devNull = "/dev/null"

// Prompt is written before each line when the shell is interactive.
const Prompt = "gnux> "

// ExecFunc runs the tool called name with args against the given streams and
// returns its exit status. found is false when no such tool exists.
type ExecFunc func(name string, args []string, s core.Streams) (status int, found bool)

// Options controls how a script is run.
type Options struct {
	// ErrExit (-e) stops the script at the first command that fails, as
	// `set -e` does in sh. Commands whose status is tested by && or || are
	// exempt.
	ErrExit bool

	// Interactive writes a prompt before each line and keeps going after a
	// syntax error. The command line turns it on when reading from a terminal.
	Interactive bool
}

// Shell is the interpreter. Construct it with New.
type Shell struct {
	Options

	// Exec runs a tool. It is the only way the shell reaches the tools, so a
	// caller can restrict, wrap or replace the command set.
	Exec ExecFunc

	// Getenv and HomeDir are the injection points for the environment; they
	// default to os.Getenv and os.UserHomeDir.
	Getenv  func(string) string
	HomeDir func() (string, error)

	streams core.Streams
	diag    *core.Diag
	vars    map[string]string
	status  int
	exited  bool
}

// New builds a shell that runs tools through exec and reads and writes s.
func New(opts Options, exec ExecFunc, s core.Streams) *Shell {
	return &Shell{
		Options: opts,
		Exec:    exec,
		Getenv:  os.Getenv,
		HomeDir: os.UserHomeDir,
		streams: s,
		diag:    core.NewDiag(Name, s),
		vars:    map[string]string{},
	}
}

// Set defines a shell variable, visible to the script as $name.
func (sh *Shell) Set(name, value string) {
	sh.vars[name] = value
}

// Run reads a script from r, one command line at a time, and executes it. It
// returns nil when the last command succeeded and a *core.ExitError carrying
// its status otherwise.
//
// cd changes the working directory of the whole process, since that is what
// the tools resolve relative paths against; Run restores the original
// directory before returning.
func (sh *Shell) Run(r io.Reader) error {
	if wd, err := os.Getwd(); err == nil {
		defer func() { _ = os.Chdir(wd) }()
	}

	reader := bufio.NewReader(r)
	for !sh.exited {
		if sh.Interactive {
			fmt.Fprint(sh.streams.Err, Prompt)
		}
		line, err := reader.ReadString('\n')
		if line != "" {
			sh.runLine(strings.TrimRight(line, "\r\n"))
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				sh.diag.Errorf("read error: %s", core.ErrText(err))
				sh.status = 1
			}
			break
		}
	}
	if sh.Interactive && !sh.exited {
		// Leave the terminal on a fresh line after end-of-input (Ctrl-D).
		fmt.Fprintln(sh.streams.Err)
	}

	if sh.status != 0 {
		return &core.ExitError{Code: sh.status}
	}
	return nil
}

// RunString executes script, a convenience over Run.
func (sh *Shell) RunString(script string) error {
	return sh.Run(strings.NewReader(script))
}

// Eval executes a single line and returns its exit status. Unlike Run it keeps
// everything for the next call - variables, $? and the working directory - so
// a host driving the shell one line at a time (a REPL, the web playground)
// gets a continuous session. An exit in the line ends that line only.
func (sh *Shell) Eval(line string) int {
	sh.exited = false
	sh.runLine(strings.TrimRight(line, "\r\n"))
	return sh.status
}

// runLine parses and executes one line.
func (sh *Shell) runLine(line string) {
	steps, err := parse(line)
	if err != nil {
		sh.diag.Errorf("syntax error: %v", err)
		sh.status = 2
		// A script with a syntax error cannot be trusted to mean what its
		// author intended, so only an interactive session carries on.
		if !sh.Interactive {
			sh.exited = true
		}
		return
	}

	for i, st := range steps {
		if sh.exited {
			return
		}
		switch st.connector {
		case "&&":
			if sh.status != 0 {
				continue
			}
		case "||":
			if sh.status == 0 {
				continue
			}
		}

		sh.status = sh.runPipeline(st.pipeline)

		// A status that the next step tests is a condition, not a failure.
		tested := i+1 < len(steps) && (steps[i+1].connector == "&&" || steps[i+1].connector == "||")
		if sh.ErrExit && sh.status != 0 && !tested {
			sh.exited = true
		}
	}
}

// runPipeline runs the commands of a pipeline concurrently, each one's output
// feeding the next one's input, and returns the status of the last.
func (sh *Shell) runPipeline(pl pipeline) int {
	if len(pl) == 1 {
		p := sh.prepare(pl[0])
		s, closeFiles, ok := sh.open(p, sh.streams)
		defer closeFiles()
		return sh.runCommand(p, s, ok, false)
	}

	// Expansion and assignments touch shell state, so they happen here, in
	// order, before any stage starts running.
	cmds := make([]prepared, len(pl))
	for i, cmd := range pl {
		cmds[i] = sh.prepare(cmd)
	}

	// Every stage's pipes and redirections are set up here, before any stage
	// runs, so that a file one stage creates already exists when an earlier
	// stage looks - the order a POSIX shell guarantees (`ls | wc -l > count`
	// counts count).
	type stage struct {
		prepared
		streams    core.Streams
		closeFiles func()
		redirected bool
		out        *io.PipeWriter // our end of the pipe to the next stage
		upstream   *io.PipeReader // the pipe we read from
	}
	stages := make([]stage, len(pl))
	in := sh.streams.In
	var upstream *io.PipeReader
	for i, p := range cmds {
		st := stage{prepared: p, upstream: upstream}
		s := core.Streams{In: in, Out: sh.streams.Out, Err: sh.streams.Err}
		if i < len(pl)-1 {
			var next *io.PipeReader
			next, st.out = io.Pipe()
			s.Out = pipeWriter{st.out}
			in, upstream = next, next
		}
		st.streams, st.closeFiles, st.redirected = sh.open(p, s)
		stages[i] = st
	}

	statuses := make([]int, len(pl))
	var wg sync.WaitGroup
	for i, st := range stages {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer st.closeFiles()
			statuses[i] = sh.runCommand(st.prepared, st.streams, st.redirected, true)
			// Closing our end tells the reader "end of input" and the writer
			// upstream "nobody is listening", which is what ends `yes | head`.
			if st.out != nil {
				_ = st.out.Close()
			}
			if st.upstream != nil {
				_ = st.upstream.Close()
			}
		}()
	}

	wg.Wait()
	return statuses[len(pl)-1]
}

// pipeWriter reports a write to a pipe whose reader has gone away as EPIPE,
// the error a tool sees from a real pipe and already knows to treat as a quiet
// reason to stop.
type pipeWriter struct {
	w *io.PipeWriter
}

func (p pipeWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	if errors.Is(err, io.ErrClosedPipe) {
		err = syscall.EPIPE
	}
	return n, err
}

// prepared is a command after expansion, ready to run.
type prepared struct {
	args      []string
	redirects []target
	ambiguous bool // a redirection target expanded to several words
}

// target is a redirection with its file name resolved.
type target struct {
	op, path string
}

// prepare expands a command's words and redirections and applies any leading
// NAME=value assignments.
func (sh *Shell) prepare(cmd command) prepared {
	var p prepared
	for _, w := range cmd.words {
		p.args = append(p.args, sh.expand(w)...)
	}
	for len(p.args) > 0 {
		name, value, ok := assignment(p.args[0])
		if !ok {
			break
		}
		sh.vars[name] = value
		p.args = p.args[1:]
	}
	for _, r := range cmd.redirects {
		paths := sh.expand(r.target)
		if len(paths) != 1 {
			p.ambiguous = true
			continue
		}
		p.redirects = append(p.redirects, target{op: r.op, path: paths[0]})
	}
	return p
}

// open applies a prepared command's redirections to s. ok is false when one
// of them failed (and was reported); the returned function closes the files.
func (sh *Shell) open(p prepared, s core.Streams) (core.Streams, func(), bool) {
	if p.ambiguous {
		sh.diag.Errorf("ambiguous redirect")
		return s, func() {}, false
	}
	return sh.redirect(p.redirects, s)
}

// runCommand runs a prepared command against its (already redirected) streams
// and returns its exit status. redirected is false when a redirection failed,
// in which case the command is not run. piped is true for a stage of a
// multi-command pipeline, which runs as a subshell would: cd and exit there do
// not affect the shell itself.
func (sh *Shell) runCommand(p prepared, s core.Streams, redirected, piped bool) int {
	if !redirected {
		return 1
	}
	if len(p.args) == 0 {
		return 0
	}

	name, args := p.args[0], p.args[1:]
	switch name {
	case "cd":
		return sh.cd(args, piped)
	case "exit":
		return sh.exit(args, piped)
	case "help":
		// help TOOL [OPTION] is the tool's manual, which the host provides
		// as `gnux help`; the shell itself knows only its own summary.
		if len(args) > 0 && sh.Exec != nil {
			if status, found := sh.Exec("gnux", append([]string{"help"}, args...), s); found {
				return status
			}
		}
		fmt.Fprint(s.Out, helpText)
		return 0
	}

	if sh.Exec != nil {
		if status, found := sh.Exec(name, args, s); found {
			return status
		}
	}
	sh.diag.Errorf("%s: command not found", name)
	return 127
}

// redirect opens the files named by redirs and returns streams pointing at
// them, plus a function that closes them.
func (sh *Shell) redirect(redirs []target, s core.Streams) (core.Streams, func(), bool) {
	var files []*os.File
	closeFiles := func() {
		for _, f := range files {
			_ = f.Close()
		}
	}

	for _, r := range redirs {
		// /dev/null is handled here rather than opened, so that scripts can
		// use it on Windows and in the browser, where no such file exists.
		if r.path == devNull {
			switch r.op {
			case "<":
				s.In = strings.NewReader("")
			case ">", ">>":
				s.Out = io.Discard
			default:
				s.Err = io.Discard
			}
			continue
		}

		var f *os.File
		var err error
		switch r.op {
		case "<":
			f, err = os.Open(r.path)
		case ">", "2>":
			f, err = os.OpenFile(r.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		default: // ">>", "2>>"
			f, err = os.OpenFile(r.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		}
		if err != nil {
			sh.diag.Errorf("%s: %s", r.path, core.ErrText(err))
			return s, closeFiles, false
		}
		files = append(files, f)
		switch r.op {
		case "<":
			s.In = f
		case ">", ">>":
			s.Out = f
		default:
			s.Err = f
		}
	}
	return s, closeFiles, true
}

// cd changes the working directory; with no argument it goes home.
func (sh *Shell) cd(args []string, piped bool) int {
	if len(args) > 1 {
		sh.diag.Errorf("cd: too many arguments")
		return 1
	}

	dir := ""
	if len(args) == 1 {
		dir = args[0]
	} else {
		home, err := sh.HomeDir()
		if err != nil || home == "" {
			sh.diag.Errorf("cd: HOME not set")
			return 1
		}
		dir = home
	}

	if piped {
		if _, err := os.Stat(dir); err != nil {
			sh.diag.Errorf("cd: %s: %s", dir, core.ErrText(err))
			return 1
		}
		return 0
	}
	if err := os.Chdir(dir); err != nil {
		sh.diag.Errorf("cd: %s: %s", dir, core.ErrText(err))
		return 1
	}
	return 0
}

// exit ends the script with the given status, or the last command's.
func (sh *Shell) exit(args []string, piped bool) int {
	status := sh.status
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			sh.diag.Errorf("exit: %s: numeric argument required", args[0])
			n = 2
		}
		status = n & 0xff
	}
	if !piped {
		sh.exited = true
	}
	return status
}

// helpText is what the help builtin prints.
const helpText = `gnux shell - commands are the gnux tools, run in-process.

  command arg...            run a tool (rmx or rm, lsx or ls, ...) or a builtin
  a | b                     pipe a's output into b
  a && b    a || b    a; b  run b on success / on failure / always
  cmd < in > out >> log     redirect standard input and output
  cmd 2> errs               redirect standard error; /dev/null discards
  NAME=value                set a variable;  $NAME  ${NAME}  $?  $PWD  expand one
  'literal'  "with $vars"   quoting; \ escapes a special character
  *.go  file?.txt  [a-c]*   globbing;  ~  is the home directory
  # comment

Builtins: cd [DIR], exit [STATUS], help [TOOL [OPTION]].
Run 'TOOL --help' for a tool's options, 'help TOOL' for its full manual and
'help TOOL OPTION' for one option. To run an external program: envx PROGRAM ARG...
`
