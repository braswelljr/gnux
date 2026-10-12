package cmd

import (
	"strings"
	"testing"
)

// TestCompletion checks that gnux completion prints a script for every shell
// the release archives carry one for, and that the overview advertises it.
func TestCompletion(t *testing.T) {
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(sh, func(t *testing.T) {
			status, out, errb := run(t, "gnux", "completion", sh)
			if status != 0 {
				t.Fatalf("status = %d, stderr %q", status, errb)
			}
			if !strings.Contains(out, "gnux") || len(out) < 500 {
				t.Fatalf("completion %s printed %d bytes, not a script for gnux", sh, len(out))
			}
		})
	}

	_, out, _ := run(t, "gnux", "--help")
	if !strings.Contains(out, "completion") {
		t.Fatal("gnux --help does not list the completion command")
	}
}
