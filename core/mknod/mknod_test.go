package mknod_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mknod"
)

type call struct {
	kind         string
	perm         uint32
	block        bool
	major, minor uint32
}

func newEngine(opts mknod.Options) (*mknod.Mknod, *bytes.Buffer, *[]call) {
	var errb bytes.Buffer
	m := mknod.New(opts, core.Streams{In: strings.NewReader(""), Out: &errb, Err: &errb})
	calls := &[]call{}
	m.Umask = func() uint32 { return 0o022 }
	m.MakeFifo = func(_ string, perm uint32) error {
		*calls = append(*calls, call{kind: "fifo", perm: perm})
		return nil
	}
	m.MakeDevice = func(name string, perm uint32, block bool, major, minor uint32) error {
		if name == "denied" {
			return os.ErrPermission
		}
		*calls = append(*calls, call{kind: "dev", perm: perm, block: block, major: major, minor: minor})
		return nil
	}
	return m, &errb, calls
}

func TestTypes(t *testing.T) {
	tests := []struct {
		args []string
		want call
	}{
		{[]string{"p", "p"}, call{kind: "fifo", perm: 0o666}},
		{[]string{"pipe", "pipe"}, call{kind: "fifo", perm: 0o666}},
		{[]string{"null", "c", "1", "3"}, call{kind: "dev", perm: 0o666, major: 1, minor: 3}},
		{[]string{"tape", "unbuffered", "0x10", "010"}, call{kind: "dev", perm: 0o666, major: 16, minor: 8}},
		{[]string{"loop", "block", "7", "0"}, call{kind: "dev", perm: 0o666, block: true, major: 7}},
	}
	for _, tt := range tests {
		m, errb, calls := newEngine(mknod.Options{})
		if err := m.Run(tt.args); err != nil || len(*calls) != 1 || (*calls)[0] != tt.want {
			t.Errorf("%v: err=%v stderr=%q calls=%+v", tt.args, err, errb, *calls)
		}
	}
}

func TestUsage(t *testing.T) {
	tests := []struct {
		args []string
		msg  string
	}{
		{nil, "missing operand"},
		{[]string{"x"}, "missing operand after 'x'"},
		{[]string{"x", "c"}, "missing operand after 'c'\nSpecial files require major and minor device numbers."},
		{[]string{"x", "c", "1"}, "missing operand after '1'"},
		{[]string{"x", "p", "1", "3"}, "extra operand '1'\nFifos do not have major and minor device numbers."},
		{[]string{"x", "p", "1"}, "extra operand '1'"},
		{[]string{"x", "c", "1", "3", "y"}, "extra operand 'y'"},
		{[]string{"x", "z", "1", "3"}, "invalid device type 'z'"},
	}
	for _, tt := range tests {
		m, _, calls := newEngine(mknod.Options{})
		var usage *core.UsageError
		if err := m.Run(tt.args); !errors.As(err, &usage) || usage.Msg != tt.msg || len(*calls) != 0 {
			t.Errorf("%v: err = %v", tt.args, err)
		}
	}
}

func TestFailures(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"x", "c", "z", "3"}, "mknodx: invalid major device number 'z'\n"},
		{[]string{"x", "c", "1", "-3"}, "mknodx: invalid minor device number '-3'\n"},
		{[]string{"x", "c", "4294967296", "0"}, "mknodx: invalid major device number '4294967296'\n"},
		{[]string{"denied", "c", "1", "3"}, "mknodx: denied: Permission denied\n"},
	}
	for _, tt := range tests {
		m, errb, _ := newEngine(mknod.Options{})
		if err := m.Run(tt.args); !errors.Is(err, core.ErrFailed) || errb.String() != tt.want {
			t.Errorf("%v: err=%v stderr=%q, want %q", tt.args, err, errb, tt.want)
		}
	}
}
