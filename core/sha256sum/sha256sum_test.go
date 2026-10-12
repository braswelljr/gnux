package sha256sum_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sha256sum"
)

const hello = "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"

// TestDigestAndCheck runs the tool's algorithm both ways: print the digest of
// standard input, then verify a line carrying it.
func TestDigestAndCheck(t *testing.T) {
	var out, errb bytes.Buffer
	s := core.Streams{In: strings.NewReader("hello\n"), Out: &out, Err: &errb}
	if err := sha256sum.New(sha256sum.Options{}, s).Run(nil); err != nil {
		t.Fatalf("Run: %v (stderr %q)", err, errb.String())
	}
	if want := hello + "  -\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}

	out.Reset()
	s.In = strings.NewReader(hello + "  -\n")
	err := sha256sum.New(sha256sum.Options{Check: true}, s).Run(nil)
	// The list is standard input, so the "-" it names reads as empty input.
	if err == nil || !strings.Contains(out.String(), "-: FAILED") {
		t.Fatalf("check of a stale line: err=%v stdout=%q", err, out.String())
	}
	if !strings.HasPrefix(errb.String(), sha256sum.Name+": ") {
		t.Fatalf("diagnostics are not prefixed with %s: %q", sha256sum.Name, errb.String())
	}
}
