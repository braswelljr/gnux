package nohupx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nohup"
)

func TestCommand(t *testing.T) {
	var errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb})
	c.SetArgs([]string{"gnux-no-such-command", "--flag-for-the-command"})
	err := c.Execute()
	var exit *core.ExitError
	if !errors.As(err, &exit) || exit.Code != nohup.ExitNotFound {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != nohup.Name || !strings.HasPrefix(m.Use, nohup.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
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
