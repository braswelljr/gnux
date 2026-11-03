package ddx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/tool"
)

// execute runs spec as a command on args and stdin, and returns its output
// and the exit status gnux would give it.
func execute(t *testing.T, spec tool.Spec, stdin string, args ...string) (out, errText string, status int) {
	t.Helper()
	var o, e bytes.Buffer
	c := spec.Command(core.Streams{In: strings.NewReader(stdin), Out: &o, Err: &e})
	// A nil slice would make cobra read the test binary's own arguments.
	c.SetArgs(append([]string{}, args...))
	err := c.Execute()
	var exit *core.ExitError
	var usage *core.UsageError
	switch {
	case err == nil:
	case errors.As(err, &exit):
		status = exit.Code
	case errors.As(err, &usage):
		// cmd.Main prints a usage error; the command itself does not.
		e.WriteString(c.Name() + ": " + usage.Error() + "\n")
		status = tool.UsageExit(c)
	default:
		status = 1
	}
	return o.String(), e.String(), status
}
