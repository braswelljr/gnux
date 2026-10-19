package rm_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rm"
)

// tree builds root/{a.txt, sub/{b.txt}, empty/} and returns root.
func tree(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "root")
	for _, d := range []string{"sub", "empty"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"a.txt", filepath.Join("sub", "b.txt")} {
		if err := os.WriteFile(filepath.Join(root, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestInteractiveRecursiveOrder checks GNU's post-order prompting: descend
// into a directory, then each child (directories depth first), then the
// directory itself. Entries are read in name order.
func TestInteractiveRecursiveOrder(t *testing.T) {
	root := tree(t)
	r, _, errb := newEngine(rm.Options{Recursive: true, Interactive: rm.PromptAlways})
	var asked []string
	r.PromptFn = func(q string) bool {
		asked = append(asked, strings.ReplaceAll(q, root, "R"))
		return true
	}
	if err := r.Run([]string{root}); err != nil {
		t.Fatalf("Run: %v (stderr %q)", err, errb.String())
	}
	sep := string(filepath.Separator)
	want := []string{
		"descend into directory 'R'?",
		"remove regular file 'R" + sep + "a.txt'?",
		"remove directory 'R" + sep + "empty'?",
		"descend into directory 'R" + sep + "sub'?",
		"remove regular file 'R" + sep + "sub" + sep + "b.txt'?",
		"remove directory 'R" + sep + "sub'?",
		"remove directory 'R'?",
	}
	if strings.Join(asked, "\n") != strings.Join(want, "\n") {
		t.Fatalf("prompts:\n%s\nwant:\n%s", strings.Join(asked, "\n"), strings.Join(want, "\n"))
	}
	mustExist(t, root, false)
}

// TestDecliningAChildKeepsItsParents: the rest of the tree goes, but every
// directory above a kept entry stays, and nothing is reported as an error.
func TestDecliningAChildKeepsItsParents(t *testing.T) {
	root := tree(t)
	r, _, errb := newEngine(rm.Options{Recursive: true, Interactive: rm.PromptAlways})
	r.PromptFn = func(q string) bool { return !strings.Contains(q, "b.txt") }
	if err := r.Run([]string{root}); err != nil {
		t.Fatalf("Run: %v (stderr %q)", err, errb.String())
	}
	mustExist(t, filepath.Join(root, "sub", "b.txt"), true)
	mustExist(t, filepath.Join(root, "a.txt"), false)
	mustExist(t, filepath.Join(root, "empty"), false)
	if errb.Len() != 0 {
		t.Fatalf("stderr = %q", errb.String())
	}
}

func TestDecliningToDescendKeepsTheTree(t *testing.T) {
	root := tree(t)
	r, _, _ := newEngine(rm.Options{Recursive: true, Interactive: rm.PromptAlways})
	asked := 0
	r.PromptFn = func(string) bool { asked++; return false }
	if err := r.Run([]string{root}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if asked != 1 {
		t.Fatalf("asked %d questions, want 1", asked)
	}
	mustExist(t, filepath.Join(root, "a.txt"), true)
}

// TestEmptyDirectoryIsNotDescended: there is nothing to descend into, so the
// only question is whether to remove it.
func TestEmptyDirectoryIsNotDescended(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "empty")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, opts := range []rm.Options{
		{Recursive: true, Interactive: rm.PromptAlways},
		{Dir: true, Interactive: rm.PromptAlways},
	} {
		r, _, _ := newEngine(opts)
		var asked []string
		r.PromptFn = func(q string) bool { asked = append(asked, q); return false }
		if err := r.Run([]string{dir}); err != nil {
			t.Fatalf("%+v: Run: %v", opts, err)
		}
		if want := "remove directory '" + dir + "'?"; len(asked) != 1 || asked[0] != want {
			t.Fatalf("%+v: prompts %q, want %q", opts, asked, want)
		}
		mustExist(t, dir, true)
	}
}

func TestVerboseRecursiveOrder(t *testing.T) {
	root := tree(t)
	r, out, _ := newEngine(rm.Options{Recursive: true, Verbose: true})
	if err := r.Run([]string{root}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	sep := string(filepath.Separator)
	want := strings.ReplaceAll(`removed 'R/a.txt'
removed directory 'R/empty'
removed 'R/sub/b.txt'
removed directory 'R/sub'
removed directory 'R'
`, "/", sep)
	if got := strings.ReplaceAll(out.String(), root, "R"); got != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", got, want)
	}
}

// TestFailureInsideTheTree: an entry that cannot be removed is reported,
// its siblings still go, and its ancestors stay.
func TestFailureInsideTheTree(t *testing.T) {
	root := tree(t)
	r, _, errb := newEngine(rm.Options{Recursive: true})
	r.RemoveFn = func(p string) error {
		if filepath.Base(p) == "b.txt" {
			return os.ErrPermission
		}
		return os.Remove(p)
	}
	err := r.Run([]string{root})
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "rmx: cannot remove '" + filepath.Join(root, "sub", "b.txt") + "': Permission denied\n"; errb.String() != want {
		t.Fatalf("stderr = %q; want %q", errb.String(), want)
	}
	mustExist(t, filepath.Join(root, "sub", "b.txt"), true)
	mustExist(t, filepath.Join(root, "a.txt"), false)
	mustExist(t, filepath.Join(root, "empty"), false)
}

// TestForceIgnoresEntriesThatVanish: with -f, an entry removed by someone
// else mid-walk is not an error (GNU's ENOENT rule), though its parent stays.
func TestForceIgnoresEntriesThatVanish(t *testing.T) {
	root := tree(t)
	r, _, errb := newEngine(rm.Options{Recursive: true, Force: true})
	r.RemoveFn = func(p string) error {
		if filepath.Base(p) == "a.txt" {
			_ = os.Remove(p)
			return os.ErrNotExist
		}
		return os.Remove(p)
	}
	if err := r.Run([]string{root}); err != nil || errb.Len() != 0 {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

// TestOneFileSystemWalksTheSameDevice: everything under a temporary
// directory is on one device, so --one-file-system removes it all.
func TestOneFileSystemWalksTheSameDevice(t *testing.T) {
	root := tree(t)
	r, _, errb := newEngine(rm.Options{Recursive: true, OneFileSystem: true})
	if err := r.Run([]string{root}); err != nil {
		t.Fatalf("Run: %v (stderr %q)", err, errb.String())
	}
	mustExist(t, root, false)
}
