package archx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/arch"
)

func TestCommand(t *testing.T) {
	out, err := command(t, "")
	if err != nil || len(out) < 2 || !strings.HasSuffix(out, "\n") {
		t.Fatalf("out = %q, err = %v", out, err)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != arch.Name || !strings.HasPrefix(m.Use, arch.Name+" ") {
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
func command(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), err
}
