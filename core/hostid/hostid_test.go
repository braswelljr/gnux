package hostid_test

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hostid"
)

func TestPrintsEightHexDigits(t *testing.T) {
	for id, want := range map[uint32]string{0: "00000000\n", 0xa8c002d7: "a8c002d7\n", 0x7f: "0000007f\n"} {
		var out bytes.Buffer
		h := hostid.New(hostid.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &out})
		h.ID = func() uint32 { return id }
		if err := h.Run(nil); err != nil || out.String() != want {
			t.Fatalf("id %x: err=%v out=%q, want %q", id, err, out.String(), want)
		}
	}
}

func TestExtraOperand(t *testing.T) {
	var out bytes.Buffer
	err := hostid.New(hostid.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &out}).Run([]string{"x"})
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'x'" || out.Len() != 0 {
		t.Fatalf("err=%v out=%q", err, out.String())
	}
}

// TestFromAddress checks glibc's derivation against a value GNU hostid
// printed for a host at 192.168.215.2 with no /etc/hostid.
func TestFromAddress(t *testing.T) {
	if got := hostid.FromAddress(net.ParseIP("192.168.215.2")); got != 0xa8c002d7 {
		t.Fatalf("FromAddress = %08x, want a8c002d7", got)
	}
	if got := hostid.FromAddress(net.ParseIP("::1")); got != 0 {
		t.Fatalf("an IPv6 address = %08x, want 0", got)
	}
}
