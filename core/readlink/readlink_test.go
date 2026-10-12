package readlink_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/readlink"
)

// tree builds, in a symlink-free temporary directory:
//
//	dir/file
//	link     -> dir
//	flink    -> dir/file
//	rel      -> link/file
//	dangling -> nowhere
//	loop     -> loop
func tree(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dir", "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	links := map[string]string{
		"link":     "dir",
		"flink":    filepath.Join(root, "dir", "file"),
		"rel":      filepath.Join("link", "file"),
		"dangling": "nowhere",
		"loop":     "loop",
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Skipf("symbolic links unavailable: %v", err)
		}
	}
	return root
}

func run(t *testing.T, root string, opts readlink.Options, args ...string) (string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	r := readlink.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	r.Getwd = func() (string, error) { return root, nil }
	err := r.Run(args)
	return out.String(), errb.String(), err
}

func TestLinkValue(t *testing.T) {
	root := tree(t)
	out, errb, err := run(t, root, readlink.Options{}, filepath.Join(root, "link"), filepath.Join(root, "rel"))
	if err != nil || errb != "" {
		t.Fatalf("err = %v, stderr %q", err, errb)
	}
	if want := "dir\n" + filepath.Join("link", "file") + "\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestNotALinkIsQuietByDefault(t *testing.T) {
	root := tree(t)
	out, errb, err := run(t, root, readlink.Options{}, filepath.Join(root, "dir"), filepath.Join(root, "link"))
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if errb != "" {
		t.Fatalf("stderr = %q, want nothing without -v", errb)
	}
	if out != "dir\n" {
		t.Fatalf("stdout = %q: the remaining operand must still be printed", out)
	}
}

func TestVerboseReports(t *testing.T) {
	root := tree(t)
	missing := filepath.Join(root, "missing")
	_, errb, err := run(t, root, readlink.Options{Verbose: true}, missing)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "readlinkx: " + missing + ": No such file or directory\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

func TestCanonicalize(t *testing.T) {
	root := tree(t)
	file := filepath.Join(root, "dir", "file")
	tests := []struct {
		name string
		mode readlink.Mode
		arg  string
		want string // "" means failure
	}{
		{"follow a link to a directory", readlink.ModeFollow, "link", filepath.Join(root, "dir")},
		{"follow through two links", readlink.ModeFollow, "rel", file},
		{"follow an absolute link", readlink.ModeFollow, "flink", file},
		{"follow resolves dot-dot after links", readlink.ModeFollow, filepath.Join("link", "..", "dir", "file"), file},
		{"follow allows a missing last component", readlink.ModeFollow, filepath.Join("dir", "new"), filepath.Join(root, "dir", "new")},
		{"follow refuses a missing middle component", readlink.ModeFollow, filepath.Join("gone", "new"), ""},
		{"follow through a dangling link", readlink.ModeFollow, "dangling", filepath.Join(root, "nowhere")},
		{"follow refuses a file used as a directory", readlink.ModeFollow, filepath.Join("dir", "file", "x"), ""},
		{"existing requires the last component", readlink.ModeExisting, filepath.Join("dir", "new"), ""},
		{"existing refuses a dangling link", readlink.ModeExisting, "dangling", ""},
		{"existing resolves an existing name", readlink.ModeExisting, "rel", file},
		{"missing needs nothing", readlink.ModeMissing, filepath.Join("a", "b", "..", "c"), filepath.Join(root, "a", "c")},
		{"loop fails", readlink.ModeFollow, "loop", ""},
		{"dot components vanish", readlink.ModeFollow, filepath.Join(".", "dir", ".", "file"), file},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(t, root, readlink.Options{Mode: tt.mode}, tt.arg)
			if tt.want == "" {
				if !errors.Is(err, core.ErrFailed) || out != "" {
					t.Fatalf("out = %q, err = %v, want a failure", out, err)
				}
				return
			}
			if err != nil || out != tt.want+"\n" {
				t.Fatalf("out = %q, err = %v, want %q", out, err, tt.want)
			}
		})
	}
}

func TestLoopMessage(t *testing.T) {
	root := tree(t)
	_, errb, _ := run(t, root, readlink.Options{Mode: readlink.ModeFollow, Verbose: true}, "loop")
	if want := "readlinkx: loop: Too many levels of symbolic links\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	_, errb, _ = run(t, root, readlink.Options{Mode: readlink.ModeFollow, Verbose: true}, filepath.Join("dir", "file", "x"))
	if !strings.HasSuffix(errb, ": Not a directory\n") {
		t.Fatalf("stderr = %q, want Not a directory", errb)
	}
}

func TestDelimiters(t *testing.T) {
	root := tree(t)
	out, _, _ := run(t, root, readlink.Options{NoNewline: true}, filepath.Join(root, "link"))
	if out != "dir" {
		t.Fatalf("-n: stdout = %q", out)
	}
	out, _, _ = run(t, root, readlink.Options{Zero: true}, filepath.Join(root, "link"), filepath.Join(root, "link"))
	if out != "dir\x00dir\x00" {
		t.Fatalf("-z: stdout = %q", out)
	}
	out, errb, _ := run(t, root, readlink.Options{NoNewline: true}, filepath.Join(root, "link"), filepath.Join(root, "link"))
	if out != "dir\ndir\n" || errb != "readlinkx: ignoring --no-newline with multiple arguments\n" {
		t.Fatalf("-n with two operands: stdout %q, stderr %q", out, errb)
	}
}

func TestMissingOperand(t *testing.T) {
	_, _, err := run(t, t.TempDir(), readlink.Options{})
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "missing operand" {
		t.Fatalf("err = %v, want usage error 'missing operand'", err)
	}
}

func TestEmptyName(t *testing.T) {
	_, errb, err := run(t, t.TempDir(), readlink.Options{Mode: readlink.ModeMissing, Verbose: true}, "")
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(errb, "No such file or directory") {
		t.Fatalf("err = %v, stderr %q", err, errb)
	}
}
