package themepick

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// ErrNotTerminal is returned by Run when input or output is not a terminal.
var ErrNotTerminal = errors.New("the theme picker needs a terminal")

// Run shows m on the terminal in and out until a theme is chosen (Enter)
// or the picker is left (Esc, Ctrl-C). It draws on the alternate screen,
// so the terminal's contents come back as they were, and it restores the
// terminal on every way out, a SIGTERM or SIGHUP included.
func Run(in, out *os.File, m *Model) (chosen string, ok bool, err error) {
	inFd, outFd := int(in.Fd()), int(out.Fd()) //nolint:gosec // a descriptor fits an int
	if !term.IsTerminal(inFd) || !term.IsTerminal(outFd) {
		return "", false, ErrNotTerminal
	}
	state, err := term.MakeRaw(inFd)
	if err != nil {
		return "", false, err
	}
	undoVT := enableVT(out)
	restore := func() {
		_, _ = out.WriteString("\x1b[0m\x1b[?25h\x1b[?1049l")
		undoVT()
		_ = term.Restore(inFd, state)
	}
	defer restore()

	sigs := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(sigs, exitSignals...)
	go func() {
		select {
		case s := <-sigs:
			restore()
			code := 1
			if n, isNum := s.(syscall.Signal); isNum {
				code = 128 + int(n)
			}
			os.Exit(code)
		case <-done:
		}
	}()
	defer func() { signal.Stop(sigs); close(done) }()

	_, _ = out.WriteString("\x1b[?1049h\x1b[?25l")
	buf := make([]byte, 256)
	for {
		width, height, err := term.GetSize(outFd)
		if err != nil {
			width, height = 80, 24
		}
		if _, err := out.WriteString(m.Frame(width, height)); err != nil {
			return "", false, err
		}
		n, err := in.Read(buf)
		if err != nil {
			return "", false, err
		}
		page := max(height-6, 1)
		for _, k := range ParseKeys(buf[:n]) {
			if done, picked := m.Handle(k, page); done {
				return m.Selected(), picked, nil
			}
		}
	}
}
