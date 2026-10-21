// Package mknod implements the engine behind mknodx, a GNU-compatible
// implementation of the GNU/UNIX mknod command: create a block or character
// device file, or a named pipe.
package mknod

import (
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	filemode "github.com/braswelljr/gnux/core/internal/mode"
	"github.com/braswelljr/gnux/core/internal/quotearg"
	"github.com/braswelljr/gnux/core/internal/special"
	"github.com/braswelljr/gnux/core/mkfifo"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "mknodx"

// Options mirrors the GNU mknod flag surface. The zero value is plain
// mknod: files readable and writable by all, less the umask.
type Options struct {
	// Mode (-m) is the permission mode, octal or symbolic as chmod takes it,
	// applied to a=rw. Given, it is set exactly, whatever the umask.
	Mode string
	// Context (--context=CTX) is an SELinux context. gnux runs as GNU does
	// on a kernel without SELinux: it warns that the context is ignored.
	Context string
}

// Mknod is the engine. Construct it with New.
type Mknod struct {
	Options

	// Umask returns the file-creation mask; MakeFifo and MakeDevice create
	// the file. They are the injection points tests use.
	Umask      func() uint32
	MakeFifo   func(path string, perm uint32) error
	MakeDevice func(path string, perm uint32, block bool, major, minor uint32) error

	diag *core.Diag
}

// New builds a mknod engine writing its diagnostics to s.
func New(opts Options, s core.Streams) *Mknod {
	return &Mknod{Options: opts, Umask: filemode.Umask, MakeFifo: special.Mkfifo, MakeDevice: special.Mknod, diag: core.NewDiag(Name, s)}
}

func quote(s string) string { return quotearg.Quote(s, quotearg.Options{Style: quotearg.Locale}) }

// Run creates NAME of TYPE: "NAME p" for a pipe, "NAME b|c|u MAJOR MINOR"
// for a device. Only TYPE's first letter counts, so "character" is c.
func (m *Mknod) Run(args []string) error {
	if m.Context != "" {
		m.diag.Warnf("warning: ignoring --context; it requires an SELinux/SMACK-enabled kernel")
	}
	perm, explicit, ok := mkfifo.Permissions(m.Mode, m.Umask, m.diag)
	if !ok {
		return m.diag.Result()
	}

	expected := 4
	if len(args) == 0 || (len(args) > 1 && strings.HasPrefix(args[1], "p")) {
		expected = 2
	}
	if len(args) < expected {
		if len(args) == 0 {
			return core.Usagef("missing operand")
		}
		msg := "missing operand after " + quote(args[len(args)-1])
		if expected == 4 && len(args) == 2 {
			msg += "\nSpecial files require major and minor device numbers."
		}
		return core.Usagef("%s", msg)
	}
	if len(args) > expected {
		msg := "extra operand " + quote(args[expected])
		if expected == 2 && len(args) == 4 {
			msg += "\nFifos do not have major and minor device numbers."
		}
		return core.Usagef("%s", msg)
	}

	name := args[0]
	q := quotearg.Quote(name, quotearg.Options{Style: quotearg.ShellEscape, Also: ":"})
	var err error
	switch args[1][0] {
	case 'b', 'c', 'u':
		major, ok := number(args[2])
		if !ok {
			m.diag.Errorf("invalid major device number %s", quote(args[2]))
			return m.diag.Result()
		}
		minor, ok := number(args[3])
		if !ok {
			m.diag.Errorf("invalid minor device number %s", quote(args[3]))
			return m.diag.Result()
		}
		err = m.MakeDevice(name, perm, args[1][0] == 'b', major, minor)
	case 'p':
		err = m.MakeFifo(name, perm)
	default:
		return core.Usagef("invalid device type %s", quote(args[1]))
	}
	if err != nil {
		m.diag.Errorf("%s: %s", q, core.ErrText(err))
		return m.diag.Result()
	}
	if explicit {
		if err := fsys.Chmod(name, filemode.FileMode(perm)); err != nil {
			m.diag.Errorf("cannot set permissions of %s: %s", quotearg.Quote(name, quotearg.Options{Style: quotearg.ShellEscapeAlways}), core.ErrText(err))
		}
	}
	return m.diag.Result()
}

// number reads a device number as C's strtoumax with base 0 does: 0x for
// hexadecimal, a leading 0 for octal, decimal otherwise. It must fit in 32
// bits.
func number(s string) (uint32, bool) {
	t := strings.TrimLeft(s, " \t\n\v\f\r")
	base := 10
	switch {
	case strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "0X"):
		base, t = 16, t[2:]
	case len(t) > 1 && t[0] == '0':
		base, t = 8, t[1:]
	}
	if t == "" || t[0] == '+' || t[0] == '-' || strings.Contains(t, "_") {
		return 0, false
	}
	n, err := strconv.ParseUint(t, base, 32)
	return uint32(n), err == nil
}
