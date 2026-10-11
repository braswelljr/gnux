package lsx

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/ls"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (ls.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(ls.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

// want builds the expected Options from the defaults.
func want(mutate func(o *ls.Options)) ls.Options {
	o := ls.Options{TabSize: 8}
	if mutate != nil {
		mutate(&o)
	}
	return o
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want ls.Options
	}{
		{"defaults", nil, want(nil)},

		{"all", []string{"-a"}, want(func(o *ls.Options) { o.All = true })},
		{"all long", []string{"--all"}, want(func(o *ls.Options) { o.All = true })},
		{"almost all", []string{"-A"}, want(func(o *ls.Options) { o.AlmostAll = true })},
		{"almost all long", []string{"--almost-all"}, want(func(o *ls.Options) { o.AlmostAll = true })},
		{"-a then -A: last wins", []string{"-a", "-A"}, want(func(o *ls.Options) { o.AlmostAll = true })},
		{"-A then -a: last wins", []string{"-Aa"}, want(func(o *ls.Options) { o.All = true })},
		{"--all=false is not given", []string{"-A", "--all=false"}, want(func(o *ls.Options) { o.AlmostAll = true })},

		{"long", []string{"-l"}, want(func(o *ls.Options) { o.Format = ls.FormatLong })},
		{"one per line", []string{"-1"}, want(func(o *ls.Options) { o.Format = ls.FormatSingle })},
		{"columns", []string{"-C"}, want(func(o *ls.Options) { o.Format = ls.FormatColumns })},
		{"across", []string{"-x"}, want(func(o *ls.Options) { o.Format = ls.FormatAcross })},
		{"commas", []string{"-m"}, want(func(o *ls.Options) { o.Format = ls.FormatCommas })},
		{"-l then -C: last wins", []string{"-l", "-C"}, want(func(o *ls.Options) { o.Format = ls.FormatColumns })},
		{"-C then -l: last wins", []string{"-Cl"}, want(func(o *ls.Options) { o.Format = ls.FormatLong })},
		{"-m then -x then -1", []string{"-mx1"}, want(func(o *ls.Options) { o.Format = ls.FormatSingle })},
		{"-1 does not cancel -l", []string{"-l", "-1"}, want(func(o *ls.Options) { o.Format = ls.FormatLong })},
		{"-1 then -l", []string{"-1", "-l"}, want(func(o *ls.Options) { o.Format = ls.FormatLong })},
		{"-g implies long without owner", []string{"-g"}, want(func(o *ls.Options) { o.Format = ls.FormatLong; o.NoOwner = true })},
		{"-o implies long without group", []string{"-o"}, want(func(o *ls.Options) { o.Format = ls.FormatLong; o.NoGroup = true })},
		{"-n implies long with numeric ids", []string{"-n"}, want(func(o *ls.Options) { o.Format = ls.FormatLong; o.NumericIDs = true })},
		{"numeric long", []string{"--numeric-uid-gid"}, want(func(o *ls.Options) { o.Format = ls.FormatLong; o.NumericIDs = true })},
		{"-n then -C keeps numeric ids", []string{"-n", "-C"}, want(func(o *ls.Options) { o.Format = ls.FormatColumns; o.NumericIDs = true })},
		{"-G alone does not imply long", []string{"-G"}, want(func(o *ls.Options) { o.NoGroup = true })},
		{"no-group long", []string{"--no-group", "-l"}, want(func(o *ls.Options) { o.Format = ls.FormatLong; o.NoGroup = true })},
		{"full time", []string{"--full-time"}, want(func(o *ls.Options) { o.Format = ls.FormatLong; o.TimeStyle = "full-iso" })},

		{"directory", []string{"-d"}, want(func(o *ls.Options) { o.Directory = true })},
		{"directory long", []string{"--directory"}, want(func(o *ls.Options) { o.Directory = true })},
		{"recursive", []string{"-R"}, want(func(o *ls.Options) { o.Recursive = true })},
		{"recursive long", []string{"--recursive"}, want(func(o *ls.Options) { o.Recursive = true })},
		{"reverse", []string{"-r"}, want(func(o *ls.Options) { o.Reverse = true })},
		{"reverse long", []string{"--reverse"}, want(func(o *ls.Options) { o.Reverse = true })},

		{"sort by time", []string{"-t"}, want(func(o *ls.Options) { o.Sort = ls.SortTime })},
		{"sort by size", []string{"-S"}, want(func(o *ls.Options) { o.Sort = ls.SortSize })},
		{"sort by extension", []string{"-X"}, want(func(o *ls.Options) { o.Sort = ls.SortExtension })},
		{"unsorted", []string{"-U"}, want(func(o *ls.Options) { o.Sort = ls.SortNone })},
		{"sort by version", []string{"-v"}, want(func(o *ls.Options) { o.Sort = ls.SortVersion })},
		{"--sort=none", []string{"--sort=none"}, want(func(o *ls.Options) { o.Sort = ls.SortNone })},
		{"--sort=name", []string{"--sort=name"}, want(func(o *ls.Options) { o.Sort = ls.SortName })},
		{"--sort=time", []string{"--sort=time"}, want(func(o *ls.Options) { o.Sort = ls.SortTime })},
		{"--sort=size", []string{"--sort", "size"}, want(func(o *ls.Options) { o.Sort = ls.SortSize })},
		{"--sort=extension", []string{"--sort=extension"}, want(func(o *ls.Options) { o.Sort = ls.SortExtension })},
		{"--sort=version", []string{"--sort=version"}, want(func(o *ls.Options) { o.Sort = ls.SortVersion })},
		{"--sort=width", []string{"--sort=width"}, want(func(o *ls.Options) { o.Sort = ls.SortWidth })},
		{"--sort abbreviation", []string{"--sort=ext"}, want(func(o *ls.Options) { o.Sort = ls.SortExtension })},
		{"-t then -S: last wins", []string{"-tS"}, want(func(o *ls.Options) { o.Sort = ls.SortSize })},
		{"-S then --sort=time: last wins", []string{"-S", "--sort=time"}, want(func(o *ls.Options) { o.Sort = ls.SortTime })},
		{"--sort=time then -U: last wins", []string{"--sort=time", "-U"}, want(func(o *ls.Options) { o.Sort = ls.SortNone })},

		{"human readable", []string{"-h"}, want(func(o *ls.Options) { o.HumanReadable = true })},
		{"human readable long", []string{"--human-readable"}, want(func(o *ls.Options) { o.HumanReadable = true })},
		{"si", []string{"--si"}, want(func(o *ls.Options) { o.SI = true })},
		{"-h then --si: last wins", []string{"-h", "--si"}, want(func(o *ls.Options) { o.SI = true })},
		{"--si then -h: last wins", []string{"--si", "-h"}, want(func(o *ls.Options) { o.HumanReadable = true })},
		{"size", []string{"-s"}, want(func(o *ls.Options) { o.ShowBlocks = true })},
		{"size long", []string{"--size"}, want(func(o *ls.Options) { o.ShowBlocks = true })},
		{"inode", []string{"-i"}, want(func(o *ls.Options) { o.Inode = true })},
		{"inode long", []string{"--inode"}, want(func(o *ls.Options) { o.Inode = true })},

		{"classify", []string{"-F"}, want(func(o *ls.Options) { o.Indicator = ls.IndicatorClassify })},
		{"classify long", []string{"--classify"}, want(func(o *ls.Options) { o.Indicator = ls.IndicatorClassify })},
		{"slash", []string{"-p"}, want(func(o *ls.Options) { o.Indicator = ls.IndicatorSlash })},
		{"file type", []string{"--file-type"}, want(func(o *ls.Options) { o.Indicator = ls.IndicatorFileType })},
		{"-F then -p: last wins", []string{"-Fp"}, want(func(o *ls.Options) { o.Indicator = ls.IndicatorSlash })},
		{"-p then --file-type then -F", []string{"-p", "--file-type", "-F"}, want(func(o *ls.Options) { o.Indicator = ls.IndicatorClassify })},

		{"dereference", []string{"-L"}, want(func(o *ls.Options) { o.Dereference = true })},
		{"dereference long", []string{"--dereference"}, want(func(o *ls.Options) { o.Dereference = true })},
		{"dereference command line", []string{"-H"}, want(func(o *ls.Options) { o.DereferenceArgs = true })},
		{"dereference command line long", []string{"--dereference-command-line"}, want(func(o *ls.Options) { o.DereferenceArgs = true })},
		{"-L then -H: last wins", []string{"-LH"}, want(func(o *ls.Options) { o.DereferenceArgs = true })},
		{"-H then -L: last wins", []string{"-HL"}, want(func(o *ls.Options) { o.Dereference = true })},

		{"ctime", []string{"-c"}, want(func(o *ls.Options) { o.Time = ls.TimeChanged })},
		{"atime", []string{"-u"}, want(func(o *ls.Options) { o.Time = ls.TimeAccessed })},
		{"--time=atime", []string{"--time=atime"}, want(func(o *ls.Options) { o.Time = ls.TimeAccessed })},
		{"--time=access", []string{"--time=access"}, want(func(o *ls.Options) { o.Time = ls.TimeAccessed })},
		{"--time=use", []string{"--time=use"}, want(func(o *ls.Options) { o.Time = ls.TimeAccessed })},
		{"--time=ctime", []string{"--time=ctime"}, want(func(o *ls.Options) { o.Time = ls.TimeChanged })},
		{"--time=status", []string{"--time=status"}, want(func(o *ls.Options) { o.Time = ls.TimeChanged })},
		{"--time=mtime", []string{"-c", "--time=mtime"}, want(nil)},
		{"--time=birth", []string{"--time", "birth"}, want(func(o *ls.Options) { o.Time = ls.TimeBirth })},
		{"--time=creation", []string{"--time=creation"}, want(func(o *ls.Options) { o.Time = ls.TimeBirth })},
		{"-c then -u: last wins", []string{"-cu"}, want(func(o *ls.Options) { o.Time = ls.TimeAccessed })},
		{"--time=atime then -c: last wins", []string{"--time=atime", "-c"}, want(func(o *ls.Options) { o.Time = ls.TimeChanged })},

		{"time style full-iso", []string{"--time-style=full-iso"}, want(func(o *ls.Options) { o.TimeStyle = "full-iso" })},
		{"time style long-iso", []string{"--time-style=long-iso"}, want(func(o *ls.Options) { o.TimeStyle = "long-iso" })},
		{"time style iso", []string{"--time-style", "iso"}, want(func(o *ls.Options) { o.TimeStyle = "iso" })},
		{"time style locale", []string{"--time-style=locale"}, want(func(o *ls.Options) { o.TimeStyle = "locale" })},
		{"time style format", []string{"--time-style=+%F %T"}, want(func(o *ls.Options) { o.TimeStyle = "+%F %T" })},
		{"--full-time then --time-style: last wins", []string{"--full-time", "--time-style=iso"},
			want(func(o *ls.Options) { o.Format = ls.FormatLong; o.TimeStyle = "iso" })},
		{"--time-style then --full-time: last wins", []string{"--time-style=iso", "--full-time"},
			want(func(o *ls.Options) { o.Format = ls.FormatLong; o.TimeStyle = "full-iso" })},

		{"color bare means always", []string{"--color"}, want(func(o *ls.Options) { o.Color = ls.WhenAlways })},
		{"color always", []string{"--color=always"}, want(func(o *ls.Options) { o.Color = ls.WhenAlways })},
		{"color auto", []string{"--color=auto"}, want(func(o *ls.Options) { o.Color = ls.WhenAuto })},
		{"color tty", []string{"--color=tty"}, want(func(o *ls.Options) { o.Color = ls.WhenAuto })},
		{"color never", []string{"--color=never"}, want(nil)},
		{"color: last wins", []string{"--color", "--color=never", "--color=auto"}, want(func(o *ls.Options) { o.Color = ls.WhenAuto })},

		{"group directories first", []string{"--group-directories-first"}, want(func(o *ls.Options) { o.GroupDirectories = true })},

		{"width", []string{"-w", "40"}, want(func(o *ls.Options) { o.Width = 40 })},
		{"width long", []string{"--width=120"}, want(func(o *ls.Options) { o.Width = 120 })},
		{"width zero means no limit", []string{"-w0"}, want(func(o *ls.Options) { o.Width = -1 })},
		{"tabsize", []string{"-T", "4"}, want(func(o *ls.Options) { o.TabSize = 4 })},
		{"tabsize zero disables tabs", []string{"--tabsize=0"}, want(func(o *ls.Options) { o.TabSize = 0 })},

		{"quote name", []string{"-Q"}, want(func(o *ls.Options) { o.Quoting = ls.QuoteC })},
		{"quote name long", []string{"--quote-name"}, want(func(o *ls.Options) { o.Quoting = ls.QuoteC })},
		{"escape", []string{"-b"}, want(func(o *ls.Options) { o.Quoting = ls.QuoteEscape })},
		{"escape long", []string{"--escape"}, want(func(o *ls.Options) { o.Quoting = ls.QuoteEscape })},
		{"-Q then -b: last wins", []string{"-Qb"}, want(func(o *ls.Options) { o.Quoting = ls.QuoteEscape })},
		{"-b then -Q: last wins", []string{"-bQ"}, want(func(o *ls.Options) { o.Quoting = ls.QuoteC })},
		{"hide control chars", []string{"-q"}, want(func(o *ls.Options) { o.HideControlChars = true })},
		{"hide control chars long", []string{"--hide-control-chars"}, want(func(o *ls.Options) { o.HideControlChars = true })},

		{"ignore backups", []string{"-B"}, want(func(o *ls.Options) { o.IgnoreBackups = true })},
		{"ignore backups long", []string{"--ignore-backups"}, want(func(o *ls.Options) { o.IgnoreBackups = true })},
		{"ignore", []string{"-I", "*.o"}, want(func(o *ls.Options) { o.Ignore = []string{"*.o"} })},
		{"ignore repeats, commas are literal", []string{"-I", "*.o", "--ignore=a,b"}, want(func(o *ls.Options) { o.Ignore = []string{"*.o", "a,b"} })},
		{"hide", []string{"--hide=*.bak", "--hide", "tmp*"}, want(func(o *ls.Options) { o.Hide = []string{"*.bak", "tmp*"} })},

		{"combined -lah", []string{"-lah"}, want(func(o *ls.Options) { o.Format = ls.FormatLong; o.All = true; o.HumanReadable = true })},
		{"combined -ltrR", []string{"-ltrR"}, want(func(o *ls.Options) {
			o.Format, o.Sort, o.Reverse, o.Recursive = ls.FormatLong, ls.SortTime, true, true
		})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildOptions(%v) =\n%+v, want\n%+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string // prefix of the usage message
	}{
		{"bad sort word", []string{"--sort=bogus"}, "invalid argument 'bogus' for '--sort'\nValid arguments are:"},
		{"ambiguous sort word", []string{"--sort=n"}, "ambiguous argument 'n' for '--sort'"},
		{"bad time word", []string{"--time=bogus"}, "invalid argument 'bogus' for '--time'"},
		{"bad color word", []string{"--color=bogus"}, "invalid argument 'bogus' for '--color'"},
		{"bad time style", []string{"--time-style=bogus"}, "invalid argument 'bogus' for 'time style'"},
		{"bad width", []string{"-w", "wide"}, "invalid line width: 'wide'"},
		{"negative width", []string{"--width=-1"}, "invalid line width: '-1'"},
		{"bad tab size", []string{"-T", "x"}, "invalid tab size: 'x'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := options(t, tt.args...)
			var usage *core.UsageError
			if !errors.As(err, &usage) {
				t.Fatalf("buildOptions(%v) err = %v, want a usage error", tt.args, err)
			}
			if !strings.HasPrefix(usage.Msg, tt.want) {
				t.Fatalf("message = %q, want prefix %q", usage.Msg, tt.want)
			}
		})
	}
}

func TestSpec(t *testing.T) {
	s := Spec()
	if s.Name() != "lsx" || s.GNU != "ls" || s.UsageExit != 2 {
		t.Fatalf("Spec = name %q, GNU %q, usage exit %d", s.Name(), s.GNU, s.UsageExit)
	}

	m := Meta()
	if m.Use != "lsx [OPTION]... [FILE]..." || m.Short == "" || m.Long == "" || m.Permissions == "" || len(m.UseCases) == 0 {
		t.Fatalf("Meta is incomplete: %+v", m)
	}
	if n := len(m.Examples); n < 10 {
		t.Fatalf("got %d examples, want at least 10", n)
	}
	for _, e := range m.Examples {
		if e.Description == "" || e.Bash == "" || e.PowerShell == "" {
			t.Fatalf("incomplete example: %+v", e)
		}
	}
}

// TestCommand drives the real command line: -h must mean --human-readable, the
// flags must reach the engine, and a missing operand must surface as status 2.
func TestCommand(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	run := func(args ...string) (string, string, error) {
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
		c.SetArgs(args)
		err := c.Execute()
		return out.String(), errb.String(), err
	}

	if out, _, err := run("-h", "-a", "-m"); err != nil || out != "., ..\n" {
		t.Fatalf("lsx -h -a -m = %q, %v", out, err)
	}

	_, stderr, err := run("missing")
	var exit *core.ExitError
	if !errors.As(err, &exit) || exit.Code != 2 {
		t.Fatalf("err = %v, want exit status 2", err)
	}
	if want := "lsx: cannot access 'missing': No such file or directory\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}
