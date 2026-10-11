//go:build !windows

package env_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/braswelljr/gnux/core/env"
)

// Whether a file can be executed is decided by its mode bits only on Unix;
// Windows goes by the file extension instead.
func TestCommandNotExecutable(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain")
	if err := os.WriteFile(plain, []byte("not a program\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	garbage := filepath.Join(dir, "garbage")
	if err := os.WriteFile(garbage, []byte("\x00\x01not a program\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"no execute bit, by path", plain, "envx: '" + plain + "': Permission denied\n"},
		{"no execute bit, on PATH", "plain", "envx: 'plain': Permission denied\n"},
		{"directory", dir, "envx: '" + dir + "': Permission denied\n"},
		{"not a program", garbage, "envx: '" + garbage + "': Exec format error\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := run(env.Options{}, []string{"PATH=" + dir}, "", tt.command)
			wantExit(t, r.err, env.ExitCannotInvoke)
			if r.stderr != tt.want || r.stdout != "" {
				t.Errorf("stdout = %q, stderr = %q, want stderr %q", r.stdout, r.stderr, tt.want)
			}
		})
	}
}

func TestDefaultPathWhenUnset(t *testing.T) {
	// With no PATH at all the standard system directories are searched, so
	// `envx -i sh` keeps working. A name that exists nowhere still fails.
	r := run(env.Options{IgnoreEnvironment: true}, nil, "", "gnux-no-such-command")
	wantExit(t, r.err, env.ExitNotFound)
	if want := "envx: 'gnux-no-such-command': No such file or directory\n"; r.stderr != want {
		t.Fatalf("stderr = %q, want %q", r.stderr, want)
	}
}

func TestRelativeCommandResolvesAgainstChdir(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "helper")
	if err := os.Symlink(os.Args[0], link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	// "./helper" exists only as seen from the new working directory.
	r := run(env.Options{Chdir: dir}, nil, "",
		helperVar+"=args", quickExit, "./helper", "-test.run=^TestHelperProcess$", "--", "ok")
	if want := "ok\n"; r.err != nil || r.stdout != want {
		t.Fatalf("stdout = %q, err = %v (stderr %q); want %q", r.stdout, r.err, r.stderr, want)
	}
}
