package mkfifo_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mkfifo"
)

type made struct {
	path string
	perm uint32
}

func newEngine(opts mkfifo.Options) (*mkfifo.Mkfifo, *bytes.Buffer, *[]made) {
	var errb bytes.Buffer
	m := mkfifo.New(opts, core.Streams{In: strings.NewReader(""), Out: &errb, Err: &errb})
	got := &[]made{}
	m.Umask = func() uint32 { return 0o022 }
	m.Make = func(path string, perm uint32) error {
		if strings.Contains(path, "missing") {
			return os.ErrNotExist
		}
		*got = append(*got, made{path, perm})
		return os.WriteFile(path, nil, 0o600)
	}
	return m, &errb, got
}

func TestModes(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		mode string
		want uint32
	}{
		{"", 0o666},
		{"600", 0o600},
		{"u=rw,go=", 0o600},
		{"a+x", 0o777},
		{"+X", 0o666}, // no one may search a new pipe, so X adds nothing
	}
	for i, tt := range tests {
		m, errb, got := newEngine(mkfifo.Options{Mode: tt.mode})
		p := filepath.Join(dir, string(rune('a'+i)))
		if err := m.Run([]string{p}); err != nil {
			t.Fatalf("%q: %v (%s)", tt.mode, err, errb)
		}
		if len(*got) != 1 || (*got)[0].perm != tt.want {
			t.Errorf("-m %q: made %+v, want %#o", tt.mode, *got, tt.want)
		}
	}
}

func TestModeMistakes(t *testing.T) {
	for mode, want := range map[string]string{
		"rw":   "mkfifox: invalid mode\n",
		"8":    "mkfifox: invalid mode\n",
		"1666": "mkfifox: mode must specify only file permission bits\n",
		"u+s":  "mkfifox: mode must specify only file permission bits\n",
	} {
		m, errb, got := newEngine(mkfifo.Options{Mode: mode})
		if err := m.Run([]string{"x"}); !errors.Is(err, core.ErrFailed) || errb.String() != want || len(*got) != 0 {
			t.Errorf("-m %q: err=%v stderr=%q", mode, err, errb)
		}
	}
}

func TestFailureDoesNotStopTheRest(t *testing.T) {
	dir := t.TempDir()
	m, errb, got := newEngine(mkfifo.Options{})
	ok := filepath.Join(dir, "ok")
	if err := m.Run([]string{"missing/a", ok}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "mkfifox: cannot create fifo 'missing/a': No such file or directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	if len(*got) != 1 || (*got)[0].path != ok {
		t.Fatalf("made %+v", *got)
	}
}

func TestOperandAndContext(t *testing.T) {
	m, errb, _ := newEngine(mkfifo.Options{Context: "ctx"})
	var usage *core.UsageError
	if err := m.Run(nil); !errors.As(err, &usage) || usage.Msg != "missing operand" {
		t.Fatalf("err = %v", err)
	}
	if want := "mkfifox: warning: ignoring --context; it requires an SELinux/SMACK-enabled kernel\n"; errb.String() != want {
		t.Fatalf("stderr = %q", errb)
	}
}
