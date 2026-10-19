package lognamex

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/logname"
)

func TestCommand(t *testing.T) {
	out, errOut, err := command(t)
	if err != nil {
		if errors.Is(err, core.ErrFailed) {
			t.Skipf("not available here: %s", errOut)
		}
		t.Fatalf("Execute: %v (stderr %q)", err, errOut)
	}
	if out != "" && !strings.HasSuffix(out, "\n") {
		t.Fatalf("stdout = %q", out)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != logname.Name || !strings.HasPrefix(m.Use, logname.Name+" ") {
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
