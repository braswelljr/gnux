package common

import (
	"runtime/debug"
	"testing"
)

func TestBuildInfoVersion(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{"no build info", nil, false, "dev"},
		{"devel build", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true, "dev"},
		{"empty version", &debug.BuildInfo{}, true, "dev"},
		{"go install by tag", &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}, true, "v1.2.3"},
		{
			"checkout pseudo-version",
			&debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20261011120000-abcdef123456+dirty"}},
			true,
			"v0.0.0-20261011120000-abcdef123456+dirty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildInfoVersion(tt.info, tt.ok); got != tt.want {
				t.Errorf("buildInfoVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
