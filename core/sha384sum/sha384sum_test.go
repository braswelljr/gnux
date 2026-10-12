package sha384sum_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sha384sum"
)

const hello = "1d0f284efe3edea4b9ca3bd514fa134b17eae361ccc7a1eefeff801b9bd6604e01f21f6bf249ef030599f0c218f2ba8c"

// TestDigestAndCheck runs the tool's algorithm both ways: print the digest of
// standard input, then verify a line carrying it.
func TestDigestAndCheck(t *testing.T) {
	var out, errb bytes.Buffer
	s := core.Streams{In: strings.NewReader("hello\n"), Out: &out, Err: &errb}
	if err := sha384sum.New(sha384sum.Options{}, s).Run(nil); err != nil {
		t.Fatalf("Run: %v (stderr %q)", err, errb.String())
	}
	if want := hello + "  -\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}

	out.Reset()
	s.In = strings.NewReader(hello + "  -\n")
	err := sha384sum.New(sha384sum.Options{Check: true}, s).Run(nil)
	// The list is standard input, so the "-" it names reads as empty input.
	if err == nil || !strings.Contains(out.String(), "-: FAILED") {
		t.Fatalf("check of a stale line: err=%v stdout=%q", err, out.String())
	}
	if !strings.HasPrefix(errb.String(), sha384sum.Name+": ") {
		t.Fatalf("diagnostics are not prefixed with %s: %q", sha384sum.Name, errb.String())
	}
}
