package sha512sum_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sha512sum"
)

const hello = "e7c22b994c59d9cf2b48e549b1e24666636045930d3da7c1acb299d1c3b7f931f94aae41edda2c2b207a36e10f8bcb8d45223e54878f5b316e7ce3b6bc019629"

// TestDigestAndCheck runs the tool's algorithm both ways: print the digest of
// standard input, then verify a line carrying it.
func TestDigestAndCheck(t *testing.T) {
	var out, errb bytes.Buffer
	s := core.Streams{In: strings.NewReader("hello\n"), Out: &out, Err: &errb}
	if err := sha512sum.New(sha512sum.Options{}, s).Run(nil); err != nil {
		t.Fatalf("Run: %v (stderr %q)", err, errb.String())
	}
	if want := hello + "  -\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}

	out.Reset()
	s.In = strings.NewReader(hello + "  -\n")
	err := sha512sum.New(sha512sum.Options{Check: true}, s).Run(nil)
	// The list is standard input, so the "-" it names reads as empty input.
	if err == nil || !strings.Contains(out.String(), "-: FAILED") {
		t.Fatalf("check of a stale line: err=%v stdout=%q", err, out.String())
	}
	if !strings.HasPrefix(errb.String(), sha512sum.Name+": ") {
		t.Fatalf("diagnostics are not prefixed with %s: %q", sha512sum.Name, errb.String())
	}
}
