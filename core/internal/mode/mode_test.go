package mode

import "testing"

func TestParseForFiles(t *testing.T) {
	umask := func() uint32 { return 0o022 }
	tests := []struct {
		in   string
		dir  bool
		want uint32
	}{
		{"+X", true, 0o777},
		{"+X", false, 0o666},
		{"u+x,+X", false, 0o777}, // once the owner may search, X is x for all
		{"go=", false, 0o600},
		{"640", false, 0o640},
	}
	for _, tt := range tests {
		base := uint32(0o666)
		if tt.dir {
			base = 0o777
		}
		if got, err := Parse(tt.in, base, tt.dir, umask); err != nil || got != tt.want {
			t.Errorf("Parse(%q, dir=%v) = %#o, %v; want %#o", tt.in, tt.dir, got, err, tt.want)
		}
	}
}
