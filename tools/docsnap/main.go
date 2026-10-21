// Command docsnap makes the website's documentation of a release: the docs as
// that version shipped, laid out to sit beside the latest ones under
// /docs/vX.Y (the tag's major and minor version: a patch release replaces its
// minor version's docs).
//
// Usage:
//
//	go run ./tools/docsnap -tag vX.Y.Z -here -archive FILE    the release's docs, packed
//	go run ./tools/docsnap -tag vX.Y.Z [-from DIR] [-out DIR] the release's docs, unpacked
//	go run ./tools/docsnap -pack DIR -archive FILE            pack a vX.Y folder already made
//	go run ./tools/docsnap -tidy [-out DIR]                   flatten the vX.Y folders in OUT
//
// A release's docs travel with the release: GoReleaser runs the first form at
// the tag (-here: this checkout's own generator) and attaches the archive,
// gnux-docs-vX.Y.Z.tar.gz, to the GitHub release, listed in the signed
// checksums.txt. The website's build (web/scripts/gen.mjs) downloads the
// archive of the latest patch of every minor release and unpacks it into
// web/content/docs, where nothing is committed. Without -here the docs are
// generated from the tag in a temporary git worktree, or taken from -from.
//
// Every link into /docs is rewritten to point into /docs/vX.Y, the root
// meta.json makes the folder a sidebar root of its own for the version
// picker, and a tool with no option pages is written as a page of its own, not
// a folder holding only its index, so that the sidebar shows it as a link
// rather than a collapsible entry with nothing under it (Flatten).
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
	here := flag.Bool("here", false, "generate the pages from this checkout, which is the tag")
	archive := flag.String("archive", "", "pack the docs into this .tar.gz (and FILE.sha256) instead of writing them to -out")
	pack := flag.String("pack", "", "a vX.Y folder already made, to pack with -archive")
	tidy := flag.Bool("tidy", false, "flatten the tool folders of the vX.Y folders in -out, and nothing else")
	flag.Parse()
	if flag.NArg() > 0 {
		usage()
	}
	switch {
	case *tidy && *tag == "" && *pack == "":
		if err := tidyAll(*out); err != nil {
			fail(err)
		}
	case *pack != "" && *archive != "" && *tag == "":
		if err := packDir(*pack, *archive); err != nil {
			fail(err)
		}
	case *tag != "":
		if err := release(*tag, *from, *here, *out, *archive); err != nil {
			fail(err)
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: docsnap -tag vX.Y.Z [-here | -from DIR] [-archive FILE | -out DIR]\n"+
		"       docsnap -pack DIR -archive FILE\n       docsnap -tidy [-out DIR]")
	os.Exit(2)
}

// release makes the docs of tag, into out/vX.Y or packed into archive.
func release(tag, from string, here bool, out, archive string) error {
	version, err := Version(tag)
	if err != nil && archive != "" {
		// A prerelease's archive is made too, so that the release looks the
		// same; the website takes stable releases only.
		version, err = prereleaseVersion(tag)
	}
	if err != nil {
		return err
	}
	src := from
	if src == "" {
		dir, cleanup, err := generate(tag, here)
		if err != nil {
			return err
		}
		defer cleanup()
		src = dir
	}
	if archive == "" {
		dst := filepath.Join(out, version)
		if err := Snapshot(src, dst, version); err != nil {
			return err
		}
		fmt.Printf("docsnap: wrote the %s docs to %s\n", version, dst)
		return nil
	}
	tmp, err := os.MkdirTemp("", "docsnap-pack-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	dst := filepath.Join(tmp, version)
	if err := Snapshot(src, dst, version); err != nil {
		return err
	}
	return packDir(dst, archive)
}

// packDir packs a vX.Y folder into archive, with the folder at its top, and
// writes archive.sha256 beside it.
func packDir(dir, archive string) error {
	if !versionDir.MatchString(filepath.Base(filepath.Clean(dir))) {
		return fmt.Errorf("%s is not a vX.Y folder", dir)
	}
	if err := Flatten(dir); err != nil {
		return err
	}
	if err := Pack(dir, archive); err != nil {
		return err
	}
	fmt.Printf("docsnap: packed %s into %s\n", dir, archive)
	return nil
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

var prereleasePattern = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.[0-9]+-[0-9A-Za-z.-]+$`)

// prereleaseVersion is the vX.Y folder of a prerelease tag (v1.3.0-rc.1).
func prereleaseVersion(tag string) (string, error) {
	m := prereleasePattern.FindStringSubmatch(tag)
	if m == nil {
		return "", fmt.Errorf("%q is not a release tag (vX.Y.Z or vX.Y.Z-PRE)", tag)
	}
	return "v" + m[1] + "." + m[2], nil
}

// generate generates the docs of tag with its own generator: this checkout's
// when here, or the tag's in a temporary worktree. It returns the directory
// they are in.
func generate(tag string, here bool) (string, func(), error) {
	if !here {
		return generateAt(tag)
	}
	tmp, err := os.MkdirTemp("", "docsnap-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	c := exec.Command("go", "run", "./tools/gendocs", tmp, "mdx")
	c.Stdout, c.Stderr = os.Stderr, os.Stderr
	if err := c.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("generating the %s docs: %w", tag, err)
	}
	return tmp, cleanup, nil
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
