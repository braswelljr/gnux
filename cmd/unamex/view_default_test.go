package unamex

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

// TestViewFollowsTheDefault checks that unamex shows its view as every other
// tool does: from GNUX_PRETTY (or on a terminal), and never through a pipe,
// where $(unamex) keeps GNU's single line.
func TestViewFollowsTheDefault(t *testing.T) {
	run := func(args ...string) string {
		t.Helper()
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
		c.SetArgs(append([]string{}, args...))
		if err := c.Execute(); err != nil {
			t.Fatalf("%v: %v (%s)", args, err, errb.String())
		}
		return out.String()
	}
	args := []string{"-s"}
	t.Setenv("GNUX_PRETTY", "")
	t.Setenv("NO_COLOR", "1")
	if plain := run(args...); strings.Count(plain, "\n") != 1 {
		t.Fatalf("through a pipe the output is GNU's one line, got %q", plain)
	}
	t.Setenv("GNUX_PRETTY", "always")
	if view := run(args...); strings.Count(view, "\n") < 3 {
		t.Fatalf("GNUX_PRETTY=always shows the panel, got %q", view)
	}
}
