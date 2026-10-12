package unlinkx

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/unlink"
)

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != unlink.Name || !strings.HasPrefix(m.Use, unlink.Name+" ") {
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

// execute runs the tool through its cobra command, as the binary does.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs(args)
	err := c.Execute()
	return errb.String(), err
}

func TestCommand(t *testing.T) {
	f := filepath.Join(t.TempDir(), "-f")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if errb, err := execute(t, "--", f); err != nil {
		t.Fatalf("Execute: %v (stderr %q)", err, errb)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatalf("still there: %v", err)
	}
}
