package printenv_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/printenv"
)

func list(t *testing.T, opts printenv.Options, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	p := printenv.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	p.Environ = func() []string {
		return []string{"HOME=/home/ada", "PATH=/usr/bin" + string(os.PathListSeparator) + "/bin", "EMPTY="}
	}
	p.Getenv = func(string) string { return "" }
	_ = p.Run(args)
	return out.String()
}

func TestColoredListing(t *testing.T) {
	name := func(s string) string { return "\x1b[1;34m" + s + "\x1b[0m" }
	dim := func(s string) string { return "\x1b[90m" + s + "\x1b[0m" }
	got := list(t, printenv.Options{Pretty: true, Color: true})
	want := name("HOME") + dim("=") + "/home/ada\n" +
		name("PATH") + dim("=") + "/usr/bin" + dim(string(os.PathListSeparator)) + "/bin\n" +
		name("EMPTY") + dim("=") + "\n"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	// A value asked for by name is data, and -0 is never colored.
	if got := list(t, printenv.Options{Pretty: true, Color: true}, "PATH"); got != "/usr/bin:/bin\n" {
		t.Fatalf("one value: %q", got)
	}
	if got := list(t, printenv.Options{Pretty: true, Color: true, Null: true}); got != "HOME=/home/ada\x00PATH=/usr/bin:/bin\x00EMPTY=\x00" {
		t.Fatalf("-0: %q", got)
	}
}
