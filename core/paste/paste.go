// Package paste implements the engine behind pastex, a GNU-compatible
// implementation of the GNU/UNIX paste command: merge lines of files.
package paste

import (
	"bufio"
	"errors"
	"io"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "pastex"

// Options mirrors the GNU paste flag surface. The zero value is plain paste:
// corresponding lines of every file joined with a TAB.
type Options struct {
	// Delimiters (-d LIST) is GNU's delimiter list, used circularly. It is
	// written as on the command line: \n, \t, \\, \b, \f, \r, \v are escapes
	// and \0 is the empty delimiter. It applies only when DelimitersSet is
	// true, so the zero value means a TAB.
	Delimiters string
	// DelimitersSet records that -d was given (an empty list is meaningful).
	DelimitersSet bool
	// Serial (-s) pastes one file at a time instead of in parallel.
	Serial bool
	// ZeroTerminated (-z) makes NUL, not newline, the line delimiter.
	ZeroTerminated bool
}

// Paste is the engine. Construct it with New.
type Paste struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	eol     byte
	// delims holds the parsed list; an entry of -1 is the empty delimiter.
	delims []int
	// stdin is shared by every "-" operand, as GNU shares one stream.
	stdin *bufio.Reader
}

// New builds a paste engine reading from and writing to s.
func New(opts Options, s core.Streams) *Paste {
	p := &Paste{Options: opts, streams: s, diag: core.NewDiag(Name, s), eol: '\n'}
	if opts.ZeroTerminated {
		p.eol = 0
	}
	return p
}

// emptyDelim marks the \0 entry of a delimiter list: nothing is written.
const emptyDelim = -1

// parseDelims expands GNU's escapes in a delimiter list. ok is false when the
// list ends with an unescaped backslash.
func parseDelims(list string) (out []int, ok bool) {
	for i := 0; i < len(list); i++ {
		c := list[i]
		if c != '\\' {
			out = append(out, int(c))
			continue
		}
		i++
		if i == len(list) {
			return nil, false
		}
		switch list[i] {
		case '0':
			out = append(out, emptyDelim)
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'v':
			out = append(out, '\v')
		default:
			out = append(out, int(list[i]))
		}
	}
	if len(out) == 0 {
		out = []int{emptyDelim}
	}
	return out, true
}

// Run pastes the operands; no operands means standard input. The returned
// error is core.ErrFailed if anything failed, otherwise nil.
func (p *Paste) Run(args []string) error {
	p.delims = []int{'\t'}
	if p.DelimitersSet {
		d, ok := parseDelims(p.Delimiters)
		if !ok {
			p.diag.Errorf("delimiter list ends with an unescaped backslash: %s", p.Delimiters)
			return p.diag.Result()
		}
		p.delims = d
	}
	if len(args) == 0 {
		args = []string{"-"}
	}
	in := p.streams.In
	if in == nil {
		in = strings.NewReader("")
	}
	p.stdin = core.NewReader(in)
	p.out = core.NewWriter(p.streams.Out)

	if p.Serial {
		p.serial(args)
	} else {
		p.parallel(args)
	}
	if err := p.out.Flush(); err != nil && !core.IsClosedPipe(err) {
		p.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return p.diag.Result()
}

// source is one opened operand.
type source struct {
	name   string
	r      *bufio.Reader
	closer io.Closer
}

func (p *Paste) open(name string) (*source, error) {
	if name == "-" {
		return &source{name: name, r: p.stdin}, nil
	}
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	return &source{name: name, r: core.NewReader(f), closer: f}, nil
}

func (s *source) close() {
	if s.closer != nil {
		s.closer.Close()
	}
}

func (p *Paste) writeDelim(d int) {
	if d != emptyDelim {
		_ = p.out.WriteByte(byte(d))
	}
}

// readLine returns the next line without its delimiter; ok is false at the
// end of the input (a last line with no delimiter is still a line).
func (p *Paste) readLine(s *source) (line []byte, ok bool) {
	line, err := s.r.ReadBytes(p.eol)
	if len(line) > 0 && line[len(line)-1] == p.eol {
		return line[:len(line)-1], true
	}
	if err != nil && !errors.Is(err, io.EOF) {
		p.diag.Errorf("%s: %s", s.name, core.ErrText(err))
	}
	return line, len(line) > 0
}

// parallel is GNU's paste_parallel: one output line per round, each file
// contributing its next line. A file that has ended contributes nothing, but
// its delimiter is kept and written before the next file that has data, so
// the columns stay aligned.
func (p *Paste) parallel(names []string) {
	files := make([]*source, len(names))
	for i, name := range names {
		s, err := p.open(name)
		if err != nil {
			// GNU opens every file first and gives up on the first failure.
			p.diag.Errorf("%s: %s", name, core.ErrText(err))
			for _, f := range files[:i] {
				f.close()
			}
			return
		}
		files[i] = s
	}
	open := len(files)
	n := len(files)
	for open > 0 {
		var saved []int
		di := 0
		nextDelim := func() int {
			d := p.delims[di]
			di = (di + 1) % len(p.delims)
			return d
		}
		for i := 0; i < n && open > 0; i++ {
			var line []byte
			got := false
			if files[i] != nil {
				line, got = p.readLine(files[i])
			}
			if got {
				for _, d := range saved {
					p.writeDelim(d)
				}
				saved = saved[:0]
				_, _ = p.out.Write(line)
				if i+1 == n {
					_ = p.out.WriteByte(p.eol)
					// The next read is of the first file again.
					if files[0] != nil {
						core.FlushIfIdle(p.out, files[0].r)
					}
				} else {
					p.writeDelim(nextDelim())
				}
				continue
			}
			if files[i] != nil {
				files[i].close()
				files[i] = nil
				open--
			}
			if i+1 == n {
				if open > 0 {
					for _, d := range saved {
						p.writeDelim(d)
					}
					_ = p.out.WriteByte(p.eol)
				}
			} else {
				saved = append(saved, nextDelim())
			}
		}
	}
}

// serial is GNU's paste_serial: each file becomes one output line, its lines
// joined by the delimiters, used circularly from the start for every file.
func (p *Paste) serial(names []string) {
	for _, name := range names {
		s, err := p.open(name)
		if err != nil {
			p.diag.Errorf("%s: %s", name, core.ErrText(err))
			continue
		}
		di := 0
		first := true
		for {
			line, ok := p.readLine(s)
			if !ok {
				break
			}
			if !first {
				p.writeDelim(p.delims[di])
				di = (di + 1) % len(p.delims)
			}
			first = false
			_, _ = p.out.Write(line)
		}
		_ = p.out.WriteByte(p.eol)
		s.close()
	}
}
