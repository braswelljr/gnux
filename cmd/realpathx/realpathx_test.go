package realpathx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/realpath"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want realpath.Options
	}{
		{"defaults", nil, realpath.Options{}},
		{"existing", []string{"-e"}, realpath.Options{Mode: realpath.ModeExisting}},
		{"long existing", []string{"--canonicalize-existing"}, realpath.Options{Mode: realpath.ModeExisting}},
		{"missing", []string{"-m"}, realpath.Options{Mode: realpath.ModeMissing}},
		{"long missing", []string{"--canonicalize-missing"}, realpath.Options{Mode: realpath.ModeMissing}},
		{"existing beats missing", []string{"-m", "-e"}, realpath.Options{Mode: realpath.ModeExisting}},
		{"logical", []string{"-L"}, realpath.Options{Logical: true}},
		{"long logical", []string{"--logical"}, realpath.Options{Logical: true}},
		{"physical", []string{"-P"}, realpath.Options{}},
		{"long physical", []string{"--physical"}, realpath.Options{}},
		{"logical beats physical", []string{"-L", "-P"}, realpath.Options{Logical: true}},
		{"quiet", []string{"-q"}, realpath.Options{Quiet: true}},
		{"long quiet", []string{"--quiet"}, realpath.Options{Quiet: true}},
		{"strip", []string{"-s"}, realpath.Options{Strip: true}},
		{"long strip", []string{"--strip"}, realpath.Options{Strip: true}},
		{"no symlinks", []string{"--no-symlinks"}, realpath.Options{Strip: true}},
		{"zero", []string{"-z"}, realpath.Options{Zero: true}},
		{"long zero", []string{"--zero"}, realpath.Options{Zero: true}},
		{"relative to", []string{"--relative-to=/usr"}, realpath.Options{RelativeTo: "/usr"}},
		{"relative base", []string{"--relative-base", "/usr"}, realpath.Options{RelativeBase: "/usr"}},
		{"combined", []string{"-mqsz", "--relative-to=a", "--relative-base=b"}, realpath.Options{
			Mode: realpath.ModeMissing, Quiet: true, Strip: true, Zero: true, RelativeTo: "a", RelativeBase: "b",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(realpath.Name, pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("parse %v: %v", tt.args, err)
			}
			if got := buildOptions(fs); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}
