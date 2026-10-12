// Package utmp reads the record of who is logged in (the utmp or utmpx
// file) and reports the boot time and load average, for whox and uptimex.
// Each platform's file format is decoded without the C library.
package utmp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"
)

// ErrUnsupported is returned where the platform keeps no such records.
var ErrUnsupported = errors.New("operation not supported")

// Type is the kind of a record, in the Linux numbering.
type Type int

// The record types.
const (
	Empty Type = iota
	RunLevel
	BootTime
	NewTime
	OldTime
	InitProcess
	LoginProcess
	UserProcess
	DeadProcess
)

// Entry is one record.
type Entry struct {
	Type              Type
	PID               int
	Line              string // the terminal, without /dev/
	ID                string
	User              string
	Host              string
	Time              time.Time
	Termination, Exit int // of a dead process (Linux only)
}

func cstr(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

// DecodeLinux reads glibc's struct utmp records (384 bytes each, the same on
// every Linux architecture gnux supports).
func DecodeLinux(data []byte) []Entry {
	const size = 384
	le := binary.LittleEndian
	var out []Entry
	for ; len(data) >= size; data = data[size:] {
		r := data[:size]
		out = append(out, Entry{
			Type:        Type(int16(le.Uint16(r[0:]))),
			PID:         int(int32(le.Uint32(r[4:]))),
			Line:        cstr(r[8:40]),
			ID:          cstr(r[40:44]),
			User:        cstr(r[44:76]),
			Host:        cstr(r[76:332]),
			Termination: int(int16(le.Uint16(r[332:]))),
			Exit:        int(int16(le.Uint16(r[334:]))),
			Time:        time.Unix(int64(int32(le.Uint32(r[340:]))), int64(int32(le.Uint32(r[344:])))*1000),
		})
	}
	return out
}

// darwinTypes maps macOS utmpx types onto the Linux numbering; the
// signature, accounting and shutdown records have no counterpart.
var darwinTypes = map[int16]Type{
	0: Empty, 1: RunLevel, 2: BootTime, 3: OldTime, 4: NewTime, 5: InitProcess,
	6: LoginProcess, 7: UserProcess, 8: DeadProcess,
}

// DecodeDarwin reads the records of macOS's /var/run/utmpx (628 bytes each:
// user[256], id[4], line[32], pid, type, a 32-bit timeval, host[256]).
func DecodeDarwin(data []byte) []Entry {
	const size = 628
	le := binary.LittleEndian
	var out []Entry
	for ; len(data) >= size; data = data[size:] {
		r := data[:size]
		t, ok := darwinTypes[int16(le.Uint16(r[296:]))]
		if !ok {
			continue
		}
		out = append(out, Entry{
			Type: t,
			User: cstr(r[0:256]),
			ID:   cstr(r[256:260]),
			Line: cstr(r[260:292]),
			PID:  int(int32(le.Uint32(r[292:]))),
			Time: time.Unix(int64(int32(le.Uint32(r[300:]))), int64(int32(le.Uint32(r[304:])))*1000),
			Host: cstr(r[308:564]),
		})
	}
	return out
}

// freebsdTypes maps FreeBSD's utmpx types onto the Linux numbering.
var freebsdTypes = map[byte]Type{
	0: Empty, 1: BootTime, 2: OldTime, 3: NewTime, 4: UserProcess, 5: InitProcess,
	6: LoginProcess, 7: DeadProcess,
}

// DecodeFreeBSD reads FreeBSD's packed, big-endian struct futx records (197
// bytes: type, time in microseconds, id[8], pid, user[32], line[16],
// host[128]).
func DecodeFreeBSD(data []byte) []Entry {
	const size = 197
	be := binary.BigEndian
	var out []Entry
	for ; len(data) >= size; data = data[size:] {
		r := data[:size]
		t, ok := freebsdTypes[r[0]]
		if !ok {
			continue
		}
		usec := int64(be.Uint64(r[1:]))
		out = append(out, Entry{
			Type: t,
			Time: time.UnixMicro(usec),
			ID:   cstr(r[9:17]),
			PID:  int(be.Uint32(r[17:])),
			User: cstr(r[21:53]),
			Line: cstr(r[53:69]),
			Host: cstr(r[69:197]),
		})
	}
	return out
}

// withBoot puts a boot record first when the records lack one, as gnulib's
// readutmp does, from the system's own boot time.
func withBoot(entries []Entry) []Entry {
	for _, e := range entries {
		if e.Type == BootTime {
			return entries
		}
	}
	if boot, err := Boot(); err == nil {
		return append([]Entry{{Type: BootTime, Line: "~", User: "reboot", Time: boot}}, entries...)
	}
	return entries
}
