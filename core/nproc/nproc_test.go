package nproc_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nproc"
)

func run(t *testing.T, opts nproc.Options, env map[string]string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	n := nproc.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	n.Available = func() int { return 8 }
	n.Installed = func() int { return 16 }
	n.Getenv = func(k string) string { return env[k] }
	err := n.Run(args)
	return out.String(), err
}

func TestCount(t *testing.T) {
	tests := []struct {
		name string
		opts nproc.Options
		env  map[string]string
		want string
	}{
		{"available", nproc.Options{}, nil, "8\n"},
		{"all", nproc.Options{All: true}, nil, "16\n"},
		{"ignore", nproc.Options{Ignore: 3}, nil, "5\n"},
		{"ignore never goes below one", nproc.Options{Ignore: 8}, nil, "1\n"},
		{"ignore far too many", nproc.Options{Ignore: 1000}, nil, "1\n"},
		{"OMP_NUM_THREADS replaces the count", nproc.Options{}, map[string]string{"OMP_NUM_THREADS": "3"}, "3\n"},
		{"OMP_NUM_THREADS can exceed the CPUs", nproc.Options{}, map[string]string{"OMP_NUM_THREADS": "64"}, "64\n"},
		{"OMP_NUM_THREADS list takes the first", nproc.Options{}, map[string]string{"OMP_NUM_THREADS": " 4 ,2,1"}, "4\n"},
		{"OMP_NUM_THREADS garbage is ignored", nproc.Options{}, map[string]string{"OMP_NUM_THREADS": "4x"}, "8\n"},
		{"OMP_NUM_THREADS zero is ignored", nproc.Options{}, map[string]string{"OMP_NUM_THREADS": "0"}, "8\n"},
		{"OMP_THREAD_LIMIT caps", nproc.Options{}, map[string]string{"OMP_THREAD_LIMIT": "2"}, "2\n"},
		{"OMP_THREAD_LIMIT caps OMP_NUM_THREADS", nproc.Options{}, map[string]string{"OMP_NUM_THREADS": "6", "OMP_THREAD_LIMIT": "5"}, "5\n"},
		{"OMP_THREAD_LIMIT zero means none", nproc.Options{}, map[string]string{"OMP_THREAD_LIMIT": "0"}, "8\n"},
		{"all ignores OpenMP", nproc.Options{All: true}, map[string]string{"OMP_NUM_THREADS": "2"}, "16\n"},
		{"ignore applies after OpenMP", nproc.Options{Ignore: 1}, map[string]string{"OMP_NUM_THREADS": "3"}, "2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := run(t, tt.opts, tt.env)
			if err != nil || out != tt.want {
				t.Fatalf("out = %q, err = %v, want %q", out, err, tt.want)
			}
		})
	}
}

func TestRealCount(t *testing.T) {
	var out bytes.Buffer
	if err := nproc.New(nproc.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}}).Run(nil); err != nil {
		t.Fatal(err)
	}
	if out.Len() < 2 || out.String() == "0\n" {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestExtraOperand(t *testing.T) {
	_, err := run(t, nproc.Options{}, nil, "x")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'x'" {
		t.Fatalf("err = %v", err)
	}
}
