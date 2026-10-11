package cpx

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cp"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (cp.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(cp.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	standard := cp.Preserve{Mode: true, Ownership: true, Timestamps: true}

	tests := []struct {
		name string
		args []string
		want cp.Options
	}{
		{"defaults", nil, cp.Options{}},
		{"recursive short", []string{"-r"}, cp.Options{Recursive: true}},
		{"recursive upper", []string{"-R"}, cp.Options{Recursive: true}},
		{"recursive long", []string{"--recursive"}, cp.Options{Recursive: true}},
		{"archive", []string{"-a"}, cp.Options{Recursive: true, Dereference: cp.DerefNever, Preserve: cp.PreserveAll()}},
		{"archive long", []string{"--archive"}, cp.Options{Recursive: true, Dereference: cp.DerefNever, Preserve: cp.PreserveAll()}},
		{"force", []string{"-f"}, cp.Options{Force: true}},
		{"interactive", []string{"-i"}, cp.Options{Interactive: true}},
		{"no-clobber", []string{"-n"}, cp.Options{NoClobber: true}},
		{"update short", []string{"-u"}, cp.Options{Update: true}},
		{"update bare", []string{"--update"}, cp.Options{Update: true}},
		{"update=older", []string{"--update=older"}, cp.Options{Update: true}},
		{"update=none", []string{"--update=none"}, cp.Options{NoClobber: true}},
		{"update=all", []string{"--update=all"}, cp.Options{}},
		{"verbose", []string{"-v"}, cp.Options{Verbose: true}},
		{"-p", []string{"-p"}, cp.Options{Preserve: standard}},
		{"preserve bare", []string{"--preserve"}, cp.Options{Preserve: standard}},
		{"preserve=mode", []string{"--preserve=mode"}, cp.Options{Preserve: cp.Preserve{Mode: true}}},
		{"preserve list", []string{"--preserve=timestamps,links"}, cp.Options{Preserve: cp.Preserve{Timestamps: true, Links: true}}},
		{"preserve=all", []string{"--preserve=all"}, cp.Options{Preserve: cp.PreserveAll()}},
		{"preserve ignores context and xattr", []string{"--preserve=context,xattr,ownership"}, cp.Options{Preserve: cp.Preserve{Ownership: true}}},
		{"no-preserve trims -p", []string{"-p", "--no-preserve=ownership"}, cp.Options{Preserve: cp.Preserve{Mode: true, Timestamps: true}}},
		{"no-preserve trims -a", []string{"-a", "--no-preserve=links,mode"},
			cp.Options{Recursive: true, Dereference: cp.DerefNever, Preserve: cp.Preserve{Ownership: true, Timestamps: true}}},
		{"-d", []string{"-d"}, cp.Options{Dereference: cp.DerefNever, Preserve: cp.Preserve{Links: true}}},
		{"-P", []string{"-P"}, cp.Options{Dereference: cp.DerefNever}},
		{"no-dereference long", []string{"--no-dereference"}, cp.Options{Dereference: cp.DerefNever}},
		{"-L", []string{"-L"}, cp.Options{Dereference: cp.DerefAlways}},
		{"dereference long", []string{"--dereference"}, cp.Options{Dereference: cp.DerefAlways}},
		{"-H", []string{"-H"}, cp.Options{Dereference: cp.DerefCommandLine}},
		{"link", []string{"-l"}, cp.Options{Link: true}},
		{"symbolic link", []string{"-s"}, cp.Options{SymbolicLink: true}},
		{"target directory", []string{"-t", "dir"}, cp.Options{TargetDirectory: "dir"}},
		{"target directory long", []string{"--target-directory=dir"}, cp.Options{TargetDirectory: "dir"}},
		{"no target directory", []string{"-T"}, cp.Options{NoTargetDirectory: true}},
		{"one file system", []string{"-x"}, cp.Options{OneFileSystem: true}},
		{"parents", []string{"--parents"}, cp.Options{Parents: true}},
		{"strip trailing slashes", []string{"--strip-trailing-slashes"}, cp.Options{StripTrailingSlashes: true}},
		{"combined -rvf", []string{"-rvf"}, cp.Options{Recursive: true, Verbose: true, Force: true}},
		{"combined -uv keeps the default WHEN", []string{"-uv"}, cp.Options{Update: true, Verbose: true}},

		// Precedence: the most cautious option wins, whatever the order.
		{"-n beats -i", []string{"-i", "-n"}, cp.Options{NoClobber: true}},
		{"-n beats -i reversed", []string{"-n", "-i"}, cp.Options{NoClobber: true}},
		{"-f does not cancel -i", []string{"-f", "-i"}, cp.Options{Force: true, Interactive: true}},
		{"-P beats -L", []string{"-L", "-P"}, cp.Options{Dereference: cp.DerefNever}},
		{"-H beats -L", []string{"-L", "-H"}, cp.Options{Dereference: cp.DerefCommandLine}},
		{"explicit -L overrides the -P implied by -a", []string{"-aL"},
			cp.Options{Recursive: true, Dereference: cp.DerefAlways, Preserve: cp.PreserveAll()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalidValues(t *testing.T) {
	for _, args := range [][]string{
		{"--preserve=bogus"},
		{"--no-preserve=bogus"},
		{"--update=bogus"},
	} {
		_, err := options(t, args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) {
			t.Errorf("buildOptions(%v) err = %v, want a usage error", args, err)
			continue
		}
		if !strings.Contains(usage.Msg, "invalid argument 'bogus'") {
			t.Errorf("buildOptions(%v) usage = %q", args, usage.Msg)
		}
	}
}

func TestSpec(t *testing.T) {
	spec := Spec()
	if spec.Name() != "cpx" || spec.GNU != "cp" {
		t.Fatalf("spec = %q replacing %q, want cpx replacing cp", spec.Name(), spec.GNU)
	}
	if !strings.HasPrefix(spec.Meta.Use, "cpx ") {
		t.Fatalf("Use = %q, want it to start with the tool name", spec.Meta.Use)
	}

	c := spec.Command(core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard})
	c.SetArgs(nil)
	var usage *core.UsageError
	if err := c.Execute(); !errors.As(err, &usage) {
		t.Fatalf("err = %v, want a usage error for a missing operand", err)
	}
}
