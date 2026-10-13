package core_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core"
)

func TestErrText(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"not exist", &fs.PathError{Op: "open", Path: "x", Err: syscall.ENOENT}, "No such file or directory"},
		{"wrapped not exist", fmt.Errorf("ctx: %w", os.ErrNotExist), "No such file or directory"},
		{"permission", &fs.PathError{Op: "open", Path: "x", Err: syscall.EACCES}, "Permission denied"},
		{"exists", &os.LinkError{Op: "link", Old: "a", New: "b", Err: syscall.EEXIST}, "File exists"},
		// Go matches ENOTEMPTY against fs.ErrExist; it must not read as "File exists".
		{"not empty", &fs.PathError{Op: "remove", Path: "d", Err: syscall.ENOTEMPTY}, "Directory not empty"},
		{"other errno is capitalized", &fs.PathError{Op: "read", Path: "d", Err: errors.New("is a directory")}, "Is a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := core.ErrText(tt.err); got != tt.want {
				t.Fatalf("ErrText = %q, want %q", got, tt.want)
			}
		})
	}
}

type failWriter struct{ err error }

func (f failWriter) Write([]byte) (int, error) { return 0, f.err }

func TestAppendInt(t *testing.T) {
	tests := []struct {
		n     int64
		width int
		want  string
	}{
		{1, 6, "     1"},
		{123456, 6, "123456"},
		{1234567, 6, "1234567"},
		{-42, 6, "   -42"},
		{0, 0, "0"},
		{7, -3, "7"},
		{math.MinInt64, 0, "-9223372036854775808"},
	}
	for _, tt := range tests {
		if got := string(core.AppendInt([]byte("x"), tt.n, tt.width)); got != "x"+tt.want {
			t.Errorf("AppendInt(%d, %d) = %q, want %q", tt.n, tt.width, got, "x"+tt.want)
		}
		if want := fmt.Sprintf("%*d", max(tt.width, 0), tt.n); tt.want != want {
			t.Errorf("AppendInt(%d, %d): the expectation %q is not what %%*d prints, %q", tt.n, tt.width, tt.want, want)
		}
	}
}

func TestBuffersUseBufferSize(t *testing.T) {
	if got := core.NewReader(bytes.NewReader(nil)).Size(); got != core.BufferSize {
		t.Errorf("NewReader buffers %d bytes, want %d", got, core.BufferSize)
	}
	if got := core.NewWriter(&bytes.Buffer{}).Size(); got != core.BufferSize {
		t.Errorf("NewWriter buffers %d bytes, want %d", got, core.BufferSize)
	}
}

func TestFlushOutput(t *testing.T) {
	var errb bytes.Buffer
	d := core.NewDiag("x", core.Streams{Err: &errb})
	w := bufio.NewWriter(failWriter{syscall.ENOSPC})
	_, _ = w.WriteString("data")
	var err error
	core.FlushOutput(w, d, &err)
	if !errors.Is(err, core.ErrFailed) || errb.String() != "x: write error: No space left on device\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}

	errb.Reset()
	w = bufio.NewWriter(failWriter{syscall.EPIPE})
	_, _ = w.WriteString("data")
	err = nil
	core.FlushOutput(w, d, &err)
	if err != nil || errb.Len() != 0 {
		t.Fatalf("closed pipe: err = %v, stderr = %q", err, errb.String())
	}
}
