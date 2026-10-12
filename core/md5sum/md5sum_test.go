package md5sum_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/md5sum"
)

const hello = "b1946ac92492d2347c6235b4d2611184"

// TestDigestAndCheck runs the tool's algorithm both ways: print the digest of
// standard input, then verify a line carrying it.
func TestDigestAndCheck(t *testing.T) {
	var out, errb bytes.Buffer
	s := core.Streams{In: strings.NewReader("hello\n"), Out: &out, Err: &errb}
	if err := md5sum.New(md5sum.Options{}, s).Run(nil); err != nil {
		t.Fatalf("Run: %v (stderr %q)", err, errb.String())
	}
	if want := hello + "  -\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}

	out.Reset()
	s.In = strings.NewReader(hello + "  -\n")
	err := md5sum.New(md5sum.Options{Check: true}, s).Run(nil)
	// The list is standard input, so the "-" it names reads as empty input.
	if err == nil || !strings.Contains(out.String(), "-: FAILED") {
		t.Fatalf("check of a stale line: err=%v stdout=%q", err, out.String())
	}
	if !strings.HasPrefix(errb.String(), md5sum.Name+": ") {
		t.Fatalf("diagnostics are not prefixed with %s: %q", md5sum.Name, errb.String())
	}
}
