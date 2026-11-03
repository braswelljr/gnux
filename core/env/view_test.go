package env_test

import (
	"testing"

	"github.com/braswelljr/gnux/core/env"
)

func TestColoredListing(t *testing.T) {
	t.Setenv("GNUX_THEME", "")
	opts := env.Options{IgnoreEnvironment: true, Pretty: true, Color: true}
	r := run(opts, nil, "", "GOPATH=/go:/src", "A=b=c")
	want := "\x1b[1;34mGOPATH\x1b[0m\x1b[90m=\x1b[0m/go\x1b[90m:\x1b[0m/src\n" +
		"\x1b[1;34mA\x1b[0m\x1b[90m=\x1b[0mb=c\n"
	if r.stdout != want {
		t.Fatalf("got  %q\nwant %q", r.stdout, want)
	}
	opts.Null = true
	if r = run(opts, nil, "", "A=1"); r.stdout != "A=1\x00" {
		t.Fatalf("-0: %q", r.stdout)
	}
}
