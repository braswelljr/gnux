package uptimex

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uptime"
)

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"a", "b"})
	err := c.Execute()
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'b'" {
		t.Fatalf("Execute: %v", err)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != uptime.Name || !strings.HasPrefix(m.Use, uptime.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if n := len(m.Examples); n < 10 {
		t.Fatalf("%d examples, want at least 10", n)
	}
}
