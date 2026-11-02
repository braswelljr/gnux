package id_test

import (
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/id"
)

func TestPanel(t *testing.T) {
	// The panel is drawn with the locale's characters.
	t.Setenv("LC_ALL", "C.UTF-8")
	got, _, err := run(t, id.Options{Pretty: true}, self)
	if err != nil || !strings.HasPrefix(got, "╭─ ann ") ||
		!strings.Contains(got, "│ User    501  ann ") ||
		!strings.Contains(got, "│ Groups   20  staff ") ||
		!strings.Contains(got, "│          77 ") {
		t.Fatalf("got (%v)\n%s", err, got)
	}

	// A setuid process shows its effective IDs; -u stays GNU's.
	got, _, _ = run(t, id.Options{Pretty: true}, setuid)
	if !strings.Contains(got, "│ Effective user     0  root ") || !strings.Contains(got, "│ Effective group    0  wheel ") {
		t.Fatalf("setuid:\n%s", got)
	}
	if got, _, _ = run(t, id.Options{Pretty: true, User: true}, self); got != "501\n" {
		t.Fatalf("-u: %q", got)
	}
}
