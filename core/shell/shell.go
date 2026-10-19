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
//	! a                       invert a's exit status
//	cmd < in > out >> log     redirect standard input and output
//	cmd 2> errs 2>> errlog    redirect standard error
//	cmd > /dev/null           discard: /dev/null works on every platform
//	NAME=value                set a shell variable
//	$NAME  ${NAME}  $?        expand a variable / the last exit status
//	${NAME:-default}          a default for an unset or empty variable
//	$1 ... $9  ${10}  $#      the script's arguments and their count
//	"$@"  $*                  all of them: one word each / joined by spaces
//	$PWD                      the working directory
//	$(command)                the output of a command
//	'literal'  "with $vars"   quoting; \ escapes a special character
//	*.go  file?.txt  [a-c]*   globbing
//	~  ~/dir                  the home directory
//	# comment
//
//	if a; then b; elif c; then d; else e; fi
//	for NAME in word...; do a; done
//	while a; do b; done
//	case word in pattern | pattern) a ;; *) b ;; esac
//
// The builtins are cd, exit, help, set (-e, -u, -- ARG...), shift, break,
// continue and [ (the test tool). There are no functions, no arithmetic and
// no external programs: to run one, go through the env tool
// (envx program arg...).
//
// Two things differ from sh on purpose. A variable's value is never re-split
// or re-globbed, so a path with spaces needs no quoting; an unquoted $(...)
// is split into lines, never into words, for the same reason. And
// NAME=value before a command sets the variable for the rest of the script.
package shell

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the program name diagnostics are prefixed with.
const Name = "gnux"

// devNull is the one device name the shell understands, on every platform.
const devNull = "/dev/null"

// Prompt is written before each line when the shell is interactive, and
// ContinuePrompt before a line that continues an unfinished command (the body
// of an if, a loop or a case).
const (
	Prompt         = "gnux> "
	ContinuePrompt = "> "
)

// ExecFunc runs the tool called name with args against the given streams and
// returns its exit status. found is false when no such tool exists.
type ExecFunc func(name string, args []string, s core.Streams) (status int, found bool)

// Options controls how a script is run.
type Options struct {
	// ErrExit (-e) stops the script at the first command that fails, as
	// `set -e` does in sh. Commands whose status is tested - by && or ||, by
	// !, or as the condition of an if or a while - are exempt.
	ErrExit bool

	// NoUnset (-u) makes expanding a variable that is not set an error that
	// stops the script, as `set -u` does in sh. ${NAME:-default} is how a
	// script reads one that may be missing.
	NoUnset bool

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

	// Glob lists the files an unquoted pattern matches; it defaults to
	// fsys.Glob. It is the injection point for a host that supplies its
	// own filesystem, and for tests that must not walk the real one.
	Glob func(pattern string) ([]string, error)

	// LoopLimit, when positive, is the most times one loop may run its body
	// before the shell stops it with an error. It is for a host that cannot
	// interrupt a script, such as a browser tab; zero means no limit.
	LoopLimit int

	streams core.Streams
	vars    map[string]string
	name    string   // $0
	args    []string // $1, $2, ...
	status  int
	exited  bool

	// aborted ends the command being run after an error that makes going on
	// meaningless (an unset variable under -u).
	aborted bool
	// breaking and continuing count the loops a break or continue still has
	// to leave; loops is how many enclose the command being run.
	breaking, continuing, loops int
	// testing is positive while a condition runs, whose failure -e ignores.
	testing int
	// subshell is positive while the stages of a pipeline run. They share
	// the process, so what a subshell could not do to its parent (cd) is
	// refused there.
	subshell int
}

// New builds a shell that runs tools through exec and reads and writes s.
func New(opts Options, exec ExecFunc, s core.Streams) *Shell {
	return &Shell{
		Options: opts,
		Exec:    exec,
		Getenv:  os.Getenv,
		HomeDir: os.UserHomeDir,
		Glob:    fsys.Glob,
		streams: s,
		vars:    map[string]string{},
		name:    Name,
	}
}

// Set defines a shell variable, visible to the script as $name.
func (sh *Shell) Set(name, value string) {
	sh.vars[name] = value
}

// SetArgs gives the script its name ($0) and its arguments ($1, $2, ... and
// "$@").
func (sh *Shell) SetArgs(name string, args []string) {
	sh.name, sh.args = name, slices.Clone(args)
}

// errorf writes one of the shell's own diagnostics to the standard error of
// the command it is about.
func (sh *Shell) errorf(s core.Streams, format string, a ...any) {
	core.NewDiag(Name, s).Errorf(format, a...)
}

// Run reads a script from r and executes it command by command: a command is
// one line, or as many as an if, a loop, a case or a $(...) spans. It returns
// nil when the last command succeeded and a *core.ExitError carrying its
// status otherwise.
//
// cd changes the working directory of the whole process, since that is what
// the tools resolve relative paths against; Run restores the original
// directory before returning.
func (sh *Shell) Run(r io.Reader) error {
	if wd, err := fsys.Getwd(); err == nil {
		defer func() { _ = fsys.Chdir(wd) }()
	}

	reader := core.NewReader(r)
	pending := ""
	for !sh.exited {
		if sh.Interactive {
			prompt := Prompt
			if pending != "" {
				prompt = ContinuePrompt
			}
			fmt.Fprint(sh.streams.Err, prompt)
		}
		line, err := reader.ReadString('\n')
		if line != "" {
			pending += strings.TrimRight(line, "\r\n") + "\n"
		}
		if pending != "" {
			steps, perr := parse(pending)
			// An unfinished command waits for its next line, unless there
			// are no more.
			if err != nil || !incomplete(perr) {
				pending = ""
				sh.runParsed(steps, perr)
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				sh.errorf(sh.streams, "read error: %s", core.ErrText(err))
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

// Eval executes one command - a line, or a whole if, loop or case written
// across several - and returns its exit status. Unlike Run it keeps
// everything for the next call - variables, $? and the working directory - so
// a host driving the shell one line at a time (a REPL, the web playground)
// gets a continuous session. An exit in the line ends that line only.
func (sh *Shell) Eval(line string) int {
	sh.exited = false
	steps, err := parse(strings.ReplaceAll(strings.TrimRight(line, "\r\n"), "\r\n", "\n"))
	sh.runParsed(steps, err)
	return sh.status
}

// Incomplete reports whether src is the start of a command that more lines
// would finish: an if without its fi, a loop without its done, an unclosed
// $(. A host reading one line at a time uses it to decide whether to ask for
// another before calling Eval.
func Incomplete(src string) bool {
	_, err := parse(src)
	return incomplete(err)
}

// runParsed executes what parse returned, or reports its syntax error.
func (sh *Shell) runParsed(steps []step, err error) {
	if err != nil {
		sh.errorf(sh.streams, "syntax error: %v", err)
		sh.status = 2
		// A script with a syntax error cannot be trusted to mean what its
		// author intended, so only an interactive session carries on.
		if !sh.Interactive {
			sh.exited = true
		}
		return
	}
	sh.aborted, sh.breaking, sh.continuing = false, 0, 0
	sh.runList(steps, sh.streams)
}

// stopped reports whether the commands being run must be abandoned: the
// script exited, an error aborted it, or a break or continue is on its way
// out to its loop.
func (sh *Shell) stopped() bool {
	return sh.exited || sh.aborted || sh.breaking > 0 || sh.continuing > 0
}

// runList executes steps in order against s, leaving the status of the last
// one run in sh.status.
func (sh *Shell) runList(steps []step, s core.Streams) {
	for i, st := range steps {
		if sh.stopped() {
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

		sh.status = sh.runPipeline(st.pipeline, s)

		// A status that something tests is a condition, not a failure.
		tested := sh.testing > 0 || st.pipeline.negate ||
			(i+1 < len(steps) && (steps[i+1].connector == "&&" || steps[i+1].connector == "||"))
		if sh.ErrExit && sh.status != 0 && !tested {
			sh.exited = true
		}
	}
}

// state is what a subshell must not change for its parent.
type state struct {
	vars             map[string]string
	args             []string
	errExit, noUnset bool
	exited, aborted  bool
	loops, testing   int
	status           int
}

func (sh *Shell) save() state {
	return state{
		vars: maps.Clone(sh.vars), args: sh.args,
		errExit: sh.ErrExit, noUnset: sh.NoUnset,
		exited: sh.exited, aborted: sh.aborted,
		loops: sh.loops, testing: sh.testing, status: sh.status,
	}
}

func (sh *Shell) restore(st state) {
	sh.vars, sh.args = st.vars, st.args
	sh.ErrExit, sh.NoUnset = st.errExit, st.noUnset
	sh.exited, sh.aborted = st.exited, st.aborted
	sh.loops, sh.testing, sh.status = st.loops, st.testing, st.status
	sh.breaking, sh.continuing = 0, 0
}

// runPipeline runs the commands of a pipeline concurrently, each one's output
// feeding the next one's input, and returns the status of the last (inverted
// when the pipeline starts with !).
func (sh *Shell) runPipeline(pl pipeline, s core.Streams) int {
	status := sh.runStages(pl.cmds, s)
	if !pl.negate {
		return status
	}
	if status == 0 {
		return 1
	}
	return 0
}

func (sh *Shell) runStages(cmds []command, base core.Streams) int {
	if len(cmds) == 1 {
		p := sh.prepare(cmds[0], base)
		s, closeFiles, ok := sh.open(p, base)
		defer closeFiles()
		return sh.runCommand(cmds[0], p, s, ok, false)
	}

	// Expansion and assignments touch shell state, so they happen here, in
	// order, before any stage starts running.
	prepped := make([]prepared, len(cmds))
	compound := false
	for i, cmd := range cmds {
		prepped[i] = sh.prepare(cmd, base)
		compound = compound || cmd.compound != nil
	}
	for _, p := range prepped {
		if p.failed {
			sh.abort()
			return 1
		}
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
	stages := make([]stage, len(cmds))
	in := base.In
	var upstream *io.PipeReader
	for i, p := range prepped {
		st := stage{prepared: p, upstream: upstream}
		s := core.Streams{In: in, Out: base.Out, Err: base.Err}
		if i < len(cmds)-1 {
			var next *io.PipeReader
			next, st.out = io.Pipe()
			s.Out = pipeWriter{st.out}
			in, upstream = next, next
		}
		st.streams, st.closeFiles, st.redirected = sh.open(p, s)
		stages[i] = st
	}

	// The one compound command a pipeline may hold runs its body on its
	// stage's goroutine while this one waits, so the shell's state still has
	// a single user at a time. It is a subshell: what it sets is put back.
	var saved state
	if compound {
		saved = sh.save()
	}
	sh.subshell++

	statuses := make([]int, len(cmds))
	var wg sync.WaitGroup
	// A stage that panics still closes its pipes, or its neighbors would
	// wait on it for ever; the panic is raised again once all have ended.
	var trap core.Trap
	for i, st := range stages {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer st.closeFiles()
			// Closing our end tells the reader "end of input" and the writer
			// upstream "nobody is listening", which is what ends `yes | head`.
			defer func() {
				if st.out != nil {
					_ = st.out.Close()
				}
				if st.upstream != nil {
					_ = st.upstream.Close()
				}
			}()
			defer trap.Catch()
			statuses[i] = sh.runCommand(cmds[i], st.prepared, st.streams, st.redirected, true)
		}()
	}

	wg.Wait()
	sh.subshell--
	if compound {
		sh.restore(saved)
	}
	trap.Release()
	return statuses[len(cmds)-1]
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
	failed    bool // an expansion was refused, and reported

	// last is $? as the command starts, which is what a bare exit returns.
	last int
	// substituted and status carry the result of the command's last $(...),
	// the exit status of a command that is only assignments.
	substituted bool
	status      int
}

// target is a redirection with its file name resolved.
type target struct {
	op, path string
}

// prepare expands a command's words and redirections and applies any leading
// NAME=value assignments. As in sh, the command's own words are expanded
// before its assignments take effect (`X=1 echox $X` sees the old X), while
// each assignment sees the ones before it (`A=1 B=$A`).
func (sh *Shell) prepare(cmd command, s core.Streams) prepared {
	p := prepared{last: sh.status}
	ex := &expander{sh: sh, s: s}

	type assign struct {
		name  string
		value word
	}
	var assigns []assign
	words := cmd.words
	for len(words) > 0 {
		name, value, ok := assignment(words[0])
		if !ok {
			break
		}
		assigns = append(assigns, assign{name, value})
		words = words[1:]
	}
	for _, w := range words {
		p.args = append(p.args, ex.fields(w)...)
	}
	for _, a := range assigns {
		if v := ex.single(a.value); !ex.failed {
			sh.vars[a.name] = v
		}
	}
	for _, r := range cmd.redirects {
		paths := ex.fields(r.target)
		if len(paths) != 1 {
			p.ambiguous = true
			continue
		}
		p.redirects = append(p.redirects, target{op: r.op, path: paths[0]})
	}

	p.failed, p.substituted, p.status = ex.failed, ex.substituted, ex.status
	return p
}

// open applies a prepared command's redirections to s. ok is false when one
// of them failed (and was reported); the returned function closes the files.
func (sh *Shell) open(p prepared, s core.Streams) (core.Streams, func(), bool) {
	if p.failed {
		return s, func() {}, false
	}
	if p.ambiguous {
		sh.errorf(s, "ambiguous redirect")
		return s, func() {}, false
	}
	return sh.redirect(p.redirects, s)
}

// abort ends the command being run, and with it a script: one that read a
// variable it never set cannot be trusted to mean what it says. An
// interactive session carries on with the next line.
func (sh *Shell) abort() {
	sh.aborted = true
	if !sh.Interactive {
		sh.exited = true
	}
}

// runCommand runs a prepared command against its (already redirected) streams
// and returns its exit status. redirected is false when a redirection failed,
// in which case the command is not run. piped is true for a stage of a
// multi-command pipeline, which runs as a subshell would: cd, exit and the
// other builtins that change the shell do not affect the shell itself.
func (sh *Shell) runCommand(cmd command, p prepared, s core.Streams, redirected, piped bool) int {
	if p.failed {
		sh.abort()
		return 1
	}
	if !redirected {
		return 1
	}
	if cmd.compound != nil {
		return sh.runCompound(cmd.compound, s)
	}
	if len(p.args) == 0 {
		if p.substituted {
			return p.status
		}
		return 0
	}

	name, args := p.args[0], p.args[1:]
	switch name {
	case "cd":
		return sh.cd(args, s, piped || sh.subshell > 0)
	case "exit":
		return sh.exit(args, s, p.last, piped)
	case "set", "shift", "break", "continue":
		if piped {
			return 0
		}
		return sh.builtin(name, args, s)
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
	case "[":
		// [ EXPRESSION ] is the test tool, written the way scripts write it.
		if len(args) == 0 || args[len(args)-1] != "]" {
			sh.errorf(s, "[: missing ']'")
			return 2
		}
		name, args = "test", args[:len(args)-1]
	}

	if sh.Exec != nil {
		if status, found := sh.Exec(name, args, s); found {
			return status
		}
	}
	sh.errorf(s, "%s: command not found", name)
	return 127
}

// runCompound runs an if, for, while or case against s and returns its exit
// status: that of the last command it ran, or 0 when it ran none.
func (sh *Shell) runCompound(c compound, s core.Streams) int {
	switch c := c.(type) {
	case *ifClause:
		for _, b := range c.branches {
			ok := sh.test(b.cond, s)
			if sh.stopped() {
				return sh.status
			}
			if ok {
				return sh.runBody(b.body, s)
			}
		}
		return sh.runBody(c.elseBody, s)

	case *forClause:
		items := sh.args
		if !c.overArgs {
			ex := &expander{sh: sh, s: s}
			items = nil
			for _, w := range c.items {
				items = append(items, ex.fields(w)...)
			}
			if ex.failed {
				sh.abort()
				return 1
			}
		}
		i := 0
		return sh.loop(s, c.body, func() bool {
			if i >= len(items) {
				return false
			}
			sh.vars[c.name] = items[i]
			i++
			return true
		})

	case *whileClause:
		return sh.loop(s, c.body, func() bool { return sh.test(c.cond, s) })

	case *caseClause:
		ex := &expander{sh: sh, s: s}
		subject := ex.single(c.subject)
		for _, arm := range c.arms {
			for _, w := range arm.patterns {
				pattern := ex.pattern(w)
				if ex.failed {
					break
				}
				if match(pattern, subject) {
					return sh.runBody(arm.body, s)
				}
			}
		}
		if ex.failed {
			sh.abort()
			return 1
		}
	}
	return 0
}

// test runs the commands of a condition and reports whether the last one
// succeeded. A failure there is an answer, so -e does not act on it.
func (sh *Shell) test(cond []step, s core.Streams) bool {
	sh.testing++
	sh.runList(cond, s)
	sh.testing--
	return sh.status == 0
}

// runBody runs the commands a compound command guards. A body that is empty
// succeeds.
func (sh *Shell) runBody(body []step, s core.Streams) int {
	if len(body) == 0 {
		return 0
	}
	sh.runList(body, s)
	return sh.status
}

// loop runs body for as long as more reports another turn, honoring break and
// continue, and returns the status of the last turn (0 when there was none).
func (sh *Shell) loop(s core.Streams, body []step, more func() bool) int {
	sh.loops++
	defer func() { sh.loops-- }()

	status := 0
	for turns := 1; more(); turns++ {
		if sh.stopped() {
			// The condition itself exited or aborted.
			return sh.status
		}
		if sh.LoopLimit > 0 && turns > sh.LoopLimit {
			sh.errorf(s, "loop stopped after %d iterations", sh.LoopLimit)
			sh.abort()
			return 1
		}
		status = sh.runBody(body, s)

		if sh.continuing > 0 {
			sh.continuing--
			if sh.continuing > 0 {
				return status // an outer loop is the one to continue
			}
			continue
		}
		if sh.breaking > 0 {
			sh.breaking--
			return status
		}
		if sh.stopped() {
			return status
		}
	}
	if sh.stopped() {
		return sh.status
	}
	return status
}

// builtin runs set, shift, break or continue: the builtins that change the
// shell itself.
func (sh *Shell) builtin(name string, args []string, s core.Streams) int {
	switch name {
	case "set":
		if len(args) == 0 {
			sh.errorf(s, "set: expected -e, -u, +e, +u or -- ARG...")
			return 2
		}
		for i, a := range args {
			if a == "--" {
				sh.args = slices.Clone(args[i+1:])
				return 0
			}
			if len(a) < 2 || (a[0] != '-' && a[0] != '+') {
				sh.errorf(s, "set: %s: invalid option", a)
				return 2
			}
			on := a[0] == '-'
			for _, c := range a[1:] {
				switch c {
				case 'e':
					sh.ErrExit = on
				case 'u':
					sh.NoUnset = on
				default:
					sh.errorf(s, "set: %c%c: invalid option", a[0], c)
					return 2
				}
			}
		}
		return 0

	case "shift":
		n, ok := count(args)
		if !ok {
			sh.errorf(s, "shift: %s: numeric argument required", args[0])
			return 2
		}
		if n > len(sh.args) {
			sh.errorf(s, "shift: %d: shift count out of range", n)
			return 1
		}
		sh.args = sh.args[n:]
		return 0
	}

	// break and continue
	n, ok := count(args)
	if !ok || n == 0 {
		sh.errorf(s, "%s: %s: loop count out of range", name, args[0])
		return 2
	}
	if sh.loops == 0 {
		sh.errorf(s, "%s: only meaningful in a for or while loop", name)
		return 1
	}
	n = min(n, sh.loops)
	if name == "break" {
		sh.breaking = n
	} else {
		sh.continuing = n
	}
	return 0
}

// count reads the optional count shift, break and continue take; it is 1
// when absent.
func count(args []string) (int, bool) {
	if len(args) == 0 {
		return 1, true
	}
	n, err := strconv.Atoi(args[0])
	return n, err == nil && n >= 0
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
			f, err = fsys.Open(r.path)
		case ">", "2>":
			f, err = fsys.OpenFile(r.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		default: // ">>", "2>>"
			f, err = fsys.OpenFile(r.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		}
		if err != nil {
			sh.errorf(s, "%s: %s", r.path, core.ErrText(err))
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
func (sh *Shell) cd(args []string, s core.Streams, piped bool) int {
	if len(args) > 1 {
		sh.errorf(s, "cd: too many arguments")
		return 1
	}

	dir := ""
	if len(args) == 1 {
		dir = args[0]
	} else {
		home, err := sh.HomeDir()
		if err != nil || home == "" {
			sh.errorf(s, "cd: HOME not set")
			return 1
		}
		dir = home
	}

	if piped {
		if _, err := fsys.Stat(dir); err != nil {
			sh.errorf(s, "cd: %s: %s", dir, core.ErrText(err))
			return 1
		}
		return 0
	}
	if err := fsys.Chdir(dir); err != nil {
		sh.errorf(s, "cd: %s: %s", dir, core.ErrText(err))
		return 1
	}
	return 0
}

// exit ends the script with the given status, or the last command's.
func (sh *Shell) exit(args []string, s core.Streams, last int, piped bool) int {
	status := last
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			sh.errorf(s, "exit: %s: numeric argument required", args[0])
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
  ! a                       invert a's exit status
  cmd < in > out >> log     redirect standard input and output
  cmd 2> errs               redirect standard error; /dev/null discards
  NAME=value                set a variable;  $NAME  ${NAME}  $?  $PWD  expand one
  ${NAME:-default}          a default for an unset or empty variable
  $1 $2 ...  $#  "$@"       the script's arguments, their count, all of them
  $(command)                the output of a command; unquoted, one word per line
  'literal'  "with $vars"   quoting; \ escapes a special character
  *.go  file?.txt  [a-c]*   globbing;  ~  is the home directory
  # comment

  if a; then b; elif c; then d; else e; fi
  for NAME in word...; do a; done
  while a; do b; done
  case word in pattern | pattern) a ;; *) b ;; esac

Builtins: cd [DIR], exit [STATUS], help [TOOL [OPTION]].
          set -e | -u | -- ARG..., shift [N], break [N], continue [N], [ TEST ].
Run 'TOOL --help' for a tool's options, 'help TOOL' for its full manual and
'help TOOL OPTION' for one option. To run an external program: envx PROGRAM ARG...
`
