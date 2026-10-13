package sed

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// The file names GNU sed gives a meaning of its own.
const (
	devStdin  = "/dev/stdin"
	devStdout = "/dev/stdout"
	devStderr = "/dev/stderr"
)

// output is somewhere the script writes: standard output, the file being
// edited in place, or the file of a w command.
type output struct {
	m    *machine
	w    *bufio.Writer
	f    *os.File // closed at the end; nil for a stream
	name string   // for a diagnostic
	sync bool     // flush after every write
	// missing says the last line written came from an input line with no
	// newline at its end, and so was written without one. It gets one only
	// if something else is written after it: the input's last line keeps
	// its missing newline.
	missing bool
}

// fail reports a write error and stops the run. A reader that has gone away
// stops it quietly.
func (o *output) fail(err error) {
	if core.IsClosedPipe(err) {
		panic(abort{})
	}
	o.m.fatalf("couldn't write 1 item to %s: %s", o.name, core.ErrText(err))
}

// begin ends a line left without its newline, before more is written.
func (o *output) begin() {
	if o.missing {
		o.missing = false
		_ = o.w.WriteByte(o.m.delim)
	}
}

// end finishes one write.
func (o *output) end(err error) {
	if err == nil && o.sync {
		err = o.w.Flush()
	}
	if err != nil {
		o.fail(err)
	}
}

// line writes b as one line. chomped says whether the input line it comes
// from had a newline at its end.
func (o *output) line(b []byte, chomped bool) {
	o.begin()
	_, err := o.w.Write(b)
	if chomped {
		err = o.w.WriteByte(o.m.delim)
	} else {
		o.missing = true
	}
	o.end(err)
}

// text writes b as it is: text that brings its own newline.
func (o *output) text(b []byte) {
	o.begin()
	_, err := o.w.Write(b)
	o.end(err)
}

// flush empties the buffer at the end of the run.
func (o *output) flush() {
	if err := o.w.Flush(); err != nil {
		o.fail(err)
	}
}

// openOutputs opens the file of every w command and w flag before any input
// is read, as GNU sed does: the files exist, emptied, even if nothing is
// ever written to them.
func (m *machine) openOutputs() {
	for i := range m.prog {
		c := &m.prog[i]
		if c.file == "" || c.name == 'r' || c.name == 'R' {
			continue
		}
		switch c.file {
		case devStdout:
			c.out = m.stdout
		case devStderr:
			if m.stderr == nil {
				m.stderr = &output{m: m, w: bufio.NewWriter(m.e.streams.Err), name: "stderr", sync: true}
			}
			c.out = m.stderr
		default:
			c.out = m.outputs[c.file]
		}
		if c.out != nil {
			continue
		}
		f, err := os.Create(c.file)
		if err != nil {
			m.fatalf("couldn't open file %s: %s", c.file, core.ErrText(err))
		}
		// Flushed after every line, so that the file can be read back by
		// the same script or followed by another program.
		c.out = &output{m: m, w: core.NewWriter(f), f: f, name: c.file, sync: true}
		if m.outputs == nil {
			m.outputs = map[string]*output{}
		}
		m.outputs[c.file] = c.out
	}
}

// edit is the state of the file being edited in place.
type edit struct {
	name string   // as given on the command line
	path string   // the file that is rewritten
	temp *os.File // where its new contents go
}

// openNext moves to the next input that can be opened, and reports whether
// there was one.
func (m *machine) openNext() bool {
	for len(m.args) > 0 {
		name := m.args[0]
		m.args = m.args[1:]
		if m.open(name) {
			return true
		}
	}
	return false
}

// open makes name the current input. A file that cannot be opened is
// diagnosed and skipped.
func (m *machine) open(name string) bool {
	// With -i there is no standard input: "-" is a file of that name.
	if name == "-" && !m.e.InPlace {
		m.cur = m.stdin
	} else {
		path := name
		if m.e.InPlace && m.e.FollowSymlinks {
			if resolved, err := filepath.EvalSymlinks(name); err == nil {
				path = resolved
			}
		}
		f, err := os.Open(path)
		if err != nil {
			m.diag.Errorf("can't read %s: %s", name, core.ErrText(err))
			m.badInput = true
			return false
		}
		if m.e.InPlace {
			m.beginEdit(name, path, f)
		}
		if m.files == nil {
			m.files = core.NewReader(f)
		} else {
			m.files.Reset(f)
		}
		m.cur, m.curFile = m.files, f
	}
	m.name = name
	m.atEnd = false
	if m.separate {
		// Each file starts afresh: its own line numbers, and the files
		// of R commands read from their beginning again.
		m.lineNo = 0
		m.closeReaders()
	}
	return true
}

// beginEdit sends the script's output to a temporary file beside the one
// being edited, which replaces it when the file has been read.
func (m *machine) beginEdit(name, path string, f *os.File) {
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		m.fatalf("couldn't edit %s: not a regular file", name)
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "sed")
	if err != nil {
		f.Close()
		m.fatalf("couldn't open temporary file %s: %s", filepath.Join(dir, "sedXXXXXX"), core.ErrText(err))
	}
	// The new file takes the old one's permissions. Where they cannot be
	// set (a filesystem without them) the edit still goes ahead.
	_ = os.Chmod(temp.Name(), info.Mode().Perm())
	m.edit = &edit{name: name, path: path, temp: temp}
	m.out = &output{m: m, w: core.NewWriter(temp), name: temp.Name()}
}

// closeInput leaves the current input and, when it was being edited in
// place, puts the new contents in its place.
func (m *machine) closeInput() {
	if m.curFile != nil {
		m.curFile.Close()
		m.curFile = nil
	}
	m.cur = nil
	if m.edit == nil {
		return
	}
	// m.edit stays set until the file is in place: cleanup removes the
	// temporary file if anything below fails.
	ed, out := m.edit, m.out
	m.out = m.stdout
	out.flush()
	if err := ed.temp.Close(); err != nil {
		out.fail(err)
	}
	if m.e.Suffix != "" {
		if err := os.Rename(ed.path, backupName(ed.path, m.e.Suffix)); err != nil {
			m.fatalf("cannot rename %s: %s", ed.path, core.ErrText(err))
		}
	}
	if err := os.Rename(ed.temp.Name(), ed.path); err != nil {
		m.fatalf("cannot rename %s: %s", ed.temp.Name(), core.ErrText(err))
	}
	m.edit = nil
}

// backupName is where the original of a file edited in place is kept: its
// name with suffix appended, or suffix with each '*' replaced by its base
// name. A result that names a directory is used as it is; otherwise the
// backup sits beside the file.
func backupName(path, suffix string) string {
	dir, base := filepath.Split(path)
	name := base + suffix
	if strings.Contains(suffix, "*") {
		name = strings.ReplaceAll(suffix, "*", base)
	}
	if strings.ContainsAny(name, `/`+string(filepath.Separator)) {
		return name
	}
	return dir + name
}

// more reports whether another line can be read. cross lets it move on to
// the following input files when the current one is exhausted; without it
// the answer is about the current file alone.
func (m *machine) more(cross bool) bool {
	for {
		if m.cur != nil {
			if !m.atEnd {
				_, err := m.cur.Peek(1)
				if err == nil {
					return true
				}
				if !errors.Is(err, io.EOF) {
					m.fatalf("read error on %s: %s", m.name, core.ErrText(err))
				}
				m.atEnd = true
			}
			if !cross {
				return false
			}
			m.closeInput()
		}
		if !m.openNext() {
			return false
		}
	}
}

// noNextLine reports whether the line in the pattern space is the last one
// n or N could follow: the last of the input, or of its file with -s or -i.
func (m *machine) noNextLine() bool {
	return !m.more(!m.separate)
}

// lastLine is the $ address.
func (m *machine) lastLine() bool {
	return m.noNextLine()
}

// readLine starts a new line: the text queued by a, r and R is written, and
// the next input line replaces the pattern space, or is added to it after a
// newline when join is set. It reports false at the end of the input.
func (m *machine) readLine(join bool) bool {
	m.flushQueue()
	// What the script has written so far is passed on before a read that
	// will wait for the producer: see core.FlushIfIdle.
	if m.cur == nil || m.cur.Buffered() == 0 {
		_ = m.stdout.w.Flush()
	}
	if !m.more(true) {
		return false
	}
	m.replaced = false
	if join {
		m.ps = append(m.ps, m.delim)
	} else {
		m.ps = m.ps[:0]
	}
	for {
		frag, err := m.cur.ReadSlice(m.delim)
		m.ps = append(m.ps, frag...)
		if err == nil {
			m.ps = m.ps[:len(m.ps)-1]
			m.chomped = true
			break
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if !errors.Is(err, io.EOF) {
			m.fatalf("read error on %s: %s", m.name, core.ErrText(err))
		}
		m.chomped = false
		m.atEnd = true
		break
	}
	m.lineNo++
	return true
}

// flushQueue writes what a, r and R queued during the cycle.
func (m *machine) flushQueue() {
	for i := range m.queue {
		q := &m.queue[i]
		if q.file == "" {
			m.out.text(q.text)
			continue
		}
		// A file that cannot be read is no error: it adds nothing.
		var src io.Reader = m.stdin
		if q.file != devStdin {
			f, err := os.Open(q.file)
			if err != nil {
				continue
			}
			src = f
		}
		m.out.begin()
		_, err := io.Copy(m.out.w, src)
		if c, ok := src.(io.Closer); ok {
			c.Close()
		}
		if err != nil {
			// Only a failure to write counts, which the writer remembers;
			// one to read just ends the copy.
			_, err = m.out.w.Write(nil)
		}
		m.out.end(err)
	}
	m.queue = m.queue[:0]
}

// lineReader is the file of an R command, read one line per use.
type lineReader struct {
	r *bufio.Reader
	f *os.File
}

// readFileLine returns the next line of the file an R command names, its
// newline included, or nothing when the file cannot be read or has ended.
func (m *machine) readFileLine(name string) []byte {
	lr, ok := m.readers[name]
	if !ok {
		lr = &lineReader{}
		if name == devStdin {
			lr.r = m.stdin
		} else if f, err := os.Open(name); err == nil {
			lr.f, lr.r = f, core.NewReader(f)
		}
		if m.readers == nil {
			m.readers = map[string]*lineReader{}
		}
		m.readers[name] = lr
	}
	if lr.r == nil {
		return nil
	}
	line, _ := lr.r.ReadBytes(m.delim)
	return line
}

func (m *machine) closeReaders() {
	for name, lr := range m.readers {
		if lr.f != nil {
			lr.f.Close()
		}
		delete(m.readers, name)
	}
}

// finish ends a run that was not aborted: the file being edited is put in
// place and every output is flushed.
func (m *machine) finish() {
	m.closeInput()
	m.stdout.flush()
	for _, o := range m.outputs {
		o.flush()
	}
}

// cleanup releases what the run holds, however it ended.
func (m *machine) cleanup() {
	if m.curFile != nil {
		m.curFile.Close()
	}
	if m.edit != nil {
		// The run stopped before the file was replaced: it is left as it
		// was, and the unfinished copy goes.
		m.edit.temp.Close()
		_ = os.Remove(m.edit.temp.Name())
	}
	m.closeReaders()
	for _, o := range m.outputs {
		o.f.Close()
	}
}
