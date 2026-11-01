package df

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/fsstat"
)

// A small machine: a root disk, a /home on another disk mounted twice (a
// bind mount), a network share, /proc, an empty drive and a tmpfs with no
// blocks.
var (
	mounts = []fsstat.Mount{
		{Source: "/dev/sda1", Target: "/", Type: "ext4"},
		{Source: "proc", Target: "/proc", Type: "proc", Dummy: true},
		{Source: "/dev/sdb1", Target: "/home", Type: "xfs"},
		{Source: "/dev/sdb1", Target: "/srv/home", Type: "xfs"},
		{Source: "nas:/export", Target: "/mnt/nas", Type: "nfs", Remote: true},
		{Source: "/dev/sr0", Target: "/media/cdrom", Type: "iso9660"},
		{Source: "none", Target: "/run/empty", Type: "tmpfs"},
	}
	stats = map[string]fsstat.FS{
		"/":          {TypeName: "ext2/ext3", BlockSize: 4096, Blocks: 1000000, Bfree: 400000, Bavail: 350000, Files: 65536, Ffree: 60000, FilesKnown: true},
		"/proc":      {BlockSize: 4096},
		"/home":      {BlockSize: 4096, Blocks: 2000000, Bfree: 1500000, Bavail: 1500000, Files: 1000, Ffree: 999, FilesKnown: true},
		"/srv/home":  {BlockSize: 4096, Blocks: 2000000, Bfree: 1500000, Bavail: 1500000, Files: 1000, Ffree: 999, FilesKnown: true},
		"/mnt/nas":   {BlockSize: 1024, Blocks: 3, Bfree: 2, Bavail: 2},
		"/run/empty": {BlockSize: 4096},
	}
	devs = map[string]uint64{"/": 1, "/proc": 2, "/home": 3, "/srv/home": 3, "/mnt/nas": 4, "/media/cdrom": 5, "/run/empty": 6}
)

func run(t *testing.T, opts Options, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	d := New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	d.Getenv = func(string) string { return "" }
	d.mounts = func() ([]fsstat.Mount, error) { return mounts, nil }
	d.statfs = func(p string) (fsstat.FS, error) {
		if s, ok := stats[p]; ok {
			return s, nil
		}
		return fsstat.FS{}, &os.PathError{Op: "statfs", Path: p, Err: fs.ErrNotExist}
	}
	d.devOf = func(p string) (uint64, error) {
		if dev, ok := devs[p]; ok {
			return dev, nil
		}
		return 0, fs.ErrNotExist
	}
	err = d.Run(args)
	return out.String(), errb.String(), err
}

func TestDefault(t *testing.T) {
	out, errb, err := run(t, Options{})
	if err != nil {
		t.Fatalf("%v %q", err, errb)
	}
	want := "" +
		"Filesystem     1K-blocks    Used Available Use% Mounted on\n" +
		"/dev/sda1        4000000 2400000   1400000  64% /\n" +
		"/dev/sdb1        8000000 2000000   6000000  25% /home\n" +
		"nas:/export            3       1         2  34% /mnt/nas\n"
	if out != want {
		t.Fatalf("got\n%s\nwant\n%s", out, want)
	}
}

func TestHumanPosixInodesType(t *testing.T) {
	out, _, _ := run(t, Options{BlockSize: "human-readable", Local: true})
	want := "" +
		"Filesystem      Size  Used Avail Use% Mounted on\n" +
		"/dev/sda1       3.9G  2.3G  1.4G  64% /\n" +
		"/dev/sdb1       7.7G  2.0G  5.8G  25% /home\n"
	if out != want {
		t.Fatalf("-hl: got\n%s\nwant\n%s", out, want)
	}
	out, _, _ = run(t, Options{Portability: true, Local: true})
	if !strings.HasPrefix(out, "Filesystem     1024-blocks    Used Available Capacity Mounted on\n") {
		t.Fatalf("-P: %q", out)
	}
	out, _, _ = run(t, Options{Inodes: true, Types: []string{"ext4"}})
	want = "" +
		"Filesystem     Inodes IUsed IFree IUse% Mounted on\n" +
		"/dev/sda1       65536  5536 60000    9% /\n"
	if out != want {
		t.Fatalf("-i -t ext4: got\n%s\nwant\n%s", out, want)
	}
	out, _, _ = run(t, Options{PrintType: true, ExcludeTypes: []string{"nfs", "xfs"}})
	want = "" +
		"Filesystem     Type 1K-blocks    Used Available Use% Mounted on\n" +
		"/dev/sda1      ext4   4000000 2400000   1400000  64% /\n"
	if out != want {
		t.Fatalf("-T -x: got\n%s\nwant\n%s", out, want)
	}
}

func TestAll(t *testing.T) {
	out, _, _ := run(t, Options{All: true})
	for _, target := range []string{"/proc", "/srv/home", "/media/cdrom", "/run/empty"} {
		if !strings.Contains(out, " "+target+"\n") {
			t.Errorf("-a does not list %s:\n%s", target, out)
		}
	}
	if !strings.Contains(out, "/dev/sr0               -       -         -    - /media/cdrom\n") {
		t.Errorf("-a shows the empty drive as unknown:\n%s", out)
	}
}

func TestOutputAndTotal(t *testing.T) {
	out, _, err := run(t, Options{OutputSet: true, Output: []string{"source", "fstype", "size", "pcent", "target"}, Total: true, Local: true})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 4 || !strings.HasPrefix(lines[3], "total") || !strings.Contains(lines[3], "12000000") {
		t.Fatalf("--output --total:\n%s", out)
	}
	for _, tt := range []struct {
		opts Options
		msg  string
	}{
		{Options{OutputSet: true, Output: []string{"bogus"}}, "option --output: field 'bogus' unknown"},
		{Options{OutputSet: true, Output: []string{"size", "size"}}, "option --output: field 'size' used more than once"},
		{Options{OutputSet: true, Inodes: true}, "options -i and --output are mutually exclusive"},
		{Options{OutputSet: true, PrintType: true}, "options -T and --output are mutually exclusive"},
		{Options{BlockSize: "x"}, "invalid --block-size argument 'x'"},
	} {
		_, _, err := run(t, tt.opts)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.msg {
			t.Errorf("%+v: %v, want %q", tt.opts, err, tt.msg)
		}
	}
}

func TestErrors(t *testing.T) {
	_, errb, err := run(t, Options{Types: []string{"xfs"}, ExcludeTypes: []string{"xfs"}})
	if !errors.Is(err, core.ErrFailed) || errb != "dfx: file system type 'xfs' both selected and excluded\n" {
		t.Fatalf("-t xfs -x xfs: %v %q", err, errb)
	}
	_, errb, err = run(t, Options{Types: []string{"zfs"}})
	if !errors.Is(err, core.ErrFailed) || errb != "dfx: no file systems processed\n" {
		t.Fatalf("-t zfs: %v %q", err, errb)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	_, errb, err = run(t, Options{}, missing)
	if !errors.Is(err, core.ErrFailed) || !strings.HasPrefix(errb, "dfx: "+missing+": ") {
		t.Fatalf("missing operand: %v %q", err, errb)
	}
}

func TestPercentAndHeader(t *testing.T) {
	for _, tt := range []struct {
		used, avail int64
		want        string
	}{
		{1, 2, "34%"}, {0, 10, "0%"}, {10, 0, "100%"}, {0, 0, "-"}, {5, -1, "125%"},
	} {
		if got := percent(tt.used, tt.avail); got != tt.want {
			t.Errorf("percent(%d, %d) = %s, want %s", tt.used, tt.avail, got, tt.want)
		}
	}
	for size, want := range map[uint64]string{1024: "1K-blocks", 1 << 20: "1M-blocks", 512: "512B-blocks", 1000: "1kB-blocks", 1: "1B-blocks"} {
		if got := blocksHeader(size, 0); got != want {
			t.Errorf("blocksHeader(%d) = %s, want %s", size, got, want)
		}
	}
}

func TestRealSystem(t *testing.T) {
	var out, errb bytes.Buffer
	err := New(Options{}, core.Streams{Out: &out, Err: &errb}).Run([]string{t.TempDir()})
	if errors.Is(err, core.ErrFailed) && strings.Contains(errb.String(), "not supported") {
		t.Skip("no file system statistics here")
	}
	if err != nil || !strings.HasPrefix(out.String(), "Filesystem") || strings.Count(out.String(), "\n") != 2 {
		t.Fatalf("%v %q %q", err, out.String(), errb.String())
	}
}

func TestPretty(t *testing.T) {
	out, errb, err := run(t, Options{Pretty: true})
	if err != nil {
		t.Fatalf("%v %q", err, errb)
	}
	// Without color: a bar (in eighths of a cell) after each percentage, the
	// rest as GNU has it.
	want := "" +
		"Filesystem     1K-blocks    Used Available Use%              Mounted on\n" +
		"/dev/sda1        4000000 2400000   1400000  64% ███████▋░░░░ /\n" +
		"/dev/sdb1        8000000 2000000   6000000  25% ███░░░░░░░░░ /home\n" +
		"nas:/export            3       1         2  34% ████▏░░░░░░░ /mnt/nas\n"
	if out != want {
		t.Fatalf("got\n%s\nwant\n%s", out, want)
	}

	out, _, _ = run(t, Options{Pretty: true, Color: true})
	if !strings.Contains(out, "\x1b[4mFilesystem\x1b[0m") || !strings.Contains(out, "\x1b[32m64%\x1b[0m") {
		t.Fatalf("colored:\n%q", out)
	}
	// Color alone, without the view, is not a GNU option and does nothing.
	if out, _, _ := run(t, Options{Color: true}); strings.Contains(out, "\x1b") {
		t.Fatalf("Color without Pretty:\n%q", out)
	}
}
