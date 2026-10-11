// Command gendocs generates the gnux reference - the entry page, the guide and
// one page per tool (and per flag, where a tool documents them) - in every
// cobra format.
//
// Usage:
//
//	go run ./tools/gendocs [dir] [format]
//
//	dir      output base directory (default "documentation")
//	format   one of: all (default), markdown, man, yaml, rest, or mdx
//
// "all" writes the four reference formats under dir/<format>. "mdx" is
// separate: it writes the website's content straight into dir, which should be
// the Fumadocs content root (web/content/docs), and includes the hand-written
// guides in documentation/guides (the requirements, workflows and rules in
// docs/ stay in the repository). Run it from the repository root.
//
// It is also wired as a `go generate` step (see main.go) and `make docs`.
package main

import (
	"log"
	"os"

	"github.com/braswelljr/gnux/cmd/docs"
)

// The hand-written documents published on the website with the generated
// pages: each folder of handwrittenRoot becomes a section of the same name.
// They sit beside the generated reference in documentation/, which the
// generator rebuilds format by format (documentation/<format>), so a folder
// that is not a format name is never touched by it.
const handwrittenRoot = "documentation"

var handwrittenFolders = []string{"guides"}

func main() {
	dir := "documentation"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	format := "all"
	if len(os.Args) > 2 {
		format = os.Args[2]
	}

	if err := generate(dir, format); err != nil {
		log.Fatalf("gendocs: %v", err)
	}

	log.Printf("gendocs: wrote %s reference to %s/", format, dir)
}

func generate(dir, format string) error {
	switch format {
	case "all":
		return docs.GenerateAll(dir)
	case "mdx":
		// The hand-written documents live beside the reference and are read
		// relative to the repository root, where gendocs is run from.
		pages, err := docs.HandwrittenPages(handwrittenRoot, handwrittenRoot, handwrittenFolders...)
		if err != nil {
			return err
		}
		return docs.GenerateMDX(dir, pages...)
	case "md":
		format = "markdown"
	case "yml":
		format = "yaml"
	case "rst":
		format = "rest"
	}
	return docs.Generate(dir, format)
}
