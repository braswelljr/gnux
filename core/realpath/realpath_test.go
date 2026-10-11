package realpath_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/realpath"
)

// fixture is a small tree inside a symlink-free temporary directory:
//
//	root/
//	  dir/file
//	  dir/sub/
//	  other/
//	  link     -> dir          (when symlinks are available)
//	  filelink -> dir/file
//	  dangling -> nowhere
//	  loop     -> loop
type fixture struct {
	root  string
	links bool
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{filepath.Join("dir", "sub"), "other"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "dir", "file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	f := fixture{root: root, links: true}
	for name, target := range map[string]string{
		"link":     "dir",
		"filelink": filepath.Join("dir", "file"),
		"dangling": "nowhere",
		"loop":     "loop",
	} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			f.links = false
			break
		}
	}
	return f
}

func (f fixture) path(elem ...string) string {
	return filepath.Join(append([]string{f.root}, elem...)...)
}

func (f fixture) needLinks(t *testing.T) {
	t.Helper()
	if !f.links {
		t.Skip("symlinks unavailable")
	}
}

// run executes the engine with f.root as the working directory.
func (f fixture) run(opts realpath.Options, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	r := realpath.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	r.Getwd = func() (string, error) { return f.root, nil }
	err = r.Run(args)
	return out.String(), errb.String(), err
}

// rel joins path elements with the native separator. Unlike filepath.Join it
// does not clean the result, so "." and ".." reach the engine as written.
func rel(elem ...string) string {
	return strings.Join(elem, sep)
}

const sep = string(filepath.Separator)

func TestResolve(t *testing.T) {
	f := newFixture(t)

	tests := []struct {
		name  string
		opts  realpath.Options
		arg   string
		want  string // relative to the fixture root
		links bool
	}{
		{"relative file", realpath.Options{}, rel("dir", "file"), rel("dir", "file"), false},
		{"absolute file", realpath.Options{}, f.path("dir", "file"), rel("dir", "file"), false},
		{"dot", realpath.Options{}, ".", "", false},
		{"dot components", realpath.Options{}, rel("dir", ".", "sub", "..", "file"), rel("dir", "file"), false},
		{"repeated separators", realpath.Options{}, "dir" + sep + sep + "file", rel("dir", "file"), false},
		{"trailing separator on a directory", realpath.Options{}, "dir" + sep, "dir", false},
		{"missing last component", realpath.Options{}, rel("dir", "new"), rel("dir", "new"), false},
		{"existing with -e", realpath.Options{Mode: realpath.ModeExisting}, rel("dir", "file"), rel("dir", "file"), false},
		{"missing tree with -m", realpath.Options{Mode: realpath.ModeMissing}, rel("no", "such", "file"), rel("no", "such", "file"), false},
		{"dotdot below a missing name with -m", realpath.Options{Mode: realpath.ModeMissing}, rel("no", "such", "..", "x"), rel("no", "x"), false},
		{"below a file with -m", realpath.Options{Mode: realpath.ModeMissing}, rel("dir", "file", "x"), rel("dir", "file", "x"), false},

		{"link to directory", realpath.Options{}, "link", "dir", true},
		{"through a link", realpath.Options{}, rel("link", "file"), rel("dir", "file"), true},
		{"link to file", realpath.Options{}, "filelink", rel("dir", "file"), true},
		{"dangling link names its target", realpath.Options{}, "dangling", "nowhere", true},
		{"physical dotdot follows the link", realpath.Options{}, rel("link", "sub", "..", "..", "other"), "other", true},
		{"-P link then dotdot", realpath.Options{}, rel("link", ".."), "", true},
		{"-L collapses dotdot first", realpath.Options{Logical: true}, rel("link", "..", "other"), "other", true},
		{"-L still resolves remaining links", realpath.Options{Logical: true}, rel("other", "..", "link", "file"), rel("dir", "file"), true},
		{"-s keeps the link", realpath.Options{Strip: true}, rel("link", "file"), rel("link", "file"), true},
		{"-s keeps a dangling link", realpath.Options{Strip: true}, "dangling", "dangling", true},
		{"-s collapses dotdot lexically", realpath.Options{Strip: true}, rel("link", "..", "other"), "other", true},
		{"-s -m on a missing tree", realpath.Options{Strip: true, Mode: realpath.ModeMissing}, rel("a", "b", "..", "c"), rel("a", "c"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.links {
				f.needLinks(t)
			}
			out, errb, err := f.run(tt.opts, tt.arg)
			if err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb)
			}
			if want := f.path(tt.want) + "\n"; out != want {
				t.Fatalf("stdout = %q, want %q", out, want)
			}
		})
	}
}

func TestResolveErrors(t *testing.T) {
	f := newFixture(t)

	tests := []struct {
		name  string
		opts  realpath.Options
		arg   string
		want  string // the message after "realpathx: <arg>: "
		links bool
	}{
		{"missing parent", realpath.Options{}, rel("no", "file"), "No such file or directory", false},
		{"missing with -e", realpath.Options{Mode: realpath.ModeExisting}, rel("dir", "new"), "No such file or directory", false},
		{"trailing separator on a file", realpath.Options{}, rel("dir", "file") + sep, "Not a directory", false},
		{"dotdot after a file", realpath.Options{}, rel("dir", "file", ".."), "Not a directory", false},
		{"-s missing parent", realpath.Options{Strip: true}, rel("no", "file"), "No such file or directory", false},
		{"-s -e missing", realpath.Options{Strip: true, Mode: realpath.ModeExisting}, "new", "No such file or directory", false},
		{"dangling link with -e", realpath.Options{Mode: realpath.ModeExisting}, "dangling", "No such file or directory", true},
		{"link loop", realpath.Options{}, "loop", "Too many levels of symbolic links", true},
		{"-L trailing separator on a file link", realpath.Options{Logical: true}, "filelink" + sep, "Not a directory", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.links {
				f.needLinks(t)
			}
			out, errb, err := f.run(tt.opts, tt.arg)
			if !errors.Is(err, core.ErrFailed) {
				t.Fatalf("err = %v, want ErrFailed (stdout: %q)", err, out)
			}
			if want := "realpathx: " + tt.arg + ": " + tt.want + "\n"; errb != want {
				t.Fatalf("stderr = %q, want %q", errb, want)
			}
			if out != "" {
				t.Fatalf("stdout = %q, want empty", out)
			}
		})
	}
}

func TestEmptyOperand(t *testing.T) {
	f := newFixture(t)
	_, errb, err := f.run(realpath.Options{}, "")
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "realpathx: '': No such file or directory\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

func TestQuietAndPartialFailure(t *testing.T) {
	f := newFixture(t)
	bad := rel("no", "file")

	out, errb, err := f.run(realpath.Options{Quiet: true}, bad, "dir")
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if errb != "" {
		t.Fatalf("stderr = %q, want empty under -q", errb)
	}
	// The failure is skipped; the remaining operands are still printed.
	if want := f.path("dir") + "\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestZero(t *testing.T) {
	f := newFixture(t)
	out, errb, err := f.run(realpath.Options{Zero: true}, "dir", "other")
	if err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb)
	}
	if want := f.path("dir") + "\x00" + f.path("other") + "\x00"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestRelative(t *testing.T) {
	f := newFixture(t)

	tests := []struct {
		name string
		opts realpath.Options
		arg  string
		want string // literal expected output; "" means the absolute name of arg
	}{
		{"relative-to a sibling", realpath.Options{RelativeTo: "other"}, rel("dir", "file"), rel("..", "dir", "file")},
		{"relative-to an ancestor", realpath.Options{RelativeTo: "dir"}, rel("dir", "sub"), "sub"},
		{"relative-to itself", realpath.Options{RelativeTo: "dir"}, "dir", "."},
		{"relative-to a missing directory", realpath.Options{RelativeTo: "new"}, "dir", rel("..", "dir")},
		{"relative-base, inside", realpath.Options{RelativeBase: "dir"}, rel("dir", "sub"), "sub"},
		{"relative-base, outside stays absolute", realpath.Options{RelativeBase: "dir"}, "other", ""},
		{"relative-base is not a string prefix test", realpath.Options{RelativeBase: "dir"}, "dir2", ""},
		{"both, inside the base", realpath.Options{RelativeTo: rel("dir", "sub"), RelativeBase: "dir"}, rel("dir", "file"), rel("..", "file")},
		{"both, outside the base", realpath.Options{RelativeTo: rel("dir", "sub"), RelativeBase: "dir"}, "other", ""},
		{"relative-to outside the base disables both", realpath.Options{RelativeTo: "other", RelativeBase: "dir"}, rel("dir", "file"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := f.run(tt.opts, tt.arg)
			if err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb)
			}
			want := tt.want
			if want == "" {
				want = f.path(tt.arg)
			}
			if out != want+"\n" {
				t.Fatalf("stdout = %q, want %q", out, want+"\n")
			}
		})
	}
}

func TestRelativeDirErrors(t *testing.T) {
	f := newFixture(t)

	tests := []struct {
		name string
		opts realpath.Options
		want string
	}{
		{
			"unresolvable relative-to",
			realpath.Options{RelativeTo: rel("no", "dir")},
			"realpathx: " + rel("no", "dir") + ": No such file or directory\n",
		},
		{
			"relative-to must be a directory with -e",
			realpath.Options{Mode: realpath.ModeExisting, RelativeTo: rel("dir", "file")},
			"realpathx: " + rel("dir", "file") + ": Not a directory\n",
		},
		{
			// A bad directory is fatal, so -q does not hide it.
			"unresolvable relative-base under -q",
			realpath.Options{Quiet: true, RelativeBase: rel("no", "dir")},
			"realpathx: " + rel("no", "dir") + ": No such file or directory\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := f.run(tt.opts, "dir")
			if !errors.Is(err, core.ErrFailed) {
				t.Fatalf("err = %v, want ErrFailed", err)
			}
			if errb != tt.want {
				t.Fatalf("stderr = %q, want %q", errb, tt.want)
			}
			if out != "" {
				t.Fatalf("stdout = %q, want empty", out)
			}
		})
	}
}

func TestMissingOperand(t *testing.T) {
	f := newFixture(t)
	_, _, err := f.run(realpath.Options{})
	var uerr *core.UsageError
	if !errors.As(err, &uerr) || uerr.Msg != "missing operand" {
		t.Fatalf("err = %v, want usage error %q", err, "missing operand")
	}
}

func TestGetwdFailure(t *testing.T) {
	var out, errb bytes.Buffer
	r := realpath.New(realpath.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	r.Getwd = func() (string, error) { return "", os.ErrPermission }
	if err := r.Run([]string{"x"}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "realpathx: x: Permission denied\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}
