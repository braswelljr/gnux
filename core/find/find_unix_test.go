//go:build unix

package find_test

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/find"
	"github.com/braswelljr/gnux/core/internal/sysstat"
)

// linked builds, and enters, a tree with symbolic links:
//
//	file (4)  real/inner (2)
//	to-file -> file   to-dir -> real   broken -> nowhere
func linked(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	write(t, "file", 4)
	write(t, filepath.Join("real", "inner"), 2)
	for name, target := range map[string]string{"to-file": "file", "to-dir": "real", "broken": "nowhere"} {
		if err := os.Symlink(target, name); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSymlinkModes(t *testing.T) {
	linked(t)
	// -P, the default: links are files of their own and are not entered.
	physical := lines(".", "./broken", "./file", "./real", "./real/inner", "./to-dir", "./to-file")
	check(t, physical, ".")
	check(t, physical, "-P", ".")
	check(t, physical, "-L", "-H", "-P", ".")
	check(t, lines("./broken", "./to-dir", "./to-file"), ".", "-type", "l")
	check(t, lines("to-dir"), "to-dir")

	// -H follows the links named on the command line only.
	check(t, lines("to-dir", "to-dir/inner"), "-H", "to-dir")
	check(t, physical, "-H", ".")
	check(t, lines("to-dir"), "-P", "-H", "to-dir", "-type", "d")
	check(t, lines("broken"), "-H", "broken", "-type", "l")

	// -L follows them all; a broken link stays a link.
	check(t, lines(".", "./broken", "./file", "./real", "./real/inner", "./to-dir", "./to-dir/inner", "./to-file"), "-L", ".")
	check(t, lines("./broken"), "-L", ".", "-type", "l")
	check(t, lines("./file", "./real/inner", "./to-dir/inner", "./to-file"), "-H", "-L", ".", "-type", "f")
	check(t, lines(".", "./real", "./to-dir"), "-L", ".", "-type", "d")
	// -follow is -L from where it stands.
	check(t, lines("./file", "./real/inner", "./to-dir/inner", "./to-file"), ".", "-follow", "-type", "f")
}

func TestXtypeAndLname(t *testing.T) {
	linked(t)
	// -xtype looks at what -type does not: the target without -L...
	check(t, lines("./file", "./real/inner", "./to-file"), ".", "-xtype", "f")
	check(t, lines(".", "./real", "./to-dir"), ".", "-xtype", "d")
	check(t, lines("./broken"), ".", "-xtype", "l")
	// ...and the link itself with it.
	check(t, lines("./broken", "./to-dir", "./to-file"), "-L", ".", "-xtype", "l")

	check(t, lines("./to-dir", "./to-file"), ".", "-lname", "[fr]*")
	check(t, lines("./to-file"), ".", "-ilname", "FILE")
	check(t, lines("./broken"), ".", "-lname", "no*")
	// Under -L only a broken link is still a link.
	check(t, lines("./broken"), "-L", ".", "-lname", "*")

	check(t, "broken l N nowhere\nfile f f \nto-dir l d real\nto-file l f file\n",
		".", "-maxdepth", "1", "!", "-type", "d", "-printf", `%f %y %Y %l\n`)
	check(t, "to-dir d d \n", "-L", ".", "-name", "to-dir", "-printf", `%f %y %Y %l\n`)
}

func TestSymlinkLoop(t *testing.T) {
	t.Chdir(t.TempDir())
	write(t, filepath.Join("a", "b", "f"), 1)
	if err := os.Symlink("..", filepath.Join("a", "b", "up")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("self", "self"); err != nil {
		t.Fatal(err)
	}
	check(t, lines("a", "a/b", "a/b/f", "a/b/up"), "a")

	r := run(t, nil, "-L", "a")
	if !errors.Is(r.e, core.ErrFailed) || r.out != lines("a", "a/b", "a/b/f") {
		t.Fatalf("result = %+v", r)
	}
	if want := "findx: File system loop detected; 'a/b/up' is part of the same file system loop as 'a'.\n"; r.err != want {
		t.Fatalf("stderr = %q, want %q", r.err, want)
	}

	// A link that points at itself cannot be followed: diagnosed, then
	// treated as the link it is.
	r = run(t, nil, "-L", "self", "-printf", `%p %y %Y\n`)
	if !errors.Is(r.e, core.ErrFailed) || r.out != "self l L\n" {
		t.Fatalf("result = %+v", r)
	}
	if want := "findx: 'self': Too many levels of symbolic links\n"; r.err != want {
		t.Fatalf("stderr = %q, want %q", r.err, want)
	}
}

func TestNewerFollowsReference(t *testing.T) {
	linked(t)
	old, recent := now.Add(-72*time.Hour), now.Add(-time.Hour)
	if err := os.Chtimes("file", old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join("real", "inner"), recent, recent); err != nil {
		t.Fatal(err)
	}
	// Without -H or -L the reference is the link itself, made just now.
	check(t, "", "real", "-type", "f", "-newer", "to-file")
	check(t, lines("real/inner"), "-H", "real", "-type", "f", "-newer", "to-file")
	check(t, lines("real/inner"), "-L", "real", "-type", "f", "-newer", "to-file")
	check(t, lines("./file", "./to-file"), "-L", ".", "-samefile", "to-file")
	check(t, lines("./to-file"), ".", "-samefile", "to-file")
}

func TestPerm(t *testing.T) {
	t.Chdir(t.TempDir())
	for name, mode := range map[string]os.FileMode{
		"m644": 0o644, "m600": 0o600, "m755": 0o755, "m000": 0, "m4755": 0o755 | os.ModeSetuid, "m1777": 0o777 | os.ModeSticky,
	} {
		write(t, name, 1)
		if err := os.Chmod(name, mode); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		mode string
		want []string
	}{
		{"644", []string{"./m644"}},
		{"0644", []string{"./m644"}},
		{"u=rw,g=r,o=r", []string{"./m644"}},
		{"u=rw,go=r", []string{"./m644"}},
		{"a=r,u+w", []string{"./m644"}},
		{"=r,u+w", []string{"./m644"}},
		{"+r,u+w", []string{"./m644"}},
		{"u=rwx,g=u-w,o=g", []string{"./m755"}},
		{"0", []string{"./m000"}},
		{"4755", []string{"./m4755"}},
		{"u=rwxs,go=rx", []string{"./m4755"}},
		{"a=rwx,+t", []string{"./m1777"}},
		// -MODE: all of these bits.
		{"-644", []string{"./m1777", "./m4755", "./m644", "./m755"}},
		{"-u+x", []string{"./m1777", "./m4755", "./m755"}},
		{"-g+w,o+w", []string{"./m1777"}},
		{"-4000", []string{"./m4755"}},
		{"-u+s", []string{"./m4755"}},
		{"-1000", []string{"./m1777"}},
		{"-0", []string{"./m000", "./m1777", "./m4755", "./m600", "./m644", "./m755"}},
		// /MODE: any of them.
		{"/022", []string{"./m1777"}},
		{"/111", []string{"./m1777", "./m4755", "./m755"}},
		{"/u+x,g+w", []string{"./m1777", "./m4755", "./m755"}},
		{"/044", []string{"./m1777", "./m4755", "./m644", "./m755"}},
		{"/6000", []string{"./m4755"}},
		{"/000", []string{"./m000", "./m1777", "./m4755", "./m600", "./m644", "./m755"}},
		{"-a=x,u-x", []string{"./m1777", "./m4755", "./m755"}},
		{"u+X", []string{"./m000"}},
	}
	for _, tt := range tests {
		check(t, lines(tt.want...), ".", "-type", "f", "-perm", tt.mode)
	}
	check(t, "m000 0 ---------- 0\nm1777 1777 -rwxrwxrwt 01777\nm4755 4755 -rwsr-xr-x 04755\nm600 600 -rw------- 0600\nm644 644 -rw-r--r-- 0644\nm755 755 -rwxr-xr-x 0755\n",
		".", "-type", "f", "-printf", `%f %m %M %#m\n`)
}

func TestAccess(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root may do anything")
	}
	t.Chdir(t.TempDir())
	for name, mode := range map[string]os.FileMode{"r": 0o400, "w": 0o200, "x": 0o100, "rwx": 0o700, "none": 0} {
		write(t, name, 1)
		if err := os.Chmod(name, mode); err != nil {
			t.Fatal(err)
		}
	}
	check(t, lines("./r", "./rwx"), ".", "-type", "f", "-readable")
	check(t, lines("./rwx", "./w"), ".", "-type", "f", "-writable")
	check(t, lines("./rwx", "./x"), ".", "-type", "f", "-executable")
	check(t, lines("./none"), ".", "-type", "f", "!", "-readable", "!", "-writable", "!", "-executable")
}

func TestUnreadableDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root may read anything")
	}
	t.Chdir(t.TempDir())
	write(t, filepath.Join("a", "locked", "secret"), 1)
	write(t, filepath.Join("a", "z"), 1)
	if err := os.Chmod(filepath.Join("a", "locked"), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join("a", "locked"), 0o755) })

	// The directory is diagnosed, and the walk carries on past it.
	r := run(t, nil, "a")
	if !errors.Is(r.e, core.ErrFailed) || r.out != lines("a", "a/locked", "a/z") {
		t.Fatalf("result = %+v", r)
	}
	if want := "findx: 'a/locked': Permission denied\n"; r.err != want {
		t.Fatalf("stderr = %q, want %q", r.err, want)
	}

	r = run(t, nil, "a", "-empty")
	if !errors.Is(r.e, core.ErrFailed) || r.out != "" {
		t.Fatalf("result = %+v", r)
	}
	if want := "findx: 'a/locked': Permission denied\nfindx: 'a/locked': Permission denied\n"; r.err != want {
		t.Fatalf("stderr = %q, want %q", r.err, want)
	}
}

func TestOwners(t *testing.T) {
	tree(t)
	info, err := os.Stat("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	st := sysstat.Of(info)
	uid, gid := strconv.FormatInt(st.UID, 10), strconv.FormatInt(st.GID, 10)
	other := strconv.FormatInt(st.UID+12345, 10)
	want := lines("./a.txt")
	none := ""

	check(t, want, ".", "-name", "a.txt", "-uid", uid)
	check(t, want, ".", "-name", "a.txt", "-gid", gid)
	check(t, none, ".", "-name", "a.txt", "-uid", other)
	check(t, want, ".", "-name", "a.txt", "-uid", "-"+other)
	check(t, none, ".", "-name", "a.txt", "-uid", "+"+other)
	check(t, want, ".", "-name", "a.txt", "-gid", "-"+strconv.FormatInt(st.GID+1, 10), "!", "-gid", "+"+gid)
	// A number is accepted where a name is wanted.
	check(t, want, ".", "-name", "a.txt", "-user", uid)
	check(t, want, ".", "-name", "a.txt", "-group", gid)
	check(t, none, ".", "-name", "a.txt", "-user", other)

	userName, groupName := uid, gid
	if u, err := user.LookupId(uid); err == nil {
		userName = u.Username
		check(t, want, ".", "-name", "a.txt", "-user", u.Username)
		check(t, none, ".", "-name", "a.txt", "-nouser")
	} else {
		check(t, want, ".", "-name", "a.txt", "-nouser")
	}
	if g, err := user.LookupGroupId(gid); err == nil {
		groupName = g.Name
		check(t, want, ".", "-name", "a.txt", "-group", g.Name)
		check(t, none, ".", "-name", "a.txt", "-nogroup")
	} else {
		check(t, want, ".", "-name", "a.txt", "-nogroup")
	}
	check(t, fmt.Sprintf("%s %s %s %s\n", userName, groupName, uid, gid), "a.txt", "-printf", `%u %g %U %G\n`)
}

func TestLinksAndInodes(t *testing.T) {
	tree(t)
	if err := os.Link("a.txt", "hard"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	st := sysstat.Of(info)
	ino := strconv.FormatUint(st.Ino, 10)

	check(t, lines("./a.txt", "./hard"), ".", "-type", "f", "-links", "2")
	check(t, lines("./a.txt", "./hard"), ".", "-type", "f", "-links", "+1")
	check(t, lines("./.hidden", "./b.log", "./dir/c.txt", "./dir/sub/d.TXT"), ".", "-type", "f", "-links", "-2")
	check(t, lines("./a.txt", "./hard"), ".", "-inum", ino)
	check(t, lines("./a.txt", "./hard"), ".", "-samefile", "hard")
	check(t, "2 "+ino+" "+strconv.FormatUint(st.Dev, 10)+"\n", "hard", "-printf", `%n %i %D\n`)
	check(t, lines("."), ".", "-maxdepth", "0", "-xdev")

	r := run(t, nil, "dir/c.txt", "-printf", `%b %k %S\n`)
	fields := strings.Fields(r.out)
	if r.e != nil || len(fields) != 3 {
		t.Fatalf("result = %+v", r)
	}
	if b, _ := strconv.Atoi(fields[0]); b < 4 {
		t.Fatalf("%%b = %q for a 2000-byte file", fields[0])
	}
}

func TestLs(t *testing.T) {
	tree(t)
	if err := os.Symlink("a.txt", "link to a"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod("a.txt", 0o640); err != nil {
		t.Fatal(err)
	}
	recent, old := now.Add(-24*time.Hour), time.Date(2020, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes("a.txt", recent, recent); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes("b.log", old, old); err != nil {
		t.Fatal(err)
	}

	r := run(t, nil, ".", "-maxdepth", "1", "(", "-name", "?.*", "-o", "-type", "l", ")", "!", "-name", ".hidden", "-ls")
	if r.e != nil || r.err != "" {
		t.Fatalf("result = %+v", r)
	}
	got := strings.Split(strings.TrimSuffix(r.out, "\n"), "\n")
	if len(got) != 3 {
		t.Fatalf("got %q", r.out)
	}
	// inode (9 wide), 1K blocks (6), mode, links (3), owner and group (8,
	// left-aligned), size (8), date, name.
	owner := `\S+ +\S+ +`
	for i, pattern := range []string{
		`^ *\d+ +\d+ -rw-r----- +1 ` + owner + ` {7}5 Jun 14 12:00 \./a\.txt$`,
		`^ *\d+ +0 -rw-r--r-- +1 ` + owner + ` {7}0 Feb  3  2020 \./b\.log$`,
		`^ *\d+ +\d+ lrwx\S{6} +1 ` + owner + ` {7}5 \S+ +\d+ +\S+ \./link\\ to\\ a -> a\.txt$`,
	} {
		if !regexp.MustCompile(pattern).MatchString(got[i]) {
			t.Errorf("line %d = %q, want a match for %s", i, got[i], pattern)
		}
	}
	// The columns line up.
	if strings.Index(got[0], "-rw") != strings.Index(got[1], "-rw") {
		t.Errorf("columns differ:\n%s\n%s", got[0], got[1])
	}

	out := filepath.Join(t.TempDir(), "listing")
	check(t, "", "a.txt", "-fls", out)
	listed, err := os.ReadFile(out)
	if err != nil || !strings.HasSuffix(string(listed), " Jun 14 12:00 a.txt\n") {
		t.Fatalf("-fls wrote %q (%v)", listed, err)
	}
}

func TestExecProgram(t *testing.T) {
	root := tree(t)
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	// With no Exec, the command is a program on PATH; its status decides.
	r := run(t, nil, ".", "-name", "*.txt", "-exec", "sh", "-c", `test "$1" = ./a.txt`, "sh", "{}", ";", "-print")
	if r.e != nil || r.out != lines("./a.txt") {
		t.Fatalf("result = %+v", r)
	}
	// -execdir runs it in the file's directory, with ./name.
	r = run(t, nil, ".", "-name", "c.txt", "-execdir", "sh", "-c", `pwd -P; echo "$1"`, "sh", "{}", ";")
	if want := filepath.Join(root, "dir") + "\n./c.txt\n"; r.e != nil || r.out != want {
		t.Fatalf("result = %+v, want %q", r, want)
	}
	r = run(t, nil, ".", "-name", "?.txt", "-execdir", "sh", "-c", `echo "$@"`, "sh", "{}", "+")
	if want := "./a.txt\n./c.txt\n"; r.e != nil || r.out != want {
		t.Fatalf("result = %+v, want %q", r, want)
	}
	// A tool that is not found in-process falls through to PATH.
	r = run(t, func(f *find.Find) { f.Exec = (&fakeTools{}).exec }, "a.txt", "-exec", "sh", "-c", "exit 3", ";", "-o", "-print")
	if r.e != nil || r.out != "a.txt\n" {
		t.Fatalf("result = %+v", r)
	}
	r = run(t, nil, "a.txt", "-exec", "sh", "-c", "exit 3", "sh", "{}", "+")
	if !errors.Is(r.e, core.ErrFailed) || r.err != "" {
		t.Fatalf("result = %+v", r)
	}
}

func TestIgnoreReaddirRace(t *testing.T) {
	tree(t)
	// A file removed between the directory listing and its examination is
	// diagnosed, unless -ignore_readdir_race says not to.
	remove := func(f *find.Find) {
		f.Exec = func(string, []string, core.Streams) (int, bool) {
			_ = os.Remove("b.log")
			return 0, true
		}
	}
	r := run(t, remove, ".", "-maxdepth", "1", "-name", "a.txt", "-exec", "rm", ";", "-o", "-name", "b.log", "-size", "0")
	if !errors.Is(r.e, core.ErrFailed) || r.err != "findx: './b.log': No such file or directory\n" {
		t.Fatalf("result = %+v", r)
	}
	write(t, "b.log", 0)
	r = run(t, remove, ".", "-ignore_readdir_race", "-maxdepth", "1", "-name", "a.txt", "-exec", "rm", ";", "-o", "-name", "b.log", "-size", "0")
	if r.e != nil || r.err != "" {
		t.Fatalf("result = %+v", r)
	}
}
