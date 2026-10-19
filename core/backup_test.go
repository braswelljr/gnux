package core_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

func TestParseBackupControl(t *testing.T) {
	for word, want := range map[string]core.BackupControl{
		"": core.ExistingBackup, "none": core.NoBackup, "off": core.NoBackup,
		"simple": core.SimpleBackup, "never": core.SimpleBackup, "s": core.SimpleBackup,
		"existing": core.ExistingBackup, "nil": core.ExistingBackup, "e": core.ExistingBackup,
		"numbered": core.NumberedBackup, "t": core.NumberedBackup, "nu": core.NumberedBackup,
		"ne": core.SimpleBackup, "no": core.NoBackup,
	} {
		got, err := core.ParseBackupControl("backup type", word)
		if err != nil || got != want {
			t.Errorf("%q: got %v, %v; want %v", word, got, err, want)
		}
	}

	const valid = "\nValid arguments are:\n  - 'none', 'off'\n  - 'simple', 'never'\n  - 'existing', 'nil'\n  - 'numbered', 't'"
	for word, want := range map[string]string{
		"bogus": "invalid argument 'bogus' for 'backup type'" + valid,
		"n":     "ambiguous argument 'n' for 'backup type'" + valid,
	} {
		_, err := core.ParseBackupControl("backup type", word)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != want {
			t.Errorf("%q: err = %v, want %q", word, err, want)
		}
	}
}

func TestBackupSuffix(t *testing.T) {
	for suffix, want := range map[string]string{
		"": "~", ".bak": ".bak", "x/y": "~", "/x": "~", "x/": "x/", "~~": "~~",
	} {
		if got := (core.Backup{Control: core.SimpleBackup, Suffix: suffix}).SuffixInUse(); got != want {
			t.Errorf("%q: suffix %q, want %q", suffix, got, want)
		}
	}
}

func TestMakeBackup(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	backup := func(b core.Backup, want string) {
		t.Helper()
		write("f", want)
		got, err := b.MakeBackup(f)
		if err != nil || got != filepath.Join(dir, want) {
			t.Fatalf("%+v: backup = %q, %v; want %q", b, got, err, want)
		}
		if data, _ := os.ReadFile(got); string(data) != want {
			t.Fatalf("%s holds %q", got, data)
		}
		if _, err := os.Lstat(f); !os.IsNotExist(err) {
			t.Fatal("the original is still there")
		}
	}

	backup(core.Backup{Control: core.SimpleBackup}, "f~")
	backup(core.Backup{Control: core.SimpleBackup}, "f~") // replaces the old one
	backup(core.Backup{Control: core.ExistingBackup}, "f~")
	backup(core.Backup{Control: core.SimpleBackup, Suffix: ".orig"}, "f.orig")
	backup(core.Backup{Control: core.NumberedBackup}, "f.~1~")
	backup(core.Backup{Control: core.ExistingBackup}, "f.~2~")
	write("f.~9~", "")
	write("f.~09~", "") // a leading zero is not a version
	write("f.~x~", "")
	backup(core.Backup{Control: core.NumberedBackup}, "f.~10~")
	backup(core.Backup{Control: core.NumberedBackup}, "f.~11~")

	// Nothing to back up is no failure.
	if got, err := (core.Backup{Control: core.NumberedBackup}).MakeBackup(filepath.Join(dir, "missing")); got != "" || err != nil {
		t.Fatalf("missing: %q, %v", got, err)
	}
	if got, err := (core.Backup{}).MakeBackup(f); got != "" || err != nil {
		t.Fatalf("NoBackup: %q, %v", got, err)
	}
}

func TestBackupNameFitsTheFileSystem(t *testing.T) {
	long := strings.Repeat("x", 255)
	if got := (core.Backup{Control: core.SimpleBackup}).BackupName(long); got != strings.Repeat("x", 254)+"~" {
		t.Fatalf("len %d: %q", len(got), got)
	}
	if got := (core.Backup{Control: core.SimpleBackup, Suffix: ".bak"}).BackupName("dir/" + long[:253]); got != "dir/"+long[:253]+"~" {
		t.Fatalf("got %q", got)
	}
}

func TestBackupWouldDestroy(t *testing.T) {
	dir := t.TempDir()
	f, back := filepath.Join(dir, "f"), filepath.Join(dir, "f~")
	if err := os.WriteFile(back, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	si, _ := os.Stat(back)
	simple := core.Backup{Control: core.SimpleBackup}
	if !simple.BackupWouldDestroy(back, si, f) {
		t.Fatal("copying f~ onto f would rename f over f~")
	}
	if (core.Backup{Control: core.NumberedBackup}).BackupWouldDestroy(back, si, f) {
		t.Fatal("a numbered backup never replaces f~")
	}
}
