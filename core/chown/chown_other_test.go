//go:build !unix

package chown_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chown"
)

func TestNotSupported(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var errb bytes.Buffer
	err := chown.New(chown.Options{}, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb}).Run([]string{"0:0", f})
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "chownx: changing ownership of '" + f + "': Operation not supported\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}
