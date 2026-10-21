// Command docsnap keeps the website's documentation of a release: the docs as
// that version shipped, in web/content/docs/vX.Y, beside the latest ones.
//
// Usage:
//
//	go run ./tools/docsnap -tag vX.Y.Z [-out web/content/docs] [-from DIR]
//	go run ./tools/docsnap -tidy [-out web/content/docs]
//
// Without -from it checks the tag out in a temporary git worktree and runs
// that version's own generator (go run ./tools/gendocs DIR mdx), so the
// snapshot says what the release says, whatever has changed since. With
// -from it takes pages already generated into DIR. Either way it writes them
// into OUT/vX.Y (the tag's major and minor version: a patch release replaces
// its minor version's snapshot), rewriting every link into /docs to point
// into the snapshot, and marks the folder as a sidebar root of its own, so
// the website's version picker can switch to it. `make docs` keeps the
// snapshots; the Release workflow runs docsnap for every stable release.
//
// A tool with no option pages is written as a page of its own, not as a
// folder holding only its index, so that the sidebar shows it as a link
// rather than a collapsible entry with nothing under it (Flatten): releases
// whose generator made such folders are laid out as the latest docs are.
// -tidy does that to the snapshots already in OUT, without generating any.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	tag := flag.String("tag", "", "the release tag, as v1.2.3")
	out := flag.String("out", filepath.Join("web", "content", "docs"), "the website's docs content directory")
	from := flag.String("from", "", "pages already generated for the tag (default: generate them from the tag)")
	tidy := flag.Bool("tidy", false, "flatten the tool folders of the snapshots already in -out, and nothing else")
	flag.Parse()
	if *tidy && *tag == "" && flag.NArg() == 0 {
		if err := tidyAll(*out); err != nil {
			fail(err)
		}
		return
	}
	if *tag == "" || flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "usage: docsnap -tag vX.Y.Z [-out DIR] [-from DIR] | docsnap -tidy [-out DIR]")
		os.Exit(2)
	}
	version, err := Version(*tag)
	if err != nil {
		fail(err)
	}
	src := *from
	if src == "" {
		dir, cleanup, err := generateAt(*tag)
		if err != nil {
			fail(err)
		}
		defer cleanup()
		src = dir
	}
	dst := filepath.Join(*out, version)
	if err := Snapshot(src, dst, version); err != nil {
		fail(err)
	}
	fmt.Printf("docsnap: wrote the %s docs to %s\n", version, dst)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "docsnap: %v\n", err)
	os.Exit(1)
}

var tagPattern = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.[0-9]+$`)

// Version is the snapshot folder of a stable release tag: v1.2.3 is v1.2. A
// prerelease (v1.2.3-rc.1) has none.
func Version(tag string) (string, error) {
	m := tagPattern.FindStringSubmatch(tag)
	if m == nil {
		return "", fmt.Errorf("%q is not a stable release tag (vX.Y.Z)", tag)
	}
	return "v" + m[1] + "." + m[2], nil
}

// generateAt generates the docs of tag with its own generator, in a
// temporary worktree, and returns the directory they are in.
func generateAt(tag string) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "docsnap-")
	if err != nil {
		return "", nil, err
	}
	tree, pages := filepath.Join(tmp, "tree"), filepath.Join(tmp, "pages")
	cleanup := func() {
		_ = exec.Command("git", "worktree", "remove", "--force", tree).Run()
		_ = os.RemoveAll(tmp)
	}
	run := func(dir string, name string, args ...string) error {
		c := exec.Command(name, args...)
		c.Dir = dir
		c.Stdout, c.Stderr = os.Stderr, os.Stderr
		return c.Run()
	}
	if err := run("", "git", "worktree", "add", "--detach", tree, tag); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("checking out %s: %w", tag, err)
	}
	if err := run(tree, "go", "run", "./tools/gendocs", pages, "mdx"); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("generating the %s docs: %w", tag, err)
	}
	return pages, cleanup, nil
}

// docsLink is a markdown link into the docs: "](/docs" followed by the end of
// the path or more of it.
var docsLink = regexp.MustCompile(`\]\(/docs([/)#?])`)

// Snapshot copies the pages in src to dst as the docs of version: links into
// /docs lead into /docs/<version>, and the root meta.json makes the folder a
// sidebar root titled with the version. Snapshots inside src are skipped.
func Snapshot(src, dst, version string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := copyPages(src, dst, version); err != nil {
		return err
	}
	return Flatten(dst)
}

// versionDir matches a snapshot's folder name.
var versionDir = regexp.MustCompile(`^v[0-9]+\.[0-9]+$`)

// tidyAll flattens every snapshot in out.
func tidyAll(out string) error {
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() && versionDir.MatchString(e.Name()) {
			if err := Flatten(filepath.Join(out, e.Name())); err != nil {
				return err
			}
			fmt.Printf("docsnap: tidied %s\n", filepath.Join(out, e.Name()))
		}
	}
	return nil
}

// Flatten turns each tools/<tool>/ folder of a docs tree that holds nothing
// but its index page (and a meta.json listing no pages) into tools/<tool>.mdx.
// The page keeps its URL, /docs/.../tools/<tool>, and becomes a link in the
// sidebar instead of a folder with nothing to open.
func Flatten(root string) error {
	tools := filepath.Join(root, "tools")
	entries, err := os.ReadDir(tools)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(tools, e.Name())
		inside, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		onlyIndex := true
		for _, f := range inside {
			switch {
			case f.Name() == "index.mdx":
			case f.Name() == "meta.json":
				if !emptyMeta(filepath.Join(dir, f.Name())) {
					onlyIndex = false
				}
			default:
				onlyIndex = false
			}
		}
		if !onlyIndex {
			continue
		}
		if err := os.Rename(filepath.Join(dir, "index.mdx"), filepath.Join(tools, e.Name()+".mdx")); err != nil {
			return err
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	return nil
}

// emptyMeta reports whether a meta.json lists no pages.
func emptyMeta(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var meta struct {
		Pages []string `json:"pages"`
	}
	return json.Unmarshal(data, &meta) == nil && len(meta.Pages) == 0
}

// copyPages copies src to dst, rewriting links and the root meta.json.
func copyPages(src, dst, version string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if rel != "." && versionDir.MatchString(d.Name()) && filepath.Dir(rel) == "." {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		switch {
		case rel == "meta.json":
			if data, err = rootMeta(data, version); err != nil {
				return err
			}
		case strings.HasSuffix(rel, ".mdx"):
			data = docsLink.ReplaceAll(data, []byte("](/docs/"+version+"$1"))
		}
		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
}

// rootMeta makes a docs root's meta.json the root of a version.
func rootMeta(data []byte, version string) ([]byte, error) {
	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("meta.json: %w", err)
	}
	meta["title"] = version
	meta["root"] = true
	out, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
