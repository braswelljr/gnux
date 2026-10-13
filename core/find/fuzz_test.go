package find_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/find"
)

// FuzzExpression checks that no expression makes the parser or the walk
// panic. The expression is the fuzzed text split at spaces and is evaluated
// over a small tree. Words that would reach outside that tree are left out:
// the actions that write, delete or run something, and a second starting
// point.
func FuzzExpression(f *testing.F) {
	for _, e := range []string{
		"", "-name *.txt", "-type f -o -type d", "! -name a -print", "( -name a -o -name b ) -prune -o -print",
		"-size +1c -a -size -2k", "-mtime -1 -mmin +0", "-newer a.txt", "-perm -u+r,g-w -perm /022 -perm 644",
		"-maxdepth 1 -mindepth 1 -depth", "-printf %p\\t%s\\n%y%m%M%u%g%Tk%T@%%", "-printf %-10.3f|%5s|%Ak", "-print0 -quit",
		"-regex .*a.* -iregex .*B", "-regextype posix-extended -regex (a|b)+", "-path */d/* -ipath *D*", "-empty , -true",
		"-links 1 -inum +1 -uid 0 -gid -1", "-user root -group wheel -nouser -nogroup", "-xtype l -lname *",
		"-newermt 2020-01-02 -newerat @0", "-name", "(", ")", "-o", "! !", "-bogus", "-size x", "-type z", "-ls",
		"-daystart -atime 0 -ctime +0", "-samefile a.txt", "-readable -writable -executable", "-O2 -false", "-printf %",
		"-L -type l", "-H", "-P -follow", "-name [ -name [a- -name \\", "- /", "-L /", "-- /", "-O2 - -print",
	} {
		f.Add(e)
	}

	root := f.TempDir()
	for _, name := range []string{"a.txt", "d/b.log", "d/e/c", "empty/"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if strings.HasSuffix(name, "/") {
			_ = os.MkdirAll(path, 0o755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		_ = os.WriteFile(path, []byte(name), 0o644)
	}
	_ = os.Symlink("a.txt", filepath.Join(root, "link"))
	now := time.Now()

	f.Fuzz(func(t *testing.T, expression string) {
		if len(expression) > 200 {
			return
		}
		words := strings.Fields(expression)
		for _, w := range words {
			for _, unsafe := range []string{"exec", "ok", "delete", "fprint", "fls"} {
				if strings.Contains(w, unsafe) {
					return
				}
			}
		}
		if namesAStartingPoint(words) {
			return
		}
		e := find.New(find.Options{}, core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard})
		e.Now = func() time.Time { return now }
		_ = e.Run(append([]string{root}, words...))
	})
}

// namesAStartingPoint reports whether words, given to find after a starting
// point, would add another one, which could be anywhere on the disk ("/").
// The words before the expression are starting points, apart from the leading
// options; a lone "-" is a file name, and "--" ends the options.
func namesAStartingPoint(words []string) bool {
	for _, w := range words {
		switch {
		case w == "-H", w == "-L", w == "-P", strings.HasPrefix(w, "-O"):
			continue // a leading option: starting points may still follow
		case w == "(", w == "!":
			return false
		case strings.HasPrefix(w, "-") && w != "-" && w != "--":
			return false // the expression starts here
		default:
			return true
		}
	}
	return false
}
