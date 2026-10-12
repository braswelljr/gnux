package human

import "testing"

func TestReadable(t *testing.T) {
	h := Autoscale | SI | Base1024
	tests := []struct {
		n        uint64
		opts     Opts
		from, to uint64
		want     string
	}{
		{0, h, 1, 1, "0"},
		{1023, h, 1, 1, "1023"},
		{1024, h, 1, 1, "1.0K"},
		{1025, h, 1, 1, "1.1K"},
		{1536, h, 1, 1, "1.5K"},
		{4096, h, 1, 1, "4.0K"},
		{10240, h, 1, 1, "10K"},
		{10241, h, 1, 1, "11K"},
		{1048575, h, 1, 1, "1.0M"},
		{1048576 * 1023, h, 1, 1, "1023M"},
		{1048576*1024 - 1, h, 1, 1, "1.0G"},
		{1000, Autoscale | SI, 1, 1, "1.0k"},
		{999, Autoscale | SI, 1, 1, "999"},
		{1500000, Autoscale | SI, 1, 1, "1.5M"},
		{4096, 0, 1, 1024, "4"},
		{4097, 0, 1, 1024, "5"},
		{5 << 20, SI | Base1024, 1, 1 << 20, "5M"},
		{5<<20 + 1, SI | Base1024, 1, 1 << 20, "6M"},
		{5 << 20, SI | Base1024 | B, 1, 1 << 20, "5MiB"},
		{5000000, SI | B, 1, 1000000, "5MB"},
		{3, 0, 1024, 1, "3072"},
		{3, h, 1024, 1, "3.0K"},
		{100, 0, 3, 7, "43"},
	}
	for _, tt := range tests {
		if got := Readable(tt.n, tt.opts, tt.from, tt.to); got != tt.want {
			t.Errorf("Readable(%d, %d, %d, %d) = %q, want %q", tt.n, tt.opts, tt.from, tt.to, got, tt.want)
		}
	}
}

func TestParseBlockSize(t *testing.T) {
	tests := []struct {
		spec string
		size uint64
		opts Opts
	}{
		{"1", 1, 0},
		{"512", 512, 0},
		{"1K", 1024, 0},
		{"1k", 1024, 0},
		{"2KiB", 2048, 0},
		{"1KB", 1000, 0},
		{"1M", 1 << 20, 0},
		{"1m", 1 << 20, 0},
		{"K", 1024, SI | Base1024},
		{"M", 1 << 20, SI | Base1024},
		{"MB", 1000000, SI | B},
		{"MiB", 1 << 20, SI | B | Base1024},
		{"human-readable", 1, Autoscale | SI | Base1024},
		{"si", 1, Autoscale | SI},
		{"'1K", 1024, GroupDigits},
	}
	for _, tt := range tests {
		size, opts, err := ParseBlockSize(tt.spec)
		if err != nil || size != tt.size || opts != tt.opts {
			t.Errorf("ParseBlockSize(%q) = %d, %d, %v; want %d, %d", tt.spec, size, opts, err, tt.size, tt.opts)
		}
	}
	for _, bad := range []string{"", "x", "0", "1X", "1Kx", "1b", "1R", "99999999999999999999"} {
		if _, _, err := ParseBlockSize(bad); err == nil {
			t.Errorf("ParseBlockSize(%q) accepted", bad)
		}
	}
}

func TestDefaultBlockSize(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	if size, _ := DefaultBlockSize(getenv, "DU_BLOCK_SIZE"); size != 1024 {
		t.Fatalf("default = %d", size)
	}
	env["POSIXLY_CORRECT"] = "1"
	if size, _ := DefaultBlockSize(getenv, "DU_BLOCK_SIZE"); size != 512 {
		t.Fatalf("POSIXLY_CORRECT = %d", size)
	}
	env["BLOCK_SIZE"] = "1M"
	if size, _ := DefaultBlockSize(getenv, "DU_BLOCK_SIZE"); size != 1<<20 {
		t.Fatalf("BLOCK_SIZE = %d", size)
	}
	env["DU_BLOCK_SIZE"] = "human-readable"
	if size, opts := DefaultBlockSize(getenv, "DU_BLOCK_SIZE"); size != 1 || opts&Autoscale == 0 {
		t.Fatalf("DU_BLOCK_SIZE = %d %d", size, opts)
	}
}
