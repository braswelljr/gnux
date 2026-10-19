// Package sum implements the engine behind sumx, a GNU-compatible
// implementation of the GNU sum command: print the 16-bit checksum and block
// count of each file. The checksums are core/cksum's, which cksumx prints
// with -a bsd and -a sysv.
package sum

import (
	"bufio"
	"strconv"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cksum"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "sumx"

// Options mirrors the GNU sum flag surface. The zero value is plain sum: the
// BSD checksum with 1024-byte blocks, which -r also selects.
type Options struct {
	// SysV (-s, --sysv) uses the System V checksum and 512-byte blocks.
	SysV bool
}

// Sum is the engine. Construct it with New.
type Sum struct {
	Options

	streams core.Streams
	diag    *core.Diag
}

// New builds a sum engine reading from and writing to s.
func New(opts Options, s core.Streams) *Sum {
	return &Sum{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// Run prints the checksum of each operand (standard input when there is
// none, or for "-"). As in GNU sum 9, an operand's name follows its checksum,
// and standard input read because there is no operand prints no name. It
// returns core.ErrFailed when any operand failed, and nil otherwise.
func (m *Sum) Run(args []string) (err error) {
	out := core.NewWriter(m.streams.Out)
	defer core.FlushOutput(out, m.diag, &err)
	if len(args) == 0 {
		m.file(out, "-", false)
	}
	for _, name := range args {
		m.file(out, name, true)
		// Keep output and diagnostics in operand order on a shared terminal.
		_ = out.Flush()
	}
	return m.diag.Result()
}

func (m *Sum) file(out *bufio.Writer, name string, named bool) {
	r := m.streams.In
	if name != "-" {
		f, err := fsys.Open(name)
		if err != nil {
			m.diag.Errorf("%s: %s", name, core.ErrText(err))
			return
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && info.IsDir() {
			m.diag.Errorf("%s: %s", name, core.ErrText(syscall.EISDIR))
			return
		}
		r = f
	}

	var line []byte
	if m.SysV {
		s, n, err := cksum.SysV(r)
		if err != nil {
			m.diag.Errorf("%s: %s", name, core.ErrText(err))
			return
		}
		line = core.AppendInt(line, int64(s), 0)
		line = append(line, ' ')
		line = core.AppendInt(line, int64((n+511)/512), 0)
	} else {
		s, n, err := cksum.BSD(r)
		if err != nil {
			m.diag.Errorf("%s: %s", name, core.ErrText(err))
			return
		}
		line = appendZeroPadded(line, s, 5)
		line = append(line, ' ')
		line = core.AppendInt(line, int64((n+1023)/1024), 5)
	}
	if named {
		line = append(line, ' ')
		line = append(line, name...)
	}
	line = append(line, '\n')
	_, _ = out.Write(line)
}

// appendZeroPadded appends n in decimal, padded with zeros to width, as
// printf's %05d prints it.
func appendZeroPadded(dst []byte, n uint64, width int) []byte {
	var digits [20]byte
	text := strconv.AppendUint(digits[:0], n, 10)
	for pad := width - len(text); pad > 0; pad-- {
		dst = append(dst, '0')
	}
	return append(dst, text...)
}
