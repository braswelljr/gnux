// Package hostid implements the engine behind hostidx, a GNU-compatible
// implementation of the GNU/UNIX hostid command: print the numeric
// identifier of the current host.
package hostid

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "hostidx"

// Options mirrors the GNU hostid flag surface, which has no options.
type Options struct{}

// Hostid is the engine. Construct it with New.
type Hostid struct {
	Options

	// ID returns the host identifier. It is the injection point tests use;
	// it defaults to the system's gethostid.
	ID func() uint32

	streams core.Streams
	diag    *core.Diag
}

// New builds a hostid engine writing to s.
func New(opts Options, s core.Streams) *Hostid {
	return &Hostid{Options: opts, ID: hostID, streams: s, diag: core.NewDiag(Name, s)}
}

// Run prints the identifier as eight hexadecimal digits. hostid takes no
// operands.
func (h *Hostid) Run(args []string) error {
	if len(args) > 0 {
		return core.Usagef("extra operand '%s'", args[0])
	}
	if _, err := fmt.Fprintf(h.streams.Out, "%08x\n", h.ID()); err != nil && !core.IsClosedPipe(err) {
		h.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return h.diag.Result()
}

// FromAddress derives a host identifier from an IPv4 address the way glibc
// does when no identifier is configured: the address as it lies in memory,
// with its two halves swapped.
func FromAddress(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	id := binary.NativeEndian.Uint32(ip4)
	return id<<16 | id>>16
}

// fromHostname is glibc's fallback: the identifier derived from the first
// IPv4 address the host name resolves to, or 0 when it resolves to none.
func fromHostname() uint32 {
	name, err := os.Hostname()
	if err != nil {
		return 0
	}
	ips, err := net.LookupIP(name)
	if err != nil {
		return 0
	}
	for _, ip := range ips {
		if ip.To4() != nil {
			return FromAddress(ip)
		}
	}
	return 0
}
