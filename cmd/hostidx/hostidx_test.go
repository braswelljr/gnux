package hostidx

import (
	"regexp"
	"testing"
)

func TestCommand(t *testing.T) {
	out, errText, status := execute(t, Spec(), "")
	if status != 0 || errText != "" || !regexp.MustCompile(`^[0-9a-f]{8}\n$`).MatchString(out) {
		t.Fatalf("out=%q stderr=%q status=%d; want eight hex digits", out, errText, status)
	}
	again, _, _ := execute(t, Spec(), "")
	if again != out {
		t.Fatalf("hostid changed between runs: %q then %q", out, again)
	}

	_, errText, status = execute(t, Spec(), "", "extra")
	if status != 1 || errText == "" {
		t.Fatalf("an operand: status=%d stderr=%q, want a usage error", status, errText)
	}
}
