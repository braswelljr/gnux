package vdirx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

const wantLines = 4 // total, then one line per entry

func TestCommand(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a b", "new\nline", "c"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Skipf("cannot create %q here: %v", name, err)
		}
	}
	out, _, err := command(t, dir)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out, "new\nline") || !strings.Contains(out, `new\nline`) || !strings.Contains(out, `a\ b`) {
		t.Fatalf("names are not escaped: %q", out)
	}
	if n := strings.Count(out, "\n"); n != wantLines {
		t.Fatalf("stdout = %q; want %d lines", out, wantLines)
	}

	// An option that picks the format or the quoting wins.
	out, _, _ = command(t, "-1", "-Q", dir)
	if want := "\"a b\"\n\"c\"\n\"new\\nline\"\n"; out != want {
		t.Fatalf("-1 -Q stdout = %q; want %q", out, want)
	}

	_, errOut, err := command(t, filepath.Join(dir, "missing"))
	var exit *core.ExitError
	if !errors.As(err, &exit) || exit.Code != 2 || !strings.HasPrefix(errOut, Name+": cannot access ") {
		t.Fatalf("err = %v, stderr = %q", err, errOut)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != Name || !strings.HasPrefix(m.Use, Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if m.Short == "" || m.Long == "" || m.Permissions == "" || len(m.UseCases) == 0 {
		t.Fatal("Meta is missing prose")
	}
	if n := len(m.Examples); n < 10 {
		t.Fatalf("%d examples, want at least 10", n)
	}
	for _, e := range m.Examples {
		if e.Description == "" || e.Bash == "" || e.PowerShell == "" {
			t.Fatalf("incomplete example %+v", e)
		}
	}
}

// command runs the tool through its cobra command, as the binary does.
func command(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), errb.String(), err
}
