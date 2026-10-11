package docs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Page is a hand-written markdown document to publish on the website alongside
// the generated reference.
type Page struct {
	Dir         string // folder under the content root, "" for the top level
	Name        string // file base name and URL slug, e.g. "functional-requirements"
	Title       string
	Description string
	Markdown    string // the document; a leading "# Title" line is dropped
}

// repoURL is where a link to a file outside the published documents points.
const repoURL = "https://github.com/braswelljr/gnux/blob/main/"

// HandwrittenPages reads the hand-written documents under dir for publication
// on the website. repoPath is where dir sits in the repository
// ("documentation"), which relative links are resolved against. Each of
// folders is a subdirectory of markdown files (gendocs publishes
// documentation/guides) and becomes a section of the same name; a README.md is
// the section's index page. A folder that does not exist is skipped.
//
// A page's title is its first level-one heading and its description its first
// paragraph. Relative links between the documents are rewritten to site URLs,
// and links to anything else in the repository to its page on GitHub.
func HandwrittenPages(dir, repoPath string, folders ...string) ([]Page, error) {
	published := map[string]bool{}
	for _, f := range folders {
		published[f] = true
	}

	var pages []Page
	for _, folder := range folders {
		entries, err := os.ReadDir(filepath.Join(dir, folder))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", folder, err)
		}

		var names []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				names = append(names, e.Name())
			}
		}
		// The index leads its section; the rest follow alphabetically.
		sort.Slice(names, func(i, j int) bool {
			if (names[i] == "README.md") != (names[j] == "README.md") {
				return names[i] == "README.md"
			}
			return names[i] < names[j]
		})

		for _, name := range names {
			raw, err := os.ReadFile(filepath.Join(dir, folder, name))
			if err != nil {
				return nil, fmt.Errorf("read %s/%s: %w", folder, name, err)
			}
			md := rewriteLinks(string(raw), repoPath, folder, published)
			title, description := titleAndDescription(md)
			if title == "" {
				title = folderTitle(strings.TrimSuffix(name, ".md"))
			}
			pages = append(pages, Page{
				Dir:         folder,
				Name:        pageName(name),
				Title:       title,
				Description: description,
				// The site shows the description under the title, so the
				// paragraph it came from is not repeated below it.
				Markdown: withoutLead(md),
			})
		}
	}
	return pages, nil
}

// pageName maps a markdown file name to its page name; a README is the index.
func pageName(file string) string {
	if file == "README.md" {
		return "index"
	}
	return strings.TrimSuffix(file, ".md")
}

// folderTitle turns a folder or file name into a heading: "srs-workflows"
// becomes "Srs workflows".
func folderTitle(name string) string {
	name = strings.ReplaceAll(name, "-", " ")
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// withoutLead removes the summary paragraph that follows a document's title:
// the paragraph titleAndDescription takes the description from. A document
// whose title is followed by anything but prose is returned unchanged.
func withoutLead(md string) string {
	lines := strings.Split(md, "\n")
	i := 0
	for i < len(lines) && !strings.HasPrefix(lines[i], "# ") {
		i++
	}
	if i == len(lines) {
		return md
	}
	start := i + 1
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := start
	for end < len(lines) {
		line := strings.TrimSpace(lines[end])
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "|") ||
			strings.HasPrefix(line, "```") || strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "---") {
			break
		}
		end++
	}
	if end == start {
		return md
	}
	return strings.Join(append(lines[:i+1:i+1], lines[end:]...), "\n")
}

// stripTitle removes a leading level-one heading: the website renders the
// title from the frontmatter.
func stripTitle(md string) string {
	md = strings.TrimLeft(md, "\n")
	if strings.HasPrefix(md, "# ") {
		if _, rest, ok := strings.Cut(md, "\n"); ok {
			return strings.TrimLeft(rest, "\n")
		}
		return ""
	}
	return md
}

var (
	inlineLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	markupRe     = regexp.MustCompile("[*_`]")
)

// titleAndDescription extracts a document's title (its first level-one
// heading) and description (its first paragraph, as plain text).
func titleAndDescription(md string) (title, description string) {
	lines := strings.Split(md, "\n")
	i := 0
	for ; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(lines[i], "# "))
			i++
			break
		}
	}

	var para []string
	for ; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			if len(para) > 0 {
				break
			}
			continue
		}
		// A paragraph is prose: stop at the first structural element.
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "|") || strings.HasPrefix(line, "```") ||
			strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "---") {
			break
		}
		para = append(para, line)
	}

	description = strings.Join(para, " ")
	description = inlineLinkRe.ReplaceAllString(description, "$1")
	description = markupRe.ReplaceAllString(description, "")
	return title, description
}

// mdLinkRe matches the target of an inline markdown link. A target may hold
// one level of balanced parentheses, as the website's route groups do
// (web/src/app/(home)/page.tsx).
var mdLinkRe = regexp.MustCompile(`\]\(((?:[^()\s]|\([^()\s]*\))+)\)`)

// referenceDir is where the generated markdown reference lives, relative to
// the repository root. The website carries the same pages, so a link into it
// becomes a site link.
const referenceDir = "documentation/markdown/"

// referenceURL maps a repository path inside the generated markdown reference
// to the website page with the same content: index.md is /docs, guide.md
// /docs/guide, shell/index.md /docs/shell, <tool>/index.md /docs/tools/<tool>,
// and a per-flag page <tool>/flags/<flag>.md /docs/tools/<tool>/<flag>.
func referenceURL(repoPath string) (string, bool) {
	rel, ok := strings.CutPrefix(repoPath, referenceDir)
	if !ok {
		return "", false
	}
	switch rel {
	case "index.md":
		return "/docs", true
	case "guide.md":
		return "/docs/guide", true
	case "shell/index.md":
		return "/docs/shell", true
	}
	parts := strings.Split(rel, "/")
	switch {
	case len(parts) == 2 && parts[1] == "index.md":
		return "/docs/tools/" + parts[0], true
	case len(parts) == 3 && parts[1] == "flags" && strings.HasSuffix(parts[2], ".md"):
		return "/docs/tools/" + parts[0] + "/" + strings.TrimSuffix(parts[2], ".md"), true
	}
	return "", false
}

// rewriteLinks points the relative links of a document in <repoPath>/<folder> at
// where their targets live once published: another published document becomes
// a site URL, anything else in the repository its GitHub page. Absolute URLs
// and in-page anchors are left alone.
func rewriteLinks(md, repoPath, folder string, published map[string]bool) string {
	return mdLinkRe.ReplaceAllStringFunc(md, func(m string) string {
		target := mdLinkRe.FindStringSubmatch(m)[1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") ||
			strings.HasPrefix(target, "/") || strings.HasPrefix(target, "mailto:") {
			return m
		}

		target, anchor, hasAnchor := strings.Cut(target, "#")
		if hasAnchor {
			anchor = "#" + anchor
		}

		// Resolve against the document's own location, relative to the
		// repository root.
		resolved := path.Join(repoPath, folder, target)
		rel := strings.TrimPrefix(resolved, repoPath+"/")

		if url, ok := referenceURL(resolved); ok {
			return "](" + url + anchor + ")"
		}

		if section, file, ok := strings.Cut(rel, "/"); ok && resolved != rel &&
			published[section] && strings.HasSuffix(file, ".md") && !strings.Contains(file, "/") {
			url := "/docs/" + section
			if name := pageName(file); name != "index" {
				url += "/" + name
			}
			return "](" + url + anchor + ")"
		}
		return "](" + repoURL + resolved + anchor + ")"
	})
}
