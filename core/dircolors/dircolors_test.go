package dircolors_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/dircolors"
)

func run(opts dircolors.Options, env map[string]string, stdin string, args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	d := dircolors.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	d.Getenv = func(k string) string { return env[k] }
	err := d.Run(args)
	return out.String(), errb.String(), err
}

func TestDefaultDatabase(t *testing.T) {
	out, errb, err := run(dircolors.Options{}, map[string]string{"SHELL": "/bin/bash", "TERM": "xterm-256color"}, "")
	if err != nil {
		t.Fatalf("%v: %s", err, errb)
	}
	if !strings.HasPrefix(out, "LS_COLORS='rs=0:di=01;34:ln=01;36:") || !strings.HasSuffix(out, ":';\nexport LS_COLORS\n") {
		t.Fatalf("stdout = %q", out)
	}
	if !strings.Contains(out, ":*.tar=01;31:") {
		t.Fatalf("no archive colors in %q", out)
	}
}

func TestShellGuess(t *testing.T) {
	out, _, _ := run(dircolors.Options{}, map[string]string{"SHELL": "/usr/bin/tcsh", "TERM": "xterm"}, "")
	if !strings.HasPrefix(out, "setenv LS_COLORS '") || !strings.HasSuffix(out, "'\n") {
		t.Fatalf("stdout = %q", out)
	}
	_, errb, err := run(dircolors.Options{}, nil, "")
	if !errors.Is(err, core.ErrFailed) || errb != "dircolorsx: no SHELL environment variable, and no shell type option given\n" {
		t.Fatalf("no SHELL: %v %q", err, errb)
	}
}

func TestTerminalFilter(t *testing.T) {
	db := "TERM xterm*\nDIR 01;34\nTERM dumb\nLINK 36\n"
	out, _, err := run(dircolors.Options{Shell: dircolors.Bourne}, map[string]string{"TERM": "xterm"}, db, "-")
	if err != nil || out != "LS_COLORS='di=01;34:';\nexport LS_COLORS\n" {
		t.Fatalf("xterm: %v %q", err, out)
	}
	out, _, _ = run(dircolors.Options{Shell: dircolors.Bourne}, map[string]string{"TERM": "dumb"}, db, "-")
	if out != "LS_COLORS='ln=36:';\nexport LS_COLORS\n" {
		t.Fatalf("dumb: %q", out)
	}
	out, _, _ = run(dircolors.Options{Shell: dircolors.Bourne}, map[string]string{"TERM": "vt100"}, "COLORTERM ?*\nDIR 1\n", "-")
	if out != "LS_COLORS='';\nexport LS_COLORS\n" {
		t.Fatalf("no COLORTERM: %q", out)
	}
}

func TestEntriesAndQuoting(t *testing.T) {
	db := "# comment\n  \n.tar 01;31 # archives\n*~ 90\nexec 01;32\nOPTIONS -F\n*a:b=c 1\n*it's 2\n"
	out, errb, err := run(dircolors.Options{Shell: dircolors.Bourne}, nil, db, "-")
	want := `LS_COLORS='*.tar=01;31:*~=90:ex=01;32:*a\:b\=c=1:*it'\''s=2:';` + "\nexport LS_COLORS\n"
	if err != nil || out != want {
		t.Fatalf("err=%v stderr=%q\nstdout %q\nwant   %q", err, errb, out, want)
	}
}

func TestPrintLsColors(t *testing.T) {
	out, _, err := run(dircolors.Options{PrintLsColors: true}, nil, "DIR 01;34\n.go 32\n", "-")
	if err != nil || out != "\x1b[01;34mdi\t01;34\x1b[0m\n\x1b[32m*.go\t32\x1b[0m\n" {
		t.Fatalf("%v %q", err, out)
	}
}

func TestDatabaseMistakes(t *testing.T) {
	f := filepath.Join(t.TempDir(), "db")
	if err := os.WriteFile(f, []byte("TYPO 1\nTERM *\nDIR\nBOGUS 1\nLINK 36\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errb, err := run(dircolors.Options{Shell: dircolors.Bourne}, nil, "", f)
	// An unknown keyword before any TERM line passes unnoticed, as in GNU.
	want := "dircolorsx: " + f + ":3: invalid line;  missing second token\ndircolorsx: " + f + ":4: unrecognized keyword BOGUS\n"
	if !errors.Is(err, core.ErrFailed) || out != "" || errb != want {
		t.Fatalf("err=%v stdout=%q stderr=%q", err, out, errb)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	_, errb, _ = run(dircolors.Options{Shell: dircolors.Bourne}, nil, "", missing)
	if errb != "dircolorsx: "+missing+": No such file or directory\n" {
		t.Fatalf("missing: %q", errb)
	}
}

func TestPrintDatabase(t *testing.T) {
	out, _, err := run(dircolors.Options{PrintDatabase: true}, nil, "")
	if err != nil || out != dircolors.Database || !strings.HasPrefix(out, "# Configuration file for dircolors") {
		t.Fatalf("err=%v stdout starts %q", err, out[:min(len(out), 40)])
	}
}

func TestUsage(t *testing.T) {
	for _, tt := range []struct {
		opts dircolors.Options
		args []string
		msg  string
	}{
		{dircolors.Options{PrintDatabase: true, Shell: dircolors.C}, nil, "the options to output non shell syntax,\nand to select a shell syntax are mutually exclusive"},
		{dircolors.Options{PrintDatabase: true, PrintLsColors: true}, nil, "options --print-database and --print-ls-colors are mutually exclusive"},
		{dircolors.Options{PrintDatabase: true}, []string{"x"}, "extra operand 'x'\nfile operands cannot be combined with --print-database (-p)"},
		{dircolors.Options{}, []string{"a", "b"}, "extra operand 'b'"},
	} {
		_, _, err := run(tt.opts, nil, "", tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.msg {
			t.Errorf("%+v %v: err = %v", tt.opts, tt.args, err)
		}
	}
}
