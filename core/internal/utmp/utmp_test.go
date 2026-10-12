package utmp

import (
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

func put(b []byte, off int, s string) { copy(b[off:], s) }

func TestDecodeLinux(t *testing.T) {
	r := make([]byte, 384*2)
	le := binary.LittleEndian
	le.PutUint16(r[0:], uint16(UserProcess))
	le.PutUint32(r[4:], 1234)
	put(r, 8, "pts/0")
	put(r, 40, "ts/0")
	put(r, 44, "alice")
	put(r, 76, "10.0.0.5")
	le.PutUint32(r[340:], 1700000000)
	le.PutUint32(r[344:], 250000)
	b := r[384:]
	le.PutUint16(b[0:], uint16(DeadProcess))
	le.PutUint16(b[332:], 0)
	le.PutUint16(b[334:], 3)
	got := DecodeLinux(r)
	want := Entry{Type: UserProcess, PID: 1234, Line: "pts/0", ID: "ts/0", User: "alice", Host: "10.0.0.5",
		Time: time.Unix(1700000000, 250000000)}
	if len(got) != 2 || got[0] != want || got[1].Type != DeadProcess || got[1].Exit != 3 {
		t.Fatalf("DecodeLinux = %+v", got)
	}
}

func TestDecodeDarwin(t *testing.T) {
	r := make([]byte, 628*2)
	le := binary.LittleEndian
	put(r, 0, "utmpx-1.00")
	le.PutUint16(r[296:], 10) // the signature record is skipped
	b := r[628:]
	put(b, 0, "bob")
	put(b, 260, "console")
	le.PutUint32(b[292:], 410)
	le.PutUint16(b[296:], 7)
	le.PutUint32(b[300:], 1791542686)
	got := DecodeDarwin(r)
	if len(got) != 1 || got[0].User != "bob" || got[0].Line != "console" || got[0].Type != UserProcess ||
		got[0].PID != 410 || got[0].Time.Unix() != 1791542686 {
		t.Fatalf("DecodeDarwin = %+v", got)
	}
}

func TestDecodeFreeBSD(t *testing.T) {
	r := make([]byte, 197)
	be := binary.BigEndian
	r[0] = 4 // USER_PROCESS
	be.PutUint64(r[1:], 1700000000*1000000)
	put(r, 9, "abcd")
	be.PutUint32(r[17:], 77)
	put(r, 21, "carol")
	put(r, 53, "ttyv0")
	got := DecodeFreeBSD(r)
	if len(got) != 1 || got[0].User != "carol" || got[0].Line != "ttyv0" || got[0].PID != 77 ||
		got[0].Type != UserProcess || got[0].Time.Unix() != 1700000000 {
		t.Fatalf("DecodeFreeBSD = %+v", got)
	}
}

func TestLive(t *testing.T) {
	entries, err := Current()
	if errors.Is(err, ErrUnsupported) {
		t.Skip("no session records here")
	}
	if err != nil {
		t.Fatal(err)
	}
	boot, err := Boot()
	if err != nil || boot.After(time.Now()) || time.Since(boot) > 10*365*24*time.Hour {
		t.Fatalf("Boot = %v, %v", boot, err)
	}
	found := false
	for _, e := range entries {
		found = found || e.Type == BootTime
	}
	if !found {
		t.Fatalf("no boot record in %+v", entries)
	}
	if avg, err := LoadAvg(); err == nil && (len(avg) != 3 || avg[0] < 0) {
		t.Fatalf("LoadAvg = %v", avg)
	}
}
