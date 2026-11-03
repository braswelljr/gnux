package kill_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/kill"
)

func TestViewTable(t *testing.T) {
	list := func(opts kill.Options, args ...string) string {
		var out bytes.Buffer
		k := kill.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
		k.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
		if err := k.Run(args); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	// -l and -L alike: every signal a row, under a header.
	for _, mode := range []kill.Mode{kill.ModeList, kill.ModeTable} {
		got := list(kill.Options{Mode: mode, Pretty: true})
		lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
		if len(lines) != len(kill.Signals())+4 || !strings.Contains(lines[1], "| No. | Name ") ||
			!strings.Contains(got, "|   9 | KILL ") {
			t.Fatalf("mode %v:\n%s", mode, got)
		}
	}
	// Converting an operand is GNU's.
	if got := list(kill.Options{Mode: kill.ModeList, Pretty: true}, "9"); got != "KILL\n" {
		t.Fatalf("-l 9: %q", got)
	}
	// KILL, which no program can catch, in the alert color.
	if got := list(kill.Options{Mode: kill.ModeTable, Pretty: true, Color: true}); !strings.Contains(got, "\x1b[") || !strings.Contains(got, "KILL\x1b[0m") {
		t.Fatalf("no colors:\n%s", got)
	}
}
