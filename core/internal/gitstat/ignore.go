package gitstat

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/fnmatch"
)

// rule is one line of an ignore file.
type rule struct {
	base     string   // the directory the file is in, relative to the top ("" for the top)
	segments []string // the pattern split at "/"
	negate   bool     // "!pattern": re-include
	dirOnly  bool     // "pattern/": directories only
	anchored bool     // the pattern has a "/" before its end: matched from base
}

// ignorer evaluates the ignore rules of one working tree, as git does: the
// global excludes file, .git/info/exclude, then each .gitignore from the
// top down to the file's own directory, the last matching line winning,
// and nothing re-included inside an excluded directory.
type ignorer struct {
	repo   *Repo
	global []rule
	loaded bool
	sets   map[string][]rule // .gitignore rules by directory
	memo   map[string]bool
}

func (g *ignorer) loadGlobal() {
	if g.loaded {
		return
	}
	g.loaded = true
	r := g.repo
	file := r.excludesFile
	if file == "" {
		if xdg := r.getenv("XDG_CONFIG_HOME"); xdg != "" {
			file = filepath.Join(xdg, "git", "ignore")
		} else if home := r.home(); home != "" {
			file = filepath.Join(home, ".config", "git", "ignore")
		}
	}
	if file != "" {
		g.global = readRules(file, "")
	}
	g.global = append(g.global, readRules(filepath.Join(r.commonDir, "info", "exclude"), "")...)
}

// rulesIn returns the rules of dir's .gitignore.
func (g *ignorer) rulesIn(dir string) []rule {
	if rules, ok := g.sets[dir]; ok {
		return rules
	}
	rules := readRules(filepath.Join(g.repo.Root, filepath.FromSlash(dir), ".gitignore"), dir)
	g.sets[dir] = rules
	return rules
}

// ignored reports whether rel (slash-separated, relative to the top) is
// excluded.
func (g *ignorer) ignored(rel string, isDir bool) bool {
	key := rel
	if isDir {
		key += "/"
	}
	if v, ok := g.memo[key]; ok {
		return v
	}
	g.loadGlobal()
	parent := path.Dir(rel)
	if parent == "." {
		parent = ""
	}
	result := parent != "" && g.ignored(parent, true)
	if !result {
		apply := func(rules []rule) {
			for i := range rules {
				if rules[i].match(rel, isDir) {
					result = !rules[i].negate
				}
			}
		}
		apply(g.global)
		dir := ""
		apply(g.rulesIn(dir))
		for part := range strings.SplitSeq(parent, "/") {
			if part == "" {
				break
			}
			dir = path.Join(dir, part)
			apply(g.rulesIn(dir))
		}
	}
	g.memo[key] = result
	return result
}

// readRules parses an ignore file; a missing one has no rules.
func readRules(file, base string) []rule {
	data, err := fsys.ReadFile(file)
	if err != nil {
		return nil
	}
	var rules []rule
	for line := range strings.SplitSeq(string(data), "\n") {
		if r, ok := parseRule(strings.TrimSuffix(line, "\r"), base); ok {
			rules = append(rules, r)
		}
	}
	return rules
}

// parseRule reads one line in gitignore(5)'s syntax.
func parseRule(line, base string) (rule, bool) {
	// Trailing spaces go unless escaped with a backslash.
	for strings.HasSuffix(line, " ") && !strings.HasSuffix(line, "\\ ") {
		line = line[:len(line)-1]
	}
	if line == "" || line[0] == '#' {
		return rule{}, false
	}
	r := rule{base: base}
	switch {
	case line[0] == '!':
		r.negate, line = true, line[1:]
	case strings.HasPrefix(line, "\\!"), strings.HasPrefix(line, "\\#"):
		line = line[1:]
	}
	if strings.HasSuffix(line, "/") {
		r.dirOnly, line = true, strings.TrimRight(line, "/")
	}
	if line == "" {
		return rule{}, false
	}
	r.anchored = strings.Contains(line, "/")
	r.segments = strings.Split(strings.TrimPrefix(line, "/"), "/")
	return r, true
}

// match reports whether the rule matches rel.
func (r *rule) match(rel string, isDir bool) bool {
	if r.dirOnly && !isDir {
		return false
	}
	sub := rel
	if r.base != "" {
		var ok bool
		if sub, ok = strings.CutPrefix(rel, r.base+"/"); !ok {
			return false
		}
	}
	if !r.anchored {
		return fnmatch.Match(r.segments[0], path.Base(sub), false)
	}
	return matchSegments(r.segments, strings.Split(sub, "/"))
}

// matchSegments matches a slash-split pattern against a slash-split path,
// "**" standing for any number of whole segments (at the end: one or more).
func matchSegments(pattern, name []string) bool {
	if len(pattern) == 0 {
		return len(name) == 0
	}
	if pattern[0] == "**" {
		if len(pattern) == 1 {
			return len(name) > 0
		}
		for i := 0; i <= len(name); i++ {
			if matchSegments(pattern[1:], name[i:]) {
				return true
			}
		}
		return false
	}
	return len(name) > 0 && fnmatch.Match(pattern[0], name[0], false) && matchSegments(pattern[1:], name[1:])
}
