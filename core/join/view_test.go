package join_test

import (
	"testing"

	"github.com/braswelljr/gnux/core/join"
)

func TestView(t *testing.T) {
	t.Setenv("GNUX_THEME", "")
	dir := t.TempDir()
	a := write(t, dir, "a", "id name\n1 ada\n2 lin\n")
	b := write(t, dir, "b", "id team\n1 web\n3 ops\n")
	c := func(code, s string) string { return "\x1b[" + code + "m" + s + "\x1b[0m" }

	opts := join.Options{Pretty: true, Color: true, Header: true, Unpaired1: true, Unpaired2: true}
	out, _, err := run(opts, "", a, b)
	want := c("4", "id") + " " + c("4", "name") + " " + c("4", "team") + "\n" +
		c("1", "1") + " " + c("36", "ada") + " " + c("35", "web") + "\n" +
		c("36", "2") + " " + c("36", "lin") + "\n" +
		c("35", "3") + " " + c("35", "ops") + "\n"
	if err != nil || out != want {
		t.Fatalf("got  %q (%v)\nwant %q", out, err, want)
	}

	// -o names each field's file; the text stays GNU's without Color.
	opts = join.Options{Pretty: true, Color: true, Format: []join.Field{{File: 2, Field: 2}, {File: 0}}}
	if out, _, _ = run(opts, "", a, b); out != c("35", "team")+" "+c("1", "id")+"\n"+c("35", "web")+" "+c("1", "1")+"\n" {
		t.Fatalf("-o: %q", out)
	}
	if out, _, _ = run(join.Options{Pretty: true}, "", a, b); out != "id name team\n1 ada web\n" {
		t.Fatalf("uncolored: %q", out)
	}
}
