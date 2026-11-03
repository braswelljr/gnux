package stty

import (
	"strings"
	"testing"
)

func TestDisplayPretty(t *testing.T) {
	s, out, _ := newEngine()
	s.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
	s.Pretty, s.device = true, "standard input"
	s.maxCol = 60
	m := sane()
	m.lflag &^= c.ECHO
	s.displayPretty(&m)
	got := out.String()
	for _, want := range []string{
		"+- standard input -",
		"| Speed       " + speeds[len(speeds)-1].name + " baud",
		"| Characters  intr    ^C ",
		"min     1 ",
		"| Local       ",
		" -echo ",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	// Every line fits the screen, the frame included.
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if len(line) > 60 {
			t.Errorf("line longer than the screen: %q", line)
		}
	}
}
