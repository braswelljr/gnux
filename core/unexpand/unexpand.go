// Package unexpand implements the engine behind unexpandx, a GNU-compatible
// implementation of the GNU/UNIX unexpand command: convert spaces to tabs.
package unexpand

import (
	"bufio"
	"errors"
	"io"
	"syscall"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/expand"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "unexpandx"

// Options mirrors the GNU unexpand flag surface. The zero value is plain
// unexpand: only the blanks at the start of each line are converted, with tab
// stops every 8 columns.
type Options struct {
	// All (-a) converts every run of two or more blanks before a tab stop,
	// not only the leading ones. On the command line -t implies it; through
	// the library it is set explicitly.
	All bool
	// FirstOnly (--first-only) converts only the leading blanks, even when
	// All is set.
	FirstOnly bool
	// Tabs are tab-stop lists in GNU's -t syntax (see expand.Options.Tabs),
	// applied in order.
	Tabs []string
}

// Unexpand is the engine. Construct it with New.
type Unexpand struct {
	Options

	streams core.Streams
	diag    *core.Diag
}

// New builds an unexpand engine reading from and writing to s.
func New(opts Options, s core.Streams) *Unexpand {
	return &Unexpand{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// Run converts the operands, read as one stream; no operands, or "-", means
// standard input. An invalid tab-stop list stops the run before any input is
// read. The returned error is core.ErrFailed if anything failed, otherwise
// nil. The algorithm is a port of GNU unexpand's, byte for byte.
func (u *Unexpand) Run(args []string) error {
	tabs, ok := expand.Build(u.Tabs, u.diag)
	if !ok {
		return core.ErrFailed
	}
	entire := u.All && !u.FirstOnly

	in := expand.NewInput(args, u.streams, u.diag)
	defer in.Close()
	out := bufio.NewWriter(u.streams.Out)
	pending := make([]byte, 0, min(max(tabs.MaxWidth(), 1), 4096))

	for {
		convert := true
		var col, next uint64
		idx := 0
		oneBlankBeforeStop := false
		prevBlank := true
		pending = pending[:0]

		for {
			c, err := in.ReadByte()
			eof := err != nil
			if convert {
				blank := !eof && (c == ' ' || c == '\t')
				switch {
				case blank:
					var last bool
					next, last = tabs.Next(col, &idx)
					if last {
						convert = false
					}
					if convert {
						if next < col {
							u.diag.Errorf("input line is too long")
							_ = out.Flush()
							return core.ErrFailed
						}
						if c == '\t' {
							col = next
							if len(pending) > 0 {
								pending[0] = '\t'
							}
						} else {
							col++
							if !prevBlank || col != next {
								// Not yet known whether these blanks become a tab.
								if col == next {
									oneBlankBeforeStop = true
								}
								pending = append(pending, c)
								prevBlank = true
								continue
							}
							// Replace the pending blanks by a tab or two.
							c = '\t'
							if len(pending) > 0 {
								pending[0] = c
							}
						}
						// Discard pending blanks, unless it was a single blank
						// just before the previous tab stop.
						if oneBlankBeforeStop {
							pending = pending[:1]
						} else {
							pending = pending[:0]
						}
					}
				case !eof && c == '\b':
					if col > 0 {
						col--
					}
					next = col
					if idx > 0 {
						idx--
					}
				case !eof:
					col++
					if col == 0 {
						u.diag.Errorf("input line is too long")
						_ = out.Flush()
						return core.ErrFailed
					}
				}
				if len(pending) > 0 {
					if len(pending) > 1 && oneBlankBeforeStop {
						pending[0] = '\t'
					}
					_, _ = out.Write(pending)
					pending = pending[:0]
					oneBlankBeforeStop = false
				}
				prevBlank = blank
				convert = convert && (entire || blank)
			}
			if eof {
				return u.finish(out)
			}
			if werr := out.WriteByte(c); werr != nil {
				return u.writeFailed(werr)
			}
			if c == '\n' {
				break
			}
		}
	}
}

func (u *Unexpand) finish(out *bufio.Writer) error {
	if err := out.Flush(); err != nil {
		return u.writeFailed(err)
	}
	return u.diag.Result()
}

func (u *Unexpand) writeFailed(err error) error {
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe) {
		return u.diag.Result()
	}
	u.diag.Errorf("write error: %s", core.ErrText(err))
	return core.ErrFailed
}
