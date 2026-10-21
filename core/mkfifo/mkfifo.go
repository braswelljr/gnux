// Package mkfifo implements the engine behind mkfifox, a GNU-compatible
// implementation of the GNU/UNIX mkfifo command: create named pipes.
package mkfifo

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	filemode "github.com/braswelljr/gnux/core/internal/mode"
	"github.com/braswelljr/gnux/core/internal/quotearg"
	"github.com/braswelljr/gnux/core/internal/special"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "mkfifox"

// Options mirrors the GNU mkfifo flag surface. The zero value is plain
// mkfifo: pipes readable and writable by all, less the umask.
type Options struct {
	// Mode (-m) is the permission mode, octal or symbolic as chmod takes it,
	// applied to a=rw. Given, it is set exactly, whatever the umask.
	Mode string
	// Context (--context=CTX) is an SELinux context. gnux runs as GNU does
	// on a kernel without SELinux: it warns that the context is ignored.
	Context string
}

// Mkfifo is the engine. Construct it with New.
type Mkfifo struct {
	Options

	// Umask returns the file-creation mask; Make creates one pipe. They are
	// the injection points tests use.
	Umask func() uint32
	Make  func(path string, perm uint32) error

	diag *core.Diag
}

// New builds a mkfifo engine writing its diagnostics to s.
func New(opts Options, s core.Streams) *Mkfifo {
	return &Mkfifo{Options: opts, Umask: filemode.Umask, Make: special.Mkfifo, diag: core.NewDiag(Name, s)}
}

// Run creates each NAME; one that cannot be created is reported and the
// rest are still created.
func (m *Mkfifo) Run(args []string) error {
	if m.Context != "" {
		m.diag.Warnf("warning: ignoring --context; it requires an SELinux/SMACK-enabled kernel")
	}
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	perm, explicit, ok := Permissions(m.Mode, m.Umask, m.diag)
	if !ok {
		return m.diag.Result()
	}
	for _, name := range args {
		if err := m.Make(name, perm); err != nil {
			m.diag.Errorf("cannot create fifo %s: %s", quoteaf(name), core.ErrText(err))
			continue
		}
		if explicit {
			if err := fsys.Chmod(name, filemode.FileMode(perm)); err != nil {
				m.diag.Errorf("cannot set permissions of %s: %s", quoteaf(name), core.ErrText(err))
			}
		}
	}
	return m.diag.Result()
}

// Permissions reads a -m MODE for a new pipe or device file, as mkfifo and
// mknod do: applied to a=rw, and limited to the permission bits. It reports
// a mistake through d and returns ok false. Without a MODE it returns
// a=rw, which the system then masks with the umask.
func Permissions(mode string, umask func() uint32, d *core.Diag) (perm uint32, explicit, ok bool) {
	if mode == "" {
		return 0o666, false, true
	}
	perm, err := filemode.Parse(mode, 0o666, false, umask)
	if err != nil {
		d.Errorf("invalid mode")
		return 0, false, false
	}
	if perm&^0o777 != 0 {
		d.Errorf("mode must specify only file permission bits")
		return 0, false, false
	}
	return perm, true, true
}

func quoteaf(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.ShellEscapeAlways})
}
