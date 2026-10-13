package grep

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
)

// Separators between the parts of an output line.
const (
	sepSelected = ':'
	sepContext  = '-'
)

// search reads r, which is shown as name, line by line and prints what the
// options ask for.
func (g *Grep) search(r io.Reader, name string) {
	g.in.Reset(r)
	g.ringHead, g.ringLen = 0, 0

	var (
		selected  int64 // selected lines so far
		lineno    int64
		offset    int64 // of the line being read
		lastOut   int64 // number of the last line printed; 0 when none
		pending   int   // lines of trailing context still owed
		quiet     = g.quiet
		stopAtHit = g.stopAtHit
		binary    = false
		atBinary  int64 // selected lines when the file turned out binary
	)

	// A file is binary once a NUL byte has been read, which GNU looks for in
	// each buffer; here that is the first buffer, then each line. With -z
	// the NUL ends a line and says nothing.
	lookForNUL := g.BinaryFiles != BinaryFilesText && !g.NullData
	foundNUL := false
	if lookForNUL {
		// Peek(1) fills the buffer with one read, so input that arrives
		// slowly (a terminal, a pipe) is not waited for.
		if _, err := g.in.Peek(1); err == nil {
			first, _ := g.in.Peek(g.in.Buffered())
			foundNUL = bytes.IndexByte(first, 0) >= 0
		}
	}

	for {
		line, terminated, err := g.readLine()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				g.fileError(name, err)
			}
			break
		}
		lineno++
		start := offset
		offset += int64(len(line))
		if terminated {
			offset++
		}

		if lookForNUL && !binary && (foundNUL || bytes.IndexByte(line, 0) >= 0) {
			if g.BinaryFiles == BinaryFilesWithoutMatch {
				// The file is taken not to match, whatever came before.
				selected = 0
				break
			}
			binary = true
			atBinary = selected
			if !g.counting {
				// No more lines are printed; one more selected line is
				// enough to say that the file matches.
				quiet, stopAtHit = true, true
			}
		}

		if g.m.match(line) == g.Invert {
			// Not selected: trailing context, or a candidate for leading
			// context.
			switch {
			case quiet:
			case pending > 0:
				g.printLine(name, line, lineno, start, sepContext)
				lastOut = lineno
				pending--
			case g.Before > 0:
				g.remember(line, start)
			}
			if g.limit >= 0 && selected >= g.limit && pending == 0 {
				break
			}
			if g.stop {
				break
			}
			continue
		}

		if g.limit >= 0 && selected >= g.limit {
			// -m is spent; this line only ends the trailing context.
			break
		}
		selected++
		if !quiet {
			first := lineno - int64(g.ringLen)
			if g.separator != nil && g.used && (lastOut == 0 || first != lastOut+1) {
				g.write(g.separator)
			}
			for i := 0; i < g.ringLen; i++ {
				c := &g.ring[(g.ringHead+i)%len(g.ring)]
				g.printLine(name, c.data, first+int64(i), c.offset, sepContext)
			}
			g.ringHead, g.ringLen = 0, 0
			g.printLine(name, line, lineno, start, sepSelected)
			lastOut = lineno
			pending = g.After
			g.used = true
		}
		if stopAtHit || g.stop {
			break
		}
		if g.limit >= 0 && selected >= g.limit && (quiet || pending == 0) {
			break
		}
	}

	if selected > 0 {
		g.matched = true
		if g.Quiet {
			g.stop = true
		}
	}
	if binary && !g.quiet && selected > atBinary {
		// A diagnostic on standard error since GNU grep 3.5, and one that
		// neither -s silences nor the exit status reflects.
		g.warnf("%s: binary file matches", name)
	}

	switch {
	case g.counting:
		b := g.buf[:0]
		if g.withName {
			b = g.pal.paint(b, g.pal.fn, name)
			if g.Null {
				b = append(b, 0)
			} else {
				b = g.appendSep(b, sepSelected)
			}
		}
		b = strconv.AppendInt(b, selected, 10)
		b = append(b, '\n')
		g.emit(b)
	case g.list > 0 && selected > 0, g.list < 0 && selected == 0:
		b := g.pal.paint(g.buf[:0], g.pal.fn, name)
		if g.Null {
			b = append(b, 0)
		} else {
			b = append(b, '\n')
		}
		g.emit(b)
	}
}

// readLine returns the next line without its terminator. The slice is valid
// until the next call. terminated is false for a last line that ended with
// the input rather than with the delimiter.
func (g *Grep) readLine() (line []byte, terminated bool, err error) {
	line, err = g.in.ReadSlice(g.delim)
	if errors.Is(err, bufio.ErrBufferFull) {
		// Longer than the read buffer: gather it in a buffer of our own,
		// which is kept for the next long line.
		g.long = append(g.long[:0], line...)
		for errors.Is(err, bufio.ErrBufferFull) {
			line, err = g.in.ReadSlice(g.delim)
			g.long = append(g.long, line...)
		}
		line = g.long
	}
	switch {
	case err == nil:
		return line[:len(line)-1], true, nil
	case errors.Is(err, io.EOF) && len(line) > 0:
		return line, false, nil
	}
	return nil, false, err
}

// remember keeps a line that was not printed, in case one of the next
// g.Before lines is selected. The ring grows only as far as it is used, so a
// huge -B costs nothing on its own.
func (g *Grep) remember(line []byte, offset int64) {
	var c *contextLine
	switch {
	case g.ringLen < len(g.ring):
		c = &g.ring[(g.ringHead+g.ringLen)%len(g.ring)]
		g.ringLen++
	case len(g.ring) < g.Before:
		// Full but not at its limit: it has never wrapped, so it is in order
		// and can simply be extended.
		g.ring = append(g.ring, contextLine{})
		c = &g.ring[len(g.ring)-1]
		g.ringLen++
	default:
		c = &g.ring[g.ringHead]
		g.ringHead = (g.ringHead + 1) % len(g.ring)
	}
	c.data = append(c.data[:0], line...)
	c.offset = offset
}

// printLine writes one line of output: a selected line (sep ':') or a context
// line (sep '-'), with its prefix, colored if asked. With -o it writes one
// line per match instead.
func (g *Grep) printLine(name string, line []byte, lineno, offset int64, sep byte) {
	b := g.buf[:0]
	if !g.OnlyMatching {
		b = g.appendHead(b, name, lineno, offset, sep, len(line))
	}
	rest := line

	if g.pal.on || g.OnlyMatching {
		// Whether the line matches the patterns, which with -v is true of
		// the context lines and not of the selected ones.
		matching := (sep == sepSelected) != g.Invert
		lineColor, matchColor := "", ""
		if g.pal.on {
			lineColor = g.pal.cx
			if (sep == sepSelected) != (g.Invert && g.pal.rv) {
				lineColor = g.pal.sl
			}
			matchColor = g.pal.mc
			if sep == sepSelected {
				matchColor = g.pal.ms
			}
		}
		if matching && (g.OnlyMatching || matchColor != "") {
			matchSep := byte(sepSelected)
			if g.Invert {
				matchSep = sepContext
			}
			g.spanBuf = g.m.spans(line, g.spanBuf[:0])
			at := 0
			for _, sp := range g.spanBuf {
				if g.OnlyMatching {
					b = g.appendHead(b, name, lineno, offset+int64(sp.s), matchSep, sp.e-sp.s)
				} else {
					b = g.pal.start(b, lineColor)
					b = append(b, line[at:sp.s]...)
				}
				b = g.pal.start(b, matchColor)
				b = append(b, line[sp.s:sp.e]...)
				b = g.pal.end(b, matchColor)
				if g.OnlyMatching {
					b = append(b, g.delim)
				}
				at = sp.e
			}
			rest = line[at:]
		}
		if !g.OnlyMatching && lineColor != "" {
			// A carriage return ending the line stays outside the color.
			tail := rest
			if n := len(tail); n > 0 && tail[n-1] == '\r' {
				tail = tail[:n-1]
			}
			if len(tail) > 0 {
				b = g.pal.start(b, lineColor)
				b = append(b, tail...)
				b = g.pal.end(b, lineColor)
				rest = rest[len(tail):]
			}
		}
	}

	if !g.OnlyMatching {
		b = append(b, rest...)
		b = append(b, g.delim)
	}
	g.emit(b)
}

// emit writes an assembled piece of output and keeps its buffer for the next.
func (g *Grep) emit(b []byte) {
	g.buf = b
	if len(b) == 0 {
		return
	}
	g.write(b)
	// A line is passed on at once when asked (--line-buffered, a terminal),
	// and whenever the input has run dry: see core.FlushIfIdle.
	if g.flushLines || g.in.Buffered() == 0 {
		g.flush()
	}
}

// appendHead appends the prefix of an output line: the file name, the line
// number and the byte offset, each followed by sep. size is the length of
// what follows, which -T needs: an empty line gets no tab.
func (g *Grep) appendHead(b []byte, name string, lineno, offset int64, sep byte, size int) []byte {
	if g.withName {
		b = g.pal.paint(b, g.pal.fn, name)
		if g.Null {
			b = append(b, 0)
		} else {
			b = g.appendSep(b, sep)
		}
	}
	if g.LineNumber {
		b = g.pal.start(b, g.pal.ln)
		b = strconv.AppendInt(b, lineno, 10)
		b = g.pal.end(b, g.pal.ln)
		b = g.appendSep(b, sep)
	}
	if g.ByteOffset {
		b = g.pal.start(b, g.pal.bn)
		b = strconv.AppendInt(b, offset, 10)
		b = g.pal.end(b, g.pal.bn)
		b = g.appendSep(b, sep)
	}
	if g.InitialTab && (g.withName || g.LineNumber || g.ByteOffset) && size > 0 {
		b = append(b, '\t')
	}
	return b
}

func (g *Grep) appendSep(b []byte, sep byte) []byte {
	b = g.pal.start(b, g.pal.se)
	b = append(b, sep)
	return g.pal.end(b, g.pal.se)
}
