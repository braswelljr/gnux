//go:build unix

package mkfifo_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mkfifo"
)

func TestRealFifo(t *testing.T) {
	p := filepath.Join(t.TempDir(), "pipe")
	var errb bytes.Buffer
	if err := mkfifo.New(mkfifo.Options{Mode: "640"}, core.Streams{In: strings.NewReader(""), Out: &errb, Err: &errb}).Run([]string{p}); err != nil {
		t.Fatalf("%v: %s", err, errb.String())
	}
	info, err := os.Lstat(p)
	if err != nil || info.Mode()&os.ModeNamedPipe == 0 || info.Mode().Perm() != 0o640 {
		t.Fatalf("made %v, %v", info.Mode(), err)
	}
}
