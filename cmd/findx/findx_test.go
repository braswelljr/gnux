package findx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/find"
	"github.com/braswelljr/gnux/internal/tool"
)

// execute runs the findx command as cmd.Main would, in dir.
func execute(t *testing.T, dir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	t.Chdir(dir)
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs(args)
	err = c.Execute()
	return out.String(), errb.String(), err
}

func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.txt", "b.log", filepath.Join("sub", "c.txt")} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestSpec(t *testing.T) {
	spec := Spec()
	if spec.Name() != find.Name || spec.GNU != "find" {
		t.Fatalf("name = %q, GNU = %q", spec.Name(), spec.GNU)
	}
	// The expression is not GNU-style options: cobra must not parse it.
	if !spec.RawArgs || spec.Flags != nil || spec.Run == nil {
		t.Fatalf("RawArgs = %v, Flags set = %v, Run set = %v", spec.RawArgs, spec.Flags != nil, spec.Run != nil)
	}
	// Usage errors exit 1, GNU find's only failure status.
	if spec.UsageExit != 0 {
		t.Fatalf("UsageExit = %d, want the default", spec.UsageExit)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if !strings.HasPrefix(m.Use, find.Name+" ") {
		t.Errorf("Use = %q", m.Use)
	}
	if m.Short == "" || m.Permissions == "" || len(m.UseCases) == 0 || len(m.SeeAlso) == 0 || len(m.ExitStatus) != 2 {
		t.Errorf("incomplete Meta: %+v", m)
	}
	if !strings.Contains(m.Long, "Differences from GNU find:") {
		t.Error("Long does not list the differences from GNU find")
	}
	if len(m.Examples) < 8 {
		t.Errorf("%d examples, want at least 8", len(m.Examples))
	}
	for _, e := range m.Examples {
		if e.Description == "" || e.Bash == "" || e.PowerShell == "" {
			t.Errorf("incomplete example: %+v", e)
		}
	}
	if strings.Contains(m.Long+m.Permissions, "\u2014") {
		t.Error("em dash in the documentation")
	}
}

func TestBuildOptions(t *testing.T) {
	// Nothing on find's command line is a flag, so the options are always
	// the zero value: plain find.
	fs := pflag.NewFlagSet(find.Name, pflag.ContinueOnError)
	if got := buildOptions(fs); got != (find.Options{}) {
		t.Fatalf("buildOptions = %+v, want the zero value", got)
	}
}

func TestCommand(t *testing.T) {
	root := tree(t)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no arguments", []string{}, ".\n./a.txt\n./b.log\n./sub\n./sub/c.txt\n"},
		{"single-dash tests reach the engine", []string{".", "-name", "*.txt"}, "./a.txt\n./sub/c.txt\n"},
		{"operators reach the engine", []string{".", "(", "-name", "a.txt", "-o", "-name", "b.log", ")", "-print"}, "./a.txt\n./b.log\n"},
		{"negation", []string{".", "-type", "f", "!", "-name", "*.txt"}, "./b.log\n"},
		{"-L before the starting point", []string{"-L", "sub", "-type", "f"}, "sub/c.txt\n"},
		{"double dash before the starting point", []string{"--", "sub"}, "sub\nsub/c.txt\n"},
		{"-printf", []string{"sub", "-type", "f", "-printf", `%f %s\n`}, "c.txt 1\n"},
		{"a global option", []string{"sub", "-maxdepth", "0"}, "sub\n"},
		{"--help among other words is not help", []string{"sub", "-maxdepth", "0", "-name", "--help"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := execute(t, root, tt.args...)
			if err != nil || errb != "" {
				t.Fatalf("err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	root := tree(t)

	_, _, err := execute(t, root, ".", "-nosuchtest")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "unknown predicate `-nosuchtest'" {
		t.Fatalf("err = %v, want the engine's usage error", err)
	}

	out, errb, err := execute(t, root, "missing", "a.txt")
	if !errors.Is(err, core.ErrFailed) || out != "a.txt\n" || errb != "findx: 'missing': No such file or directory\n" {
		t.Fatalf("out = %q, stderr = %q, err = %v", out, errb, err)
	}
}

func TestLoneHelpAndVersion(t *testing.T) {
	root := tree(t)
	out, _, err := execute(t, root, "--version")
	if err != nil || !strings.HasPrefix(out, find.Name+" ") {
		t.Fatalf("--version: %q, %v", out, err)
	}
	out, _, err = execute(t, root, "--help")
	if err != nil || !strings.Contains(out, find.Name) {
		t.Fatalf("--help: %q, %v", out, err)
	}
}

// TestExecRunsTools checks that -exec reaches the registry: the adapter
// hands tool.RunTool to the engine.
func TestExecRunsTools(t *testing.T) {
	root := tree(t)
	saved := tool.RunTool
	t.Cleanup(func() { tool.RunTool = saved })

	var calls []string
	tool.RunTool = func(name string, args []string, s core.Streams) (int, bool) {
		if name != "echo" {
			return 0, false
		}
		calls = append(calls, strings.Join(args, " "))
		_, _ = s.Out.Write([]byte("tool: " + strings.Join(args, " ") + "\n"))
		return 0, true
	}

	out, errb, err := execute(t, root, ".", "-name", "*.txt", "-exec", "echo", "{}", "+")
	want := "tool: ./a.txt ./sub/c.txt\n"
	if err != nil || errb != "" || out != want {
		t.Fatalf("out = %q, stderr = %q, err = %v; want %q", out, errb, err, want)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %q, want one", calls)
	}

	// A name the registry does not know is looked for on PATH, and not
	// finding it there is diagnosed.
	_, errb, err = execute(t, root, "a.txt", "-exec", "gnux-no-such-program-anywhere", ";")
	if !errors.Is(err, core.ErrFailed) || errb != "findx: 'gnux-no-such-program-anywhere': No such file or directory\n" {
		t.Fatalf("stderr = %q, err = %v", errb, err)
	}
}
