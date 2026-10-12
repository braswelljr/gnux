package uname_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uname"
)

var info = uname.Info{
	KernelName: "Linux", Nodename: "box", KernelRelease: "6.8.0", KernelVersion: "#1 SMP",
	Machine: "x86_64", Processor: "unknown", HardwarePlatform: "unknown", OperatingSystem: "GNU/Linux",
}

func run(opts uname.Options, sys uname.Info, args ...string) (string, error) {
	var out bytes.Buffer
	u := uname.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	u.System = func() (uname.Info, error) { return sys, nil }
	err := u.Run(args)
	return out.String(), err
}

func TestUname(t *testing.T) {
	tests := []struct {
		name string
		opts uname.Options
		want string
	}{
		{"default", uname.Options{}, "Linux\n"},
		{"-s", uname.Options{KernelName: true}, "Linux\n"},
		{"-a omits unknown -p -i", uname.Options{All: true}, "Linux box 6.8.0 #1 SMP x86_64 GNU/Linux\n"},
		{"-p prints unknown", uname.Options{Processor: true}, "unknown\n"},
		{"fixed order", uname.Options{Machine: true, KernelName: true, KernelRelease: true}, "Linux 6.8.0 x86_64\n"},
		{"-o", uname.Options{OperatingSystem: true}, "GNU/Linux\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := run(tt.opts, info)
			if err != nil || out != tt.want {
				t.Fatalf("out = %q, err = %v, want %q", out, err, tt.want)
			}
		})
	}

	known := info
	known.Processor, known.HardwarePlatform = "x86_64", "pc"
	if out, _ := run(uname.Options{All: true}, known); out != "Linux box 6.8.0 #1 SMP x86_64 x86_64 pc GNU/Linux\n" {
		t.Fatalf("-a with known -p -i: %q", out)
	}
	empty := uname.Info{}
	if out, _ := run(uname.Options{Nodename: true}, empty); out != "unknown\n" {
		t.Fatalf("empty field: %q", out)
	}
}

func TestExtraOperand(t *testing.T) {
	_, err := run(uname.Options{}, info, "x")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'x'" {
		t.Fatalf("err = %v", err)
	}
}

func TestRealSystem(t *testing.T) {
	var out bytes.Buffer
	if err := uname.New(uname.Options{All: true}, core.Streams{Out: &out, Err: &bytes.Buffer{}}).Run(nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("no output")
	}
}
