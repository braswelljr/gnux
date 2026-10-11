package touch_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/touch"
)

// zone is a fixed, non-UTC location so tests prove zone-less dates are read
// in the clock's location rather than the machine's.
var zone = time.FixedZone("TEST", 2*60*60)

// clock is the instant the injected Now reports.
var clock = time.Date(2024, time.March, 10, 12, 30, 45, 0, zone)

// old is the timestamp files start with, so any change is visible.
var old = time.Date(2001, time.February, 3, 4, 5, 6, 0, time.UTC)

func newEngine(opts touch.Options) (*touch.Touch, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	tc := touch.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	tc.Now = func() time.Time { return clock }
	return tc, &out, &errb
}

// oldFile creates a file whose access and modification times are both old.
func oldFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	return path
}

func times(t *testing.T, path string) (atime, mtime time.Time) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return touch.AccessTime(info), info.ModTime()
}

func wantTimes(t *testing.T, path string, atime, mtime time.Time) {
	t.Helper()
	gotA, gotM := times(t, path)
	if !gotA.Equal(atime) {
		t.Errorf("atime of %q = %v, want %v", filepath.Base(path), gotA.UTC(), atime.UTC())
	}
	if !gotM.Equal(mtime) {
		t.Errorf("mtime of %q = %v, want %v", filepath.Base(path), gotM.UTC(), mtime.UTC())
	}
}

func TestCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new")

	tc, out, errb := newEngine(touch.Options{})
	if err := tc.Run([]string{path}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("size = %d, want an empty file", info.Size())
	}
	wantTimes(t, path, clock, clock)
	if out.Len() != 0 || errb.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q, want both empty", out.String(), errb.String())
	}
}

func TestUpdatesExistingFileWithoutTruncating(t *testing.T) {
	path := oldFile(t, t.TempDir(), "f")

	tc, _, errb := newEngine(touch.Options{})
	if err := tc.Run([]string{path}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	wantTimes(t, path, clock, clock)
	if data, _ := os.ReadFile(path); string(data) != "data" {
		t.Fatalf("contents = %q, want them untouched", data)
	}
}

func TestWhichTimes(t *testing.T) {
	tests := []struct {
		name         string
		opts         touch.Options
		atime, mtime time.Time
	}{
		{"default changes both", touch.Options{}, clock, clock},
		{"-a keeps mtime", touch.Options{Access: true}, clock, old},
		{"-m keeps atime", touch.Options{Modify: true}, old, clock},
		{"-a -m changes both", touch.Options{Access: true, Modify: true}, clock, clock},
		{"--time=access", touch.Options{Time: "access"}, clock, old},
		{"--time=atime", touch.Options{Time: "atime"}, clock, old},
		{"--time=use", touch.Options{Time: "use"}, clock, old},
		{"--time=modify", touch.Options{Time: "modify"}, old, clock},
		{"--time=mtime", touch.Options{Time: "mtime"}, old, clock},
		{"-a with --time=mtime changes both", touch.Options{Access: true, Time: "mtime"}, clock, clock},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := oldFile(t, t.TempDir(), "f")
			tc, _, errb := newEngine(tt.opts)
			if err := tc.Run([]string{path}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			wantTimes(t, path, tt.atime, tt.mtime)
		})
	}
}

func TestNoCreate(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	existing := oldFile(t, dir, "existing")

	tc, _, errb := newEngine(touch.Options{NoCreate: true})
	if err := tc.Run([]string{missing, existing}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	if _, err := os.Lstat(missing); err == nil {
		t.Fatal("-c created the file")
	}
	wantTimes(t, existing, clock, clock)
	if errb.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errb.String())
	}
}

func TestDate(t *testing.T) {
	tests := []struct {
		date string
		want time.Time
	}{
		{"2024-01-02", time.Date(2024, 1, 2, 0, 0, 0, 0, zone)},
		{"2024-01-02 15:04", time.Date(2024, 1, 2, 15, 4, 0, 0, zone)},
		{"2024-01-02 15:04:05", time.Date(2024, 1, 2, 15, 4, 5, 0, zone)},
		{"2024-01-02T15:04:05", time.Date(2024, 1, 2, 15, 4, 5, 0, zone)},
		{"2024-01-02T15:04:05Z", time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)},
		{"2024-01-02T15:04:05-05:00", time.Date(2024, 1, 2, 20, 4, 5, 0, time.UTC)},
		{"2024-01-02 15:04:05+01:00", time.Date(2024, 1, 2, 14, 4, 5, 0, time.UTC)},
		{"  2024-01-02  ", time.Date(2024, 1, 2, 0, 0, 0, 0, zone)},
		{"@0", time.Unix(0, 0)},
		{"@1700000000", time.Unix(1700000000, 0)},
		{"now", clock},
		{"NOW", clock},
	}
	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			path := oldFile(t, t.TempDir(), "f")
			tc, _, errb := newEngine(touch.Options{Date: tt.date})
			if err := tc.Run([]string{path}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			wantTimes(t, path, tt.want, tt.want)
		})
	}
}

func TestStamp(t *testing.T) {
	tests := []struct {
		stamp string
		want  time.Time
	}{
		{"01021504", time.Date(2024, 1, 2, 15, 4, 0, 0, zone)},
		{"01021504.05", time.Date(2024, 1, 2, 15, 4, 5, 0, zone)},
		{"9901021504", time.Date(1999, 1, 2, 15, 4, 0, 0, zone)},
		{"6901021504", time.Date(1969, 1, 2, 15, 4, 0, 0, zone)},
		{"6801021504", time.Date(2068, 1, 2, 15, 4, 0, 0, zone)},
		{"2301021504.59", time.Date(2023, 1, 2, 15, 4, 59, 0, zone)},
		{"202002291200", time.Date(2020, 2, 29, 12, 0, 0, 0, zone)},
		{"199912312359.59", time.Date(1999, 12, 31, 23, 59, 59, 0, zone)},
	}
	for _, tt := range tests {
		t.Run(tt.stamp, func(t *testing.T) {
			path := oldFile(t, t.TempDir(), "f")
			tc, _, errb := newEngine(touch.Options{Stamp: tt.stamp})
			if err := tc.Run([]string{path}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			wantTimes(t, path, tt.want, tt.want)
		})
	}
}

func TestReference(t *testing.T) {
	dir := t.TempDir()
	refA := time.Date(2010, 5, 6, 7, 8, 9, 0, time.UTC)
	refM := time.Date(2011, 6, 7, 8, 9, 10, 0, time.UTC)
	ref := filepath.Join(dir, "ref")
	if err := os.WriteFile(ref, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(ref, refA, refM); err != nil {
		t.Fatal(err)
	}
	date := time.Date(2024, 1, 2, 0, 0, 0, 0, zone)

	tests := []struct {
		name         string
		opts         touch.Options
		atime, mtime time.Time
	}{
		{"copies both times", touch.Options{Reference: ref}, refA, refM},
		{"-a copies only atime", touch.Options{Reference: ref, Access: true}, refA, old},
		{"-m copies only mtime", touch.Options{Reference: ref, Modify: true}, old, refM},
		{"date 'now' is the reference", touch.Options{Reference: ref, Date: "now"}, refA, refM},
		{"absolute date overrides", touch.Options{Reference: ref, Date: "2024-01-02"}, date, date},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := oldFile(t, t.TempDir(), "f")
			tc, _, errb := newEngine(tt.opts)
			if err := tc.Run([]string{path}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			wantTimes(t, path, tt.atime, tt.mtime)
		})
	}

	t.Run("missing reference", func(t *testing.T) {
		path := filepath.Join(dir, "never")
		missing := filepath.Join(dir, "noref")
		tc, _, errb := newEngine(touch.Options{Reference: missing})
		if err := tc.Run([]string{path}); !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "touchx: failed to get attributes of '" + missing + "': No such file or directory\n"; errb.String() != want {
			t.Fatalf("stderr = %q, want %q", errb.String(), want)
		}
		if _, err := os.Lstat(path); err == nil {
			t.Fatal("file created despite the missing reference")
		}
	})
}

func TestCannotTouch(t *testing.T) {
	dir := t.TempDir()
	orphan := filepath.Join(dir, "a", "b")
	good := filepath.Join(dir, "good")

	tc, _, errb := newEngine(touch.Options{})
	if err := tc.Run([]string{orphan, good}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "touchx: cannot touch '" + orphan + "': No such file or directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	// One bad operand does not stop the rest.
	if _, err := os.Stat(good); err != nil {
		t.Fatalf("later operand not created: %v", err)
	}
}

func TestTouchDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "d")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	tc, _, errb := newEngine(touch.Options{Date: "2024-01-02"})
	if err := tc.Run([]string{dir}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	want := time.Date(2024, 1, 2, 0, 0, 0, 0, zone)
	if _, mtime := times(t, dir); !mtime.Equal(want) {
		t.Fatalf("mtime = %v, want %v", mtime, want)
	}
}

func TestFollowsSymlinkByDefault(t *testing.T) {
	dir := t.TempDir()
	target := oldFile(t, dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	tc, _, errb := newEngine(touch.Options{})
	if err := tc.Run([]string{link}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	wantTimes(t, target, clock, clock)
}

func TestNoDereference(t *testing.T) {
	dir := t.TempDir()
	target := oldFile(t, dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	tc, _, errb := newEngine(touch.Options{NoDereference: true})
	err := tc.Run([]string{link})

	// Whatever the platform can do, the link's target must never be touched.
	wantTimes(t, target, old, old)

	if err != nil {
		// Platforms that cannot set a link's own times must say so.
		if want := "touchx: setting times of '" + link + "': Operation not supported\n"; errb.String() != want {
			t.Fatalf("stderr = %q, want %q", errb.String(), want)
		}
		return
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(clock) {
		t.Fatalf("link mtime = %v, want %v", info.ModTime().UTC(), clock.UTC())
	}
}

func TestNoDereferenceOnRegularFiles(t *testing.T) {
	dir := t.TempDir()
	path := oldFile(t, dir, "f")
	missing := filepath.Join(dir, "missing")

	tc, _, errb := newEngine(touch.Options{NoDereference: true, Modify: true})
	if err := tc.Run([]string{path, missing}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	wantTimes(t, path, old, clock)
	// -h implies -c: there is no link to act on, so nothing is created.
	if _, err := os.Lstat(missing); err == nil {
		t.Fatal("-h created the file")
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts touch.Options
		args []string
		want string
	}{
		{"missing operand", touch.Options{}, nil, "missing file operand"},
		{"bad --time", touch.Options{Time: "birth"}, []string{"f"}, "invalid argument 'birth' for '--time'"},
		{"stamp and date", touch.Options{Stamp: "01021504", Date: "now"}, []string{"f"}, "cannot specify times from more than one source"},
		{"stamp and reference", touch.Options{Stamp: "01021504", Reference: "r"}, []string{"f"}, "cannot specify times from more than one source"},
		{"free-form date", touch.Options{Date: "next tuesday"}, []string{"f"}, "invalid date format 'next tuesday'"},
		{"date with bad month", touch.Options{Date: "2024-13-01"}, []string{"f"}, "invalid date format '2024-13-01'"},
		{"date with bad day", touch.Options{Date: "2024-02-30"}, []string{"f"}, "invalid date format '2024-02-30'"},
		{"bad epoch", touch.Options{Date: "@soon"}, []string{"f"}, "invalid date format '@soon'"},
		{"stamp too short", touch.Options{Stamp: "0102150"}, []string{"f"}, "invalid date format '0102150'"},
		{"stamp with letters", touch.Options{Stamp: "0102150x"}, []string{"f"}, "invalid date format '0102150x'"},
		{"stamp with bad month", touch.Options{Stamp: "13021504"}, []string{"f"}, "invalid date format '13021504'"},
		{"stamp with bad day", touch.Options{Stamp: "202302291200"}, []string{"f"}, "invalid date format '202302291200'"},
		{"stamp with bad hour", touch.Options{Stamp: "01022504"}, []string{"f"}, "invalid date format '01022504'"},
		{"stamp with bad seconds", touch.Options{Stamp: "01021504.5"}, []string{"f"}, "invalid date format '01021504.5'"},
		{"stamp with seconds out of range", touch.Options{Stamp: "01021504.61"}, []string{"f"}, "invalid date format '01021504.61'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			tc, _, _ := newEngine(tt.opts)
			err := tc.Run(tt.args)
			var uerr *core.UsageError
			if !errors.As(err, &uerr) {
				t.Fatalf("err = %v, want UsageError", err)
			}
			if uerr.Msg != tt.want {
				t.Fatalf("usage = %q, want %q", uerr.Msg, tt.want)
			}
			if _, err := os.Lstat("f"); err == nil {
				t.Fatal("file created despite the usage error")
			}
		})
	}
}

func TestDefaultClock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	var out, errb bytes.Buffer
	before := time.Now().Add(-2 * time.Second)

	tc := touch.New(touch.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	if err := tc.Run([]string{path}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	_, mtime := times(t, path)
	if after := time.Now().Add(2 * time.Second); mtime.Before(before) || mtime.After(after) {
		t.Fatalf("mtime = %v, want it between %v and %v", mtime, before, after)
	}
}
