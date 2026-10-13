package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

// The benchmarks run the real tools, through the same dispatch a command line
// takes, on generated input: `make bench` runs them, and docs/workflows/
// benchmarking.md says how to read and compare the results.

// benchInput is the text the stream benchmarks read: 8 MiB of lines.
var benchInput = bytes.Repeat(textBlock, 8<<20/len(textBlock))

// streamBenchmarks are command lines that read standard input. The names are
// what -bench selects on: `-bench 'Stream/grep'`.
var streamBenchmarks = []string{
	"cat", "cat -n", "wc", "wc -l", "head -n 100000", "tail -n 100000", "tac", "nl",
	"grep fox", "grep -c zebra", "grep -i QUICK", "grep -E (fox|cat)s?", "grep -v lazy", "grep -on [0-9]+", "grep -F -w dog",
	"sed s/quick/slow/", "sed -n /9999/p", "sed -E s/([a-z]+):.([a-z]+)/\\2:\\1/g", "sed $d", "sed y/abc/xyz/",
	"cut -c1-20", "cut -f2", "tr a-z A-Z", "tr -d aeiou", "sort", "sort -r", "sort -u", "uniq", "uniq -c",
	"fold -w 40", "fmt", "expand", "unexpand -a", "paste - -", "od", "od -c", "base64", "cksum", "md5sum", "sha256sum",
	"shuf", "xargs true", "xargs -n 10 true",
}

// BenchmarkStream measures each tool's throughput on 8 MiB of text held in
// memory, written to nowhere: the tool's own work, without the disk.
func BenchmarkStream(b *testing.B) {
	for _, line := range streamBenchmarks {
		b.Run(line, func(b *testing.B) {
			argv := strings.Fields(line)
			b.SetBytes(int64(len(benchInput)))
			b.ReportAllocs()
			for b.Loop() {
				s := core.Streams{In: bytes.NewReader(benchInput), Out: io.Discard, Err: io.Discard}
				if status, ok := RunTool(argv[0], argv[1:], s); !ok || status > 1 {
					b.Fatalf("status = %d, found = %v", status, ok)
				}
			}
		})
	}
}

// BenchmarkGenerate measures the tools that produce output from nothing.
func BenchmarkGenerate(b *testing.B) {
	for _, line := range []string{"seq 1 1000000", "seq -w 1 1000000", "seq 0 0.5 500000", "true", "echo hello", "printf %s\\n a b c"} {
		b.Run(line, func(b *testing.B) {
			argv := strings.Fields(line)
			b.ReportAllocs()
			for b.Loop() {
				RunTool(argv[0], argv[1:], core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard})
			}
		})
	}
}

// benchTree builds a directory tree of dirs directories of files files each
// and returns its root.
func benchTree(b *testing.B, dirs, files int) string {
	b.Helper()
	root := b.TempDir()
	body := textBlock[:2<<10]
	for d := range dirs {
		dir := filepath.Join(root, fmt.Sprintf("pkg%02d", d%10), fmt.Sprintf("dir%03d", d))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatal(err)
		}
		for f := range files {
			ext := []string{".go", ".txt", ".md", ".log"}[f%4]
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%03d%s", f, ext)), body, 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}
	return root
}

// BenchmarkTree measures the tools that walk a directory tree, on 2,000
// small files in 50 directories. DIR in a command line is the tree's root.
func BenchmarkTree(b *testing.B) {
	root := benchTree(b, 50, 40)
	for _, line := range []string{
		"find DIR", "find DIR -name *.go", "find DIR -type f -size +1k -newer DIR", "find DIR -name *.md -exec true {} +",
		"grep -r fox DIR", "grep -rl zebra DIR", "grep -rc --include=*.go quick DIR", "ls -R DIR", "ls -lR DIR", "du -s DIR",
		"wc -l DIR/pkg00/dir000/file000.go", "sed -n 1p DIR/pkg00/dir000/file000.go",
	} {
		b.Run(line, func(b *testing.B) {
			argv := strings.Fields(strings.ReplaceAll(line, "DIR", root))
			b.ReportAllocs()
			for b.Loop() {
				s := core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard}
				if status, ok := RunTool(argv[0], argv[1:], s); !ok || status > 1 {
					b.Fatalf("status = %d, found = %v", status, ok)
				}
			}
		})
	}
}

// BenchmarkShell measures whole pipelines in the gnux shell: the tools plus
// the pipes between them.
func BenchmarkShell(b *testing.B) {
	for _, script := range []string{
		"true",
		"seq 1 200000 | sort -rn | uniq | wc -l",
		"seq 1 200000 | grep 7 | sed s/7/seven/ | cut -c1-5 | tail -n 1",
		"yes line | head -n 500000 | nl | tail -n 1",
		"seq 1 20000 | xargs -n 100 echo | wc -l",
	} {
		b.Run(script, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				s := core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard}
				if status := Main([]string{"gnux", "shell", "-c", script}, s); status != 0 {
					b.Fatalf("status = %d", status)
				}
			}
		})
	}
}
