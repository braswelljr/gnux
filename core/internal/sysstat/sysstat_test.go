package sysstat

import (
	"os"
	"testing"
)

func TestRawMode(t *testing.T) {
	tests := []struct {
		mode os.FileMode
		want uint32
	}{
		{0o644, typeReg | 0o644},
		{os.ModeDir | 0o755, typeDir | 0o755},
		{os.ModeSymlink | 0o777, typeLnk | 0o777},
		{os.ModeNamedPipe | 0o600, typeFIFO | 0o600},
		{os.ModeSocket | 0o700, typeSock | 0o700},
		{os.ModeDevice | 0o660, typeBlk | 0o660},
		{os.ModeDevice | os.ModeCharDevice | 0o666, typeChar | 0o666},
		{os.ModeSetuid | os.ModeSetgid | os.ModeSticky | 0o755, typeReg | 0o7755},
	}
	for _, tt := range tests {
		if got := rawMode(tt.mode); got != tt.want {
			t.Errorf("rawMode(%v) = %#o, want %#o", tt.mode, got, tt.want)
		}
	}
}
