package mkdir

import "testing"

func TestParseMode(t *testing.T) {
	const mask = 0o022
	tests := []struct {
		in   string
		want uint32
	}{
		{"755", 0o755},
		{"0700", 0o700},
		{"1777", 0o1777},
		{"0", 0},
		{"u=rwx,go=rx", 0o755},
		{"u=rwx,g=rx,o=", 0o750},
		{"a=rx", 0o555},
		{"go-w", 0o755},
		{"go-rwx", 0o700},
		{"a-x", 0o666},
		{"a+x", 0o777},
		{"o-rwx,g-w", 0o750},
		{"g=u,o=", 0o770},
		{"u=rx,g=u", 0o557},
		{"a=,u+rwx", 0o700},
		{"u-w+w", 0o777},
		{"+t", 0o1777},
		{"g+s", 0o2777},
		{"u+s,o-rwx", 0o4770},
		{"a=X", 0o111},
		// No user class: bits in the umask are not touched.
		{"-w", 0o577},
		{"=rx", 0o555},
		{"=rwx", 0o755},
	}
	for _, tt := range tests {
		got, err := parseMode(tt.in, func() uint32 { return mask })
		if err != nil {
			t.Errorf("parseMode(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseMode(%q) = %#o, want %#o", tt.in, got, tt.want)
		}
	}
}

func TestParseModeInvalid(t *testing.T) {
	for _, in := range []string{"", "abc", "888", "17777", "u", "u+rwz", "rwx", "u=rwx,", ",u=rwx", "u=rwx go=rx", "+u+q"} {
		if got, err := parseMode(in, func() uint32 { return 0 }); err == nil {
			t.Errorf("parseMode(%q) = %#o, want error", in, got)
		}
	}
}
