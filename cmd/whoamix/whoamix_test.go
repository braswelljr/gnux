package whoamix

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/whoami"
)

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != whoami.Name || !strings.HasPrefix(m.Use, whoami.Name+" ") {
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

// TestCommand runs the tool through its cobra command, as the binary does.
func TestCommand(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs(nil)
	if err := c.Execute(); err != nil {
		if errors.Is(err, core.ErrFailed) {
			t.Skipf("not available here: %s", errb.String())
		}
		t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
	}
	if !strings.HasSuffix(out.String(), "\n") || out.Len() < 2 {
		t.Fatalf("stdout = %q", out.String())
	}
}
