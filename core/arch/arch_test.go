package arch_test

import (
	"bytes"
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/arch"
	"github.com/braswelljr/gnux/core/uname"
)

func run(info uname.Info, err error, args ...string) (stdout, stderr string, runErr error) {
	var out, errb bytes.Buffer
	a := arch.New(arch.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	a.System = func() (uname.Info, error) { return info, err }
	runErr = a.Run(args)
	return out.String(), errb.String(), runErr
}

func TestMachine(t *testing.T) {
	if out, _, err := run(uname.Info{Machine: "aarch64", KernelName: "Linux"}, nil); err != nil || out != "aarch64\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if out, _, err := run(uname.Info{}, nil); err != nil || out != "unknown\n" {
		t.Fatalf("empty machine: out = %q, err = %v", out, err)
	}
}

func TestMatchesUname(t *testing.T) {
	var a, u bytes.Buffer
	s := func(w *bytes.Buffer) core.Streams {
		return core.Streams{In: strings.NewReader(""), Out: w, Err: &bytes.Buffer{}}
	}
	if err := arch.New(arch.Options{}, s(&a)).Run(nil); err != nil {
		t.Fatal(err)
	}
	if err := uname.New(uname.Options{Machine: true}, s(&u)).Run(nil); err != nil {
		t.Fatal(err)
	}
	if a.String() != u.String() || a.Len() < 2 {
		t.Fatalf("archx = %q, unamex -m = %q", a.String(), u.String())
	}
}

func TestSystemFails(t *testing.T) {
	_, errOut, err := run(uname.Info{}, syscall.EPERM)
	if errOut != "archx: cannot get system name: Permission denied\n" || !errors.Is(err, core.ErrFailed) {
		t.Fatalf("stderr = %q, err = %v", errOut, err)
	}
}

func TestExtraOperand(t *testing.T) {
	_, _, err := run(uname.Info{}, nil, "x")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'x'" {
		t.Fatalf("err = %v", err)
	}
}
