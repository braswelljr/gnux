package find

import (
	"slices"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/run"
)

// batchLimit is how many bytes of file names one "-exec ... {} +" command
// line takes before it is run: comfortably below every platform's limit on
// the length of a command line.
const batchLimit = 128 << 10

// execAction is one -exec, -execdir, -ok or -okdir.
type execAction struct {
	name string   // the predicate, for messages
	argv []string // the command; for the + form, without the final {}
	plus bool     // {} + : collect the files and run the command per batch
	dir  bool     // -execdir, -okdir: run in the file's directory
	ok   bool     // -ok, -okdir: ask first

	batch     []string // the {} of each file collected so far
	batchTool []string // the same files as an in-process tool is given them
	batchDir  string
	batchSize int
}

// exec parses the command of -exec and its relatives, up to the ';' or the
// "{} +" that ends it.
func (p *parser) exec(name string) (pred, error) {
	f := p.f
	x := &execAction{name: name, dir: strings.HasSuffix(name, "dir"), ok: strings.HasPrefix(name, "-ok")}

	end := -1
	for i := p.pos; i < len(p.args); i++ {
		if p.args[i] == ";" {
			end = i
			break
		}
		// Only -exec and -execdir have the + form, and only right after {}.
		if p.args[i] == "+" && !x.ok && i > p.pos && p.args[i-1] == "{}" {
			end, x.plus = i, true
			break
		}
	}
	if end < 0 {
		return nil, core.Usagef("missing argument to `%s'", name)
	}
	x.argv = p.args[p.pos:end]
	p.pos = end + 1
	if len(x.argv) == 0 {
		return nil, core.Usagef("invalid argument `;' to `%s'", name)
	}

	if !x.plus {
		return func(e *entry) bool { return f.execOne(x, e) }, nil
	}

	x.argv = x.argv[:len(x.argv)-1]
	if len(x.argv) == 0 {
		return nil, core.Usagef("invalid argument `{}' to `%s'", name)
	}
	for _, arg := range x.argv {
		if strings.Contains(arg, "{}") {
			return nil, core.Usagef("Only one instance of {} is supported with %s ... +", name)
		}
	}
	for _, arg := range x.argv {
		x.batchSize += len(arg) + 1
	}
	f.execs = append(f.execs, x)
	return func(e *entry) bool {
		f.addBatch(x, e)
		return true
	}, nil
}

// dirArg returns the directory -execdir runs in for e and the name the
// command is given there ("./name").
func dirArg(e *entry) (dir, arg string) {
	base := baseName(e.path)
	if isSep(base[0]) {
		return dirName(e.path), base
	}
	return dirName(e.path), "./" + base
}

// execOne runs the command of a ';' action for one file. It is true when
// the command exits with status 0.
func (f *Find) execOne(x *execAction, e *entry) bool {
	arg, dir := e.path, ""
	if x.dir {
		dir, arg = dirArg(e)
	}
	argv := make([]string, len(x.argv))
	toolArgv := make([]string, len(x.argv))
	for i, a := range x.argv {
		argv[i] = strings.ReplaceAll(a, "{}", arg)
		toolArgv[i] = strings.ReplaceAll(a, "{}", e.path)
	}

	if x.ok {
		f.flushOutputs()
		if !f.ask("< " + x.argv[0] + " ... " + e.path + " > ? ") {
			return false
		}
	}
	status, ran := f.runCommand(x, argv, toolArgv, dir)
	return ran && status == 0
}

// addBatch adds e to the command line a + action is building, running the
// command first when the line is full or (for -execdir) the directory
// changes.
func (f *Find) addBatch(x *execAction, e *entry) {
	arg, dir := e.path, ""
	if x.dir {
		dir, arg = dirArg(e)
		if len(x.batch) > 0 && dir != x.batchDir {
			f.flushBatch(x)
		}
	}
	if len(x.batch) > 0 && x.batchSize+len(arg)+1 > batchLimit {
		f.flushBatch(x)
	}
	x.batch = append(x.batch, arg)
	x.batchTool = append(x.batchTool, e.path)
	x.batchDir = dir
	x.batchSize += len(arg) + 1
}

// flushBatch runs a + action's command on the files collected. The action
// itself is always true; a command that fails makes find's status 1.
func (f *Find) flushBatch(x *execAction) {
	if len(x.batch) == 0 {
		return
	}
	argv := slices.Concat(x.argv, x.batch)
	toolArgv := slices.Concat(x.argv, x.batchTool)
	for _, arg := range x.batch {
		x.batchSize -= len(arg) + 1
	}
	x.batch, x.batchTool = x.batch[:0], x.batchTool[:0]

	if status, ran := f.runCommand(x, argv, toolArgv, x.batchDir); !ran || status != 0 {
		f.failed = true
	}
}

// runCommand runs argv, in dir for -execdir. A command named after a gnux
// tool runs in this process, where there is no directory to change into
// without moving the whole walk, so it is given toolArgv instead: the same
// command with each file's path as find knows it in place of "./name".
//
// ran is false when the command could not be started, which is diagnosed.
func (f *Find) runCommand(x *execAction, argv, toolArgv []string, dir string) (status int, ran bool) {
	f.flushOutputs()
	s := f.streams
	if x.ok {
		// As in GNU, a command run by -ok does not read find's input, which
		// the answers come from.
		s.In = strings.NewReader("")
	}

	var res run.Result
	var err error
	switch {
	case x.dir && f.Exec != nil:
		if status, found := f.Exec(toolArgv[0], toolArgv[1:], s); found {
			return status, true
		}
		res, err = run.Command(nil, argv, dir, s)
	default:
		res, err = run.Command(f.Exec, argv, dir, s)
	}
	if err != nil {
		f.diag.Errorf("'%s': %s", argv[0], errText(err))
		return 0, false
	}
	return res.Status, true
}
