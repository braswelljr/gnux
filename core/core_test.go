package core_test

import (
	"errors"
	"fmt"
	"io/fs"
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
