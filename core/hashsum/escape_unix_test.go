//go:build unix

// A backslash is an ordinary file name character only on Unix; elsewhere it
// is the path separator, so these tests of its escaping run on Unix alone.

package hashsum_test

import (
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/hashsum"
)

func TestEscapedName(t *testing.T) {
	dir := t.TempDir()
	f := write(t, dir, "a\\b", "hello\n")
	out, _, err := run(hashsum.MD5, hashsum.Options{}, "", f)
	if err != nil {
		t.Skipf("cannot create a file with a backslash here: %v", err)
	}
	want := `\` + hello[hashsum.MD5] + "  " + strings.ReplaceAll(f, `\`, `\\`) + "\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	// -z disables escaping.
	out, _, _ = run(hashsum.MD5, hashsum.Options{Zero: true}, "", f)
	if !strings.HasSuffix(out, f+"\x00") {
		t.Fatalf("-z stdout = %q", out)
	}
}

func TestCheckEscapedName(t *testing.T) {
	dir := t.TempDir()
	f := write(t, dir, "a\nb", "hello\n")
	line := `\` + hello[hashsum.MD5] + "  " + strings.ReplaceAll(f, "\n", `\n`) + "\n"
	out, errb, err := run(hashsum.MD5, hashsum.Options{Check: true}, line)
	if err != nil {
		t.Skipf("cannot create a file with a newline here: %v %q", err, errb)
	}
	if want := `\` + strings.ReplaceAll(f, "\n", `\n`) + ": OK\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}
