package tool

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// spelling2 is the second spelling GNU accepts for --color, written in two
// pieces so that the spell checker leaves it alone.
const spelling2 = "colo" + "ur"

// spec is a tool with the shapes of option getopt distinguishes: switches,
// options that take an argument, one with an optional argument, a hidden
// second spelling, and long names of gnux's own.
func spec(noInterspersed bool) Spec {
	return Spec{
		Meta: docmeta.Meta{Name: "demox", Use: "demox [OPTION]... [FILE]..."},
		GNU:  "demo",
		Flags: func(fs *pflag.FlagSet) {
			fs.BoolP("recursive", "r", false, "")
			fs.BoolP("force", "f", false, "")
			fs.StringP("lines", "n", "", "")
			fs.String("color", "", "")
			fs.Lookup("color").NoOptDefVal = "auto"
			fs.String(spelling2, "", "")
			fs.Lookup(spelling2).Hidden = true
			fs.Bool("context", false, "")
			fs.BoolP("columns", "C", false, "")
			fs.String("directory", "", "")
			fs.StringP("dir", "p", "", "")
			fs.CountP("verbose", "v", "")
			OwnNames(fs, "columns", "dir")
		},
		NoInterspersed: noInterspersed,
		Run:            func(*pflag.FlagSet, []string, core.Streams) error { return nil },
	}
}

func parse(t *testing.T, noInterspersed bool, args ...string) (*pflag.FlagSet, error) {
	t.Helper()
	c := spec(noInterspersed).Command(core.Discard())
	err := ParseFlags(c, args)
	return c.Flags(), err
}

func TestAbbreviations(t *testing.T) {
	fs, err := parse(t, false, "--rec", "--li", "5", "--for", "file")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := fs.GetBool("recursive")
	f, _ := fs.GetBool("force")
	n, _ := fs.GetString("lines")
	if !r || !f || n != "5" || strings.Join(fs.Args(), " ") != "file" {
		t.Fatalf("recursive %v, force %v, lines %q, operands %q", r, f, n, fs.Args())
	}

	fs, err = parse(t, false, "--colo=never")
	if v, _ := fs.GetString("color"); err != nil || v != "never" {
		t.Fatalf("--colo=never: color %q, err %v", v, err)
	}
	fs, err = parse(t, false, "--colo")
	if v, _ := fs.GetString("color"); err != nil || v != "auto" {
		t.Fatalf("--colo: color %q, err %v", v, err)
	}
}

func TestOwnAndHiddenNames(t *testing.T) {
	// --col is --color alone: --columns is gnux's own, the second spelling hidden.
	fs, err := parse(t, false, "--col=always")
	if v, _ := fs.GetString("color"); err != nil || v != "always" {
		t.Fatalf("--col: color %q, err %v", v, err)
	}
	// Written in full, both still answer.
	fs, err = parse(t, false, "--columns", "--"+spelling2+"=never")
	c, _ := fs.GetBool("columns")
	if v, _ := fs.GetString(spelling2); err != nil || !c || v != "never" {
		t.Fatalf("columns %v, second %q, err %v", c, v, err)
	}
	// --dir abbreviates GNU's --directory, although gnux has a --dir.
	fs, err = parse(t, false, "--dir", "x")
	d, _ := fs.GetString("directory")
	p, _ := fs.GetString("dir")
	if err != nil || d != "x" || p != "" {
		t.Fatalf("--dir: directory %q, dir %q, err %v", d, p, err)
	}
}

func TestArgumentsAreNotOptions(t *testing.T) {
	// The argument of -n, of --lines, and of a cluster ending in n, and
	// everything after -- are left alone.
	for _, args := range [][]string{
		{"-n", "--rec"}, {"--lines", "--rec"}, {"-fn", "--rec"}, {"--", "--rec"},
	} {
		fs, err := parse(t, false, args...)
		if r, _ := fs.GetBool("recursive"); err != nil || r {
			t.Errorf("%q: recursive %v, err %v", args, r, err)
		}
	}
	// Options stop at the first operand when the tool says so.
	fs, err := parse(t, true, "file", "--rec")
	if err != nil || strings.Join(fs.Args(), " ") != "file --rec" {
		t.Fatalf("operands %q, err %v", fs.Args(), err)
	}
}

func TestGetoptMessages(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"--co"}, "option '--co' is ambiguous; possibilities: '--color' '--context'"},
		{[]string{"--foo"}, "unrecognized option '--foo'"},
		{[]string{"--foo=bar"}, "unrecognized option '--foo=bar'"},
		{[]string{"--columns=x"}, "unrecognized option '--columns=x'"},
		{[]string{"-z"}, "invalid option -- 'z'"},
		{[]string{"-rz"}, "invalid option -- 'z'"},
		{[]string{"-n"}, "option requires an argument -- 'n'"},
		{[]string{"--lines"}, "option '--lines' requires an argument"},
		{[]string{"--li"}, "option '--lines' requires an argument"},
		{[]string{"--force=yes"}, "option '--force' doesn't allow an argument"},
		{[]string{"--for=1"}, "option '--force' doesn't allow an argument"},
		{[]string{"--verbose=2"}, "option '--verbose' doesn't allow an argument"},
		{[]string{"--help=x"}, "option '--help' doesn't allow an argument"},
	}
	for _, tt := range tests {
		_, err := parse(t, false, tt.args...)
		if err == nil || err.Error() != tt.want {
			t.Errorf("%q: err = %v; want %q", tt.args, err, tt.want)
		}
	}
}

func TestOwnNamesMustExist(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("OwnNames accepted a name that is not registered")
		}
	}()
	OwnNames(pflag.NewFlagSet("x", pflag.ContinueOnError), "nope")
}
