//go:build unix

package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

// openFiles counts this process's open file descriptors.
func openFiles(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Skipf("cannot list open files: %v", err)
	}
	return len(entries)
}

// TestNoFileIsLeftOpen runs the tools that open many files, on their success
// and failure paths, and checks that the process holds no more descriptors
// afterwards. A tool is one call in a long-lived host (a shell script, the
// playground, an embedding program): a descriptor leaked per file would run
// that host out of them.
func TestNoFileIsLeftOpen(t *testing.T) {
	dir := t.TempDir()
	var files []string
	for i := range 40 {
		sub := filepath.Join(dir, fmt.Sprintf("d%d", i%4))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(sub, fmt.Sprintf("f%02d.txt", i))
		if err := os.WriteFile(name, []byte("alpha\nbeta fox\ngamma\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		files = append(files, name)
	}
	missing := filepath.Join(dir, "missing")
	all := strings.Join(files, " ")
	out := filepath.Join(dir, "out")

	scripts := []string{
		"cat " + all + " " + missing,
		"cat -n " + all,
		"head -n 1 " + all + " " + missing,
		"tail -n 1 " + all,
		"wc " + all + " " + missing,
		"sort " + all,
		"sort -m " + all,
		"paste " + all,
		"cksum " + all,
		"sha256sum " + all + " " + missing,
		"od -c " + all,
		"nl " + files[0] + " " + missing,
		"cut -c1-3 " + all,
		"grep -r fox " + dir,
		"grep -l fox " + all + " " + missing,
		"grep -rq fox " + dir,
		"grep -c zebra " + all,
		"sed -n 2p " + all + " " + missing,
		"sed -s -n $p " + all,
		"sed -i.bak s/alpha/ALPHA/ " + all,
		"sed 2q " + all,
		"find " + dir + " -name '*.bak' -delete",
		"find " + dir + " -type f -exec cat {} +",
		"find " + dir + " -type f -exec grep -q fox {} ;",
		"find " + dir + " -type f -size -1k -newer " + files[0] + " -printf %p\\n",
		"find " + dir + " -name '*.txt' -fprint " + out,
		"find " + dir + " -name '*.txt' -print0 | xargs -0 -n 7 wc -l",
		"find " + dir + " -name '*.txt' | xargs -P 4 -n 5 cat",
		"ls -lR " + dir,
		"du -a " + dir,
		"stat " + all,
		"cp -r " + filepath.Join(dir, "d0") + " " + filepath.Join(dir, "copy"),
		"rm -r " + filepath.Join(dir, "copy"),
		"tee " + out + " " + out + "2",
		"split -l 1 " + files[0] + " " + filepath.Join(dir, "part."),
		"cat " + files[0] + " > " + out,
		"cat < " + files[0] + " | head -n 1 >> " + out,
		"cat < " + missing,
	}

	run := func() {
		for _, script := range scripts {
			s := core.Streams{In: strings.NewReader("input\n"), Out: io.Discard, Err: io.Discard}
			Main([]string{"gnux", "shell", "-c", script}, s)
		}
	}
	run() // open whatever stays open for the life of the process
	before := openFiles(t)
	for range 3 {
		run()
	}
	if after := openFiles(t); after > before {
		t.Fatalf("%d open files before, %d after: something is not closed", before, after)
	}
}
