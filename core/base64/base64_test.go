package base64_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/base64"
)

// TestRoundTrip encodes and decodes with the tool's alphabet.
func TestRoundTrip(t *testing.T) {
	var enc, dec, errb bytes.Buffer
	if err := base64.New(base64.Options{}, core.Streams{In: strings.NewReader("hello world"), Out: &enc, Err: &errb}).Run(nil); err != nil {
		t.Fatal(err)
	}
	if err := base64.New(base64.Options{Decode: true}, core.Streams{In: &enc, Out: &dec, Err: &errb}).Run(nil); err != nil {
		t.Fatal(err)
	}
	if dec.String() != "hello world" {
		t.Fatalf("round trip = %q", dec.String())
	}
	err := base64.New(base64.Options{Decode: true}, core.Streams{In: strings.NewReader("!"), Out: &dec, Err: &errb}).Run(nil)
	if err == nil || errb.String() != base64.Name+": invalid input\n" {
		t.Fatalf("invalid input: err=%v stderr=%q", err, errb.String())
	}
}
