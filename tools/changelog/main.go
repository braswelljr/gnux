// Command changelog prints the CHANGELOG.md section of one release, for the
// release workflow to publish as that release's notes.
//
// Usage:
//
//	go run ./tools/changelog vX.Y.Z [CHANGELOG.md]
//
// The version may be given with or without its leading v. The section is the
// one headed "## [X.Y.Z]" (a date may follow), up to the next "## " heading,
// without the heading itself. It exits 1, saying so, when there is no such
// section or the section is empty, so a release cannot be cut without one.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintln(os.Stderr, "usage: changelog VERSION [CHANGELOG.md]")
		os.Exit(2)
	}
	file := "CHANGELOG.md"
	if len(os.Args) == 3 {
		file = os.Args[2]
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "changelog: %v\n", err)
		os.Exit(1)
	}
	notes, ok := Section(string(raw), os.Args[1])
	if !ok {
		fmt.Fprintf(os.Stderr, "changelog: %s has no entries for %s; add a \"## [%s]\" section before tagging\n",
			file, os.Args[1], strings.TrimPrefix(os.Args[1], "v"))
		os.Exit(1)
	}
	fmt.Print(notes)
}

// Section returns the body of the changelog's section for version ("v1.2.0"
// or "1.2.0"): the lines after its "## [1.2.0]" heading, up to the next
// level-two heading or the link definitions at the end, trimmed. ok is false
// when there is no such section or it says nothing.
func Section(changelog, version string) (notes string, ok bool) {
	heading := "## [" + strings.TrimPrefix(version, "v") + "]"
	var body []string
	in := false
	for _, line := range strings.Split(changelog, "\n") {
		if strings.HasPrefix(line, "## ") {
			if in {
				break
			}
			in = line == heading || strings.HasPrefix(line, heading+" ")
			continue
		}
		if !in {
			continue
		}
		// "[1.2.0]: https://..." link definitions close the last section.
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]: ") {
			break
		}
		body = append(body, line)
	}
	notes = strings.TrimSpace(strings.Join(body, "\n"))
	if notes == "" {
		return "", false
	}
	return notes + "\n", true
}
