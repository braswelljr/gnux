package sync_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sync"
)

func newEngine(opts sync.Options) (*sync.Sync, *bytes.Buffer, *[]sync.Mode, *int) {
	var errb bytes.Buffer
	y := sync.New(opts, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb})
	modes := &[]sync.Mode{}
	all := new(int)
	y.SyncAll = func() error { *all++; return nil }
	y.SyncFile = func(_ *os.File, m sync.Mode) error { *modes = append(*modes, m); return nil }
	return y, &errb, modes, all
}

func TestSyncAll(t *testing.T) {
	y, errb, modes, all := newEngine(sync.Options{})
	if err := y.Run(nil); err != nil || *all != 1 || len(*modes) != 0 {
		t.Fatalf("err=%v all=%d modes=%v stderr=%q", err, *all, *modes, errb)
	}
}

func TestSyncFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	if err := os.WriteFile(a, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		opts sync.Options
		want sync.Mode
	}{
		{sync.Options{}, sync.File},
		{sync.Options{Data: true}, sync.Data},
		{sync.Options{FileSystem: true}, sync.FileSystem},
	} {
		y, errb, modes, all := newEngine(tt.opts)
		if err := y.Run([]string{a, dir}); err != nil {
			t.Fatalf("%+v: %v (%q)", tt.opts, err, errb)
		}
		if *all != 0 || len(*modes) != 2 || (*modes)[0] != tt.want {
			t.Fatalf("%+v: all=%d modes=%v", tt.opts, *all, *modes)
		}
	}
}

func TestMissingFileDoesNotStopTheRest(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope")
	y, errb, modes, _ := newEngine(sync.Options{})
	if err := y.Run([]string{missing, dir}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "syncx: error opening '" + missing + "': No such file or directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	if len(*modes) != 1 {
		t.Fatalf("modes = %v, want the directory synced", *modes)
	}
}

func TestOptionMistakes(t *testing.T) {
	for _, tt := range []struct {
		opts sync.Options
		args []string
		want string
	}{
		{sync.Options{Data: true}, nil, "syncx: --data needs at least one argument\n"},
		{sync.Options{Data: true, FileSystem: true}, []string{"x"}, "syncx: cannot specify both --data and --file-system\n"},
	} {
		y, errb, _, all := newEngine(tt.opts)
		if err := y.Run(tt.args); !errors.Is(err, core.ErrFailed) || errb.String() != tt.want || *all != 0 {
			t.Fatalf("%+v: err=%v stderr=%q", tt.opts, err, errb)
		}
	}
}

func TestRealSync(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	var errb bytes.Buffer
	if err := sync.New(sync.Options{}, core.Streams{In: strings.NewReader(""), Out: &errb, Err: &errb}).Run([]string{f}); err != nil {
		t.Fatalf("%v: %s", err, errb.String())
	}
}
