package find_test

import (
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/find"
)

// view sets the view's options and an environment with only what is given.
func view(opts find.Options, env map[string]string) func(*find.Find) {
	return func(f *find.Find) {
		f.Options = opts
		f.Getenv = func(k string) string { return env[k] }
	}
}

func TestTree(t *testing.T) {
	tree(t)
	r := run(t, view(find.Options{Tree: true}, nil), ".", "-name", "*.txt", "-o", "-name", "*.TXT")
	want := ".\n" +
		"├── a.txt\n" +
		"└── dir\n" +
		"    ├── c.txt\n" +
		"    └── sub\n" +
		"        └── d.TXT\n"
	if r.e != nil || r.out != want {
		t.Fatalf("got %q (%v)\nwant %q", r.out, r.e, want)
	}

	// -depth prints a directory after its contents; it is drawn above them.
	r = run(t, view(find.Options{Tree: true}, nil), "dir", "-depth")
	want = "dir\n├── c.txt\n└── sub\n    └── d.TXT\n"
	if r.out != want {
		t.Fatalf("-depth: got %q, want %q", r.out, want)
	}

	// A file printed twice is drawn once; ASCII without a UTF-8 locale;
	// with Pretty, tree(1)'s count of what matched.
	r = run(t, view(find.Options{Tree: true, Pretty: true}, map[string]string{"LANG": "C"}),
		"dir", "-print", "-print")
	want = "dir\n|-- c.txt\n`-- sub\n    `-- d.TXT\n\n1 directory, 2 files\n"
	if r.out != want {
		t.Fatalf("ASCII: got %q, want %q", r.out, want)
	}

	// Only -print to standard output is drawn; -print0 and -printf are
	// GNU's bytes, written as the walk goes.
	r = run(t, view(find.Options{Tree: true}, nil), "dir", "-maxdepth", "1", "-printf", "%f;", "-print0")
	if r.out != "dir;dir\x00c.txt;dir/c.txt\x00sub;dir/sub\x00" {
		t.Fatalf("-print0: got %q", r.out)
	}

	// Nothing matched: nothing drawn.
	if r = run(t, view(find.Options{Tree: true, Pretty: true}, nil), ".", "-name", "none"); r.out != "" {
		t.Fatalf("no match: got %q", r.out)
	}
}

func TestColoredNames(t *testing.T) {
	tree(t)
	env := map[string]string{"LS_COLORS": "*.log=35"}
	r := run(t, view(find.Options{Color: true, Pretty: true}, env), ".", "-maxdepth", "1", "-name", "[!.]*")
	dim := "\x1b[90m"
	want := strings.Join([]string{
		dim + "./\x1b[0ma.txt",
		dim + "./\x1b[0m\x1b[35mb.log\x1b[0m",
		dim + "./\x1b[0m\x1b[01;34mdir\x1b[0m",
		dim + "./\x1b[0m\x1b[01;34mempty\x1b[0m",
	}, "\n") + "\n"
	if r.out != want {
		t.Fatalf("got  %q\nwant %q", r.out, want)
	}

	// The text under the colors is GNU's.
	plain := run(t, nil, ".", "-maxdepth", "1", "-name", "[!.]*").out
	if got := strip(r.out); got != plain {
		t.Fatalf("text changed: %q, want %q", got, plain)
	}

	// A tree's directories on the way to a match are dimmed.
	r = run(t, view(find.Options{Color: true, Tree: true}, nil), ".", "-name", "d.TXT")
	if !strings.Contains(r.out, dim+"dir\x1b[0m\n") || !strings.HasSuffix(r.out, "d.TXT\n") {
		t.Fatalf("tree: got %q", r.out)
	}
}

// strip removes the SGR sequences from s.
func strip(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
