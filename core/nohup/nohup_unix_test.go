//go:build unix

package nohup_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nohup"
)

// TestHangupIgnored checks that the command survives a HUP it sends itself.
func TestHangupIgnored(t *testing.T) {
	_, errb, status := run(t, none, "", self, "hup-self")
	if status != 0 {
		t.Fatalf("status = %d (stderr %q): the command did not survive HUP", status, errb)
	}
}

// TestOutFileMode checks that nohup.out is private to its owner.
func TestOutFileMode(t *testing.T) {
	t.Chdir(t.TempDir())
	var out bytes.Buffer
	n := nohup.New(nohup.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	n.IsTerminal = func(s any) bool { return s == any(&out) }
	if err := n.Run([]string{self, "echo", "x"}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(nohup.OutFile); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("nohup.out: %v, %v", info, err)
	}
}
