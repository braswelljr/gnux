package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/cmd"
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/tool"
)

func TestWrapCode(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"flag pair", "the -d/--dir flag", "the `-d`/`--dir` flag"},
		{"command words", "rmx wraps rm via gnux", "`rmx` wraps `rm` via `gnux`"},
		{"adjacent references merge", "run gnux rmx -rf build", "run `gnux rmx -rf` build"},
		{"gnux shell is one command", "start gnux shell with -e", "start `gnux shell` with `-e`"},
		{"prose words untouched", "is this true or false at the head", "is this true or false at the head"},
		{"links untouched", "see [rmx](rmx/index.md) for -r", "see [rmx](rmx/index.md) for `-r`"},
		{"command phrase", "run go install then make build", "run `go install` then `make build`"},
		{"hyphenated words untouched", "a write-protected read-only file", "a write-protected read-only file"},
		{"heading untouched", "## rm and -d", "## rm and -d"},
		{"inline code untouched", "already `-d` here", "already `-d` here"},
		{"long flag with dashes", "the --no-preserve-root failsafe", "the `--no-preserve-root` failsafe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(wrapCode([]byte(tt.in))); got != tt.want {
				t.Fatalf("wrapCode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestWrapCodeSkipsFencedBlocks(t *testing.T) {
	in := "prose -d\n```bash\nrmx -d dir\n```\nmore -r"
	got := string(wrapCode([]byte(in)))

	if !strings.Contains(got, "prose `-d`") {
		t.Errorf("prose flag should be wrapped: %q", got)
	}
	if !strings.Contains(got, "more `-r`") {
		t.Errorf("trailing prose flag should be wrapped: %q", got)
	}
	if !strings.Contains(got, "\nrmx -d dir\n") {
		t.Errorf("fenced content must stay untouched: %q", got)
	}
}

func TestFenceLang(t *testing.T) {
	if got := fenceLang("### options"); got != "text" {
		t.Errorf("options => %q, want text", got)
	}
	if got := fenceLang("### examples"); got != "bash" {
		t.Errorf("examples => %q, want bash", got)
	}
}

func TestGenerateAll(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateAll(dir); err != nil {
		t.Fatalf("GenerateAll: %v", err)
	}

	// A representative file exists and is non-empty in every layout position.
	want := []string{
		"markdown/index.md", "markdown/guide.md", "markdown/rmx/index.md", "markdown/rmx/flags/force.md",
		"man/index.1", "man/rmx/index.1", "man/rmx/flags/recursive.1",
		"yaml/index.yaml", "yaml/rmx/index.yaml", "rest/guide.rst", "rest/rmx/flags/directory.rst",
	}
	// Every registered tool gets a page in every format.
	for _, tool := range cmd.Commands() {
		want = append(want,
			"markdown/"+tool.Name()+"/index.md", "man/"+tool.Name()+"/index.1",
			"yaml/"+tool.Name()+"/index.yaml", "rest/"+tool.Name()+"/index.rst")
	}
	for _, p := range want {
		fi, err := os.Stat(filepath.Join(dir, p))
		if err != nil || fi.Size() == 0 {
			t.Errorf("expected non-empty %s (err=%v)", p, err)
		}
	}

	// Markdown wraps flag references; man stays plain.
	md, _ := os.ReadFile(filepath.Join(dir, "markdown/rmx/flags/directory.md"))
	if !strings.Contains(string(md), "`--dir`") {
		t.Errorf("markdown per-flag page should wrap flags in code")
	}
	man, _ := os.ReadFile(filepath.Join(dir, "man/rmx/flags/directory.1"))
	if strings.Contains(string(man), "`") {
		t.Errorf("man page should contain no backticks")
	}
}

// TestMarkdownLinksResolve checks that every relative link in the markdown
// tree points at a file the generator actually wrote.
func TestMarkdownLinksResolve(t *testing.T) {
	dir := t.TempDir()
	if err := Generate(dir, "markdown"); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	root := filepath.Join(dir, "markdown")
	for rel, content := range readTree(t, root) {
		for _, m := range mdLinkTargetRe.FindAllStringSubmatch(withoutCode(content), -1) {
			target := m[1]
			if strings.Contains(target, "://") {
				continue
			}
			path := filepath.Join(root, filepath.Dir(rel), filepath.FromSlash(target))
			if _, err := os.Stat(path); err != nil {
				t.Errorf("%s links to %q, which does not exist", rel, target)
			}
		}
	}
}

var mdLinkTargetRe = regexp.MustCompile(`\]\(([^)#]+)\)`)

// withoutCode drops fenced code blocks, where markdown has no links (a
// PowerShell example such as "[int](...)" only looks like one).
func withoutCode(md string) string {
	var b strings.Builder
	inCode := false
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "```") {
			inCode = !inCode
			continue
		}
		if !inCode {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func TestGenerateUnknownFormat(t *testing.T) {
	if err := Generate(t.TempDir(), "pdf"); err == nil {
		t.Fatal("expected an error for an unknown format")
	}
}

func TestGenerateAllDeterministic(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	for _, dir := range []string{a, b} {
		if err := GenerateAll(dir); err != nil {
			t.Fatalf("GenerateAll(%s): %v", dir, err)
		}
	}
	if diff := treeDiff(t, a, b); diff != "" {
		t.Fatalf("regeneration is not deterministic: %s", diff)
	}
}

// treeDiff returns a description of the first difference between two directory
// trees, or "" if their file sets and contents are identical.
func treeDiff(t *testing.T, a, b string) string {
	t.Helper()
	fa, fb := readTree(t, a), readTree(t, b)
	if len(fa) != len(fb) {
		return "different file counts"
	}
	for name, content := range fa {
		if fb[name] != content {
			return "differs: " + name
		}
	}
	return ""
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		out[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out
}

func TestGenerateMDX(t *testing.T) {
	dir := t.TempDir()
	extra := []Page{
		{Dir: "rules", Name: "index", Title: "Rules", Description: "Every rule.", Markdown: "# Rules\n\nUse {braces} and <tags>.\n"},
		{Dir: "rules", Name: "no-em-dashes", Title: "Rule: no em dashes", Description: "None.", Markdown: "# Rule: no em dashes\n\nNone.\n"},
	}
	if err := GenerateMDX(dir, extra...); err != nil {
		t.Fatalf("GenerateMDX: %v", err)
	}
	tree := readTree(t, dir)

	if got, want := tree[filepath.FromSlash("rules/index.mdx")], "---\ntitle: \"Rules\"\ndescription: \"Every rule.\"\n---\n\nUse \\{braces\\} and \\<tags>.\n"; got != want {
		t.Errorf("extra page = %q, want %q", got, want)
	}
	// Each folder is an open list under a heading, the hand-written ones first.
	root := tree["meta.json"]
	rules, tools := strings.Index(root, `"---Rules---"`), strings.Index(root, `"---Tools---"`)
	if rules == -1 || tools == -1 || rules > tools ||
		!strings.Contains(root, `"rules/no-em-dashes"`) || strings.Contains(root, `"rules/index"`) ||
		!strings.Contains(root, `"...tools"`) {
		t.Errorf("navigation should list the rules, then the tools, under headings: %s", root)
	}
	if meta := tree[filepath.FromSlash("rules/meta.json")]; !strings.Contains(meta, `"no-em-dashes"`) || !strings.Contains(meta, `"title": "Rules"`) || strings.Contains(meta, `"index"`) {
		t.Errorf("extra folder meta = %s", meta)
	}

	want := []string{"index.mdx", "guide.mdx", "library.mdx", "shell.mdx", "meta.json", "tools/meta.json"}
	for _, tool := range cmd.Tools() {
		if tree[filepath.FromSlash("tools/"+tool.Name()+".mdx")] != "" {
			// A tool without options is a page, not a folder.
			if tree[filepath.FromSlash("tools/"+tool.Name()+"/meta.json")] != "" {
				t.Errorf("%s is both a page and a folder", tool.Name())
			}
			continue
		}
		want = append(want, "tools/"+tool.Name()+"/index.mdx", "tools/"+tool.Name()+"/meta.json")
		for _, topic := range tool.Topics {
			want = append(want, "tools/"+tool.Name()+"/"+topic.Meta.Name+".mdx")
		}
	}
	// Every option has a page, not only the ones with a Topic, and a tool
	// without options is a single page with no folder around it.
	want = append(want, "tools/lsx/all.mdx", "tools/headx/lines.mdx", "tools/truex.mdx")
	if _, folder := tree[filepath.FromSlash("tools/truex/index.mdx")]; folder {
		t.Error("truex has no options, so it should not be a folder")
	}
	for _, p := range want {
		if tree[filepath.FromSlash(p)] == "" {
			t.Errorf("expected non-empty %s", p)
		}
	}

	for name, content := range tree {
		if !strings.HasSuffix(name, ".mdx") {
			continue
		}
		if !strings.HasPrefix(content, "---\ntitle: \"") {
			t.Errorf("%s: missing title frontmatter", name)
		}
		// Outside code, MDX would parse a bare brace or angle bracket. The
		// frontmatter is YAML, not MDX: a quoted description may hold either
		// (the default of xargsx -i is "{}").
		prose := content
		if _, rest, ok := strings.Cut(strings.TrimPrefix(content, "---\n"), "\n---\n"); ok {
			prose = rest
		}
		inFence := false
		for _, line := range strings.Split(prose, "\n") {
			if strings.HasPrefix(line, "```") {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}
			for i, part := range strings.Split(line, "`") {
				if i%2 == 0 && unescapedRe.MatchString(part) {
					t.Errorf("%s: unescaped MDX character in %q", name, line)
				}
			}
		}
		if inFence {
			t.Errorf("%s: unbalanced code fence", name)
		}
	}
}

// unescapedRe finds a brace or angle bracket MDX would parse: one with no
// backslash before it, or with an even run of them (escaped backslashes, which
// leave the character bare).
var unescapedRe = regexp.MustCompile(`(^|[^\\])(\\\\)*[{}<]`)

func TestMDXEscapeKeepsLiteralBackslashes(t *testing.T) {
	// grep's help names the operators \{ and \}: the backslash is text.
	got := mdxEscape(`where \{ \} and \\< are operators`)
	want := `where \\\{ \\\} and \\\\\< are operators`
	if got != want {
		t.Fatalf("mdxEscape = %q, want %q", got, want)
	}
	if unescapedRe.MatchString(got) {
		t.Fatalf("%q still holds a character MDX would parse", got)
	}
}

func TestMDXEscape(t *testing.T) {
	in := "use ${NAME} or <file> but not `${code}`\n```bash\necho ${x} <in\n```\nafter {x}"
	want := "use $\\{NAME\\} or \\<file> but not `${code}`\n```bash\necho ${x} <in\n```\nafter \\{x\\}"
	if got := mdxEscape(in); got != want {
		t.Fatalf("mdxEscape = %q, want %q", got, want)
	}
}

func TestHandwrittenPages(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("rules/README.md", "# Rules\n\nEvery **rule**, see [one](one-rule.md).\n\n| a | b |\n")
	write("rules/one-rule.md", "# Rule: one\n\nThe summary.\n\nBody with links: [index](README.md), [req](../requirements/functional-requirements.md#fr-1), "+
		"[agents](../../AGENTS.md), [code](../../core/core.go), [gen](../markdown/index.md), [web](https://example.com/x.md), [here](#top), "+
		"[page](../../web/src/app/(home)/page.tsx) (aside), [tool](../../documentation/markdown/rmx/index.md), "+
		"[flag](../../documentation/markdown/rmx/flags/force.md), [ref](../../documentation/markdown/index.md), "+
		"[man](../../documentation/man/rmx/index.1).\n")
	write("requirements/functional-requirements.md", "# Functional requirements\n\nWhat it does.\n")
	write("rules/notes.txt", "ignored")

	pages, err := HandwrittenPages(root, "docs", "requirements", "rules", "missing")
	if err != nil {
		t.Fatalf("HandwrittenPages: %v", err)
	}

	var got []string
	byName := map[string]Page{}
	for _, p := range pages {
		got = append(got, p.Dir+"/"+p.Name)
		byName[p.Dir+"/"+p.Name] = p
	}
	if want := "requirements/functional-requirements rules/index rules/one-rule"; strings.Join(got, " ") != want {
		t.Fatalf("pages = %v, want %s", got, want)
	}

	index := byName["rules/index"]
	if index.Title != "Rules" || index.Description != "Every rule, see one." {
		t.Errorf("index title/description = %q / %q", index.Title, index.Description)
	}

	body := byName["rules/one-rule"].Markdown
	if byName["rules/one-rule"].Description != "The summary." || strings.Contains(body, "The summary.") {
		t.Errorf("the summary should be the description and not repeat in the body: %q / %q", byName["rules/one-rule"].Description, body)
	}
	for _, want := range []string{
		"[index](/docs/rules)",
		"[req](/docs/requirements/functional-requirements#fr-1)",
		"[agents](https://github.com/braswelljr/gnux/blob/main/AGENTS.md)",
		"[code](https://github.com/braswelljr/gnux/blob/main/core/core.go)",
		"[gen](https://github.com/braswelljr/gnux/blob/main/docs/markdown/index.md)",
		"[tool](/docs/tools/rmx)",
		"[flag](/docs/tools/rmx/force)",
		"[ref](/docs)",
		"[man](https://github.com/braswelljr/gnux/blob/main/documentation/man/rmx/index.1)",
		"[web](https://example.com/x.md)",
		"[here](#top)",
		"[page](https://github.com/braswelljr/gnux/blob/main/web/src/app/(home)/page.tsx) (aside)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rewritten links missing %q in %q", want, body)
		}
	}
}

func TestPassesMatchesOnlyTheToolsOwnFlags(t *testing.T) {
	lookup := func(name string) tool.Spec {
		t.Helper()
		for _, s := range cmd.Tools() {
			if s.Name() == name {
				return s
			}
		}
		t.Fatalf("no tool %s", name)
		return tool.Spec{}
	}
	flag := func(s tool.Spec, name string) *pflag.Flag {
		t.Helper()
		f := s.Command(core.Discard()).Flags().Lookup(name)
		if f == nil {
			t.Fatalf("%s has no --%s", s.Name(), name)
		}
		return f
	}
	env, rm := lookup("envx"), lookup("rmx")

	for _, c := range []struct {
		spec    tool.Spec
		flag    string
		command string
		want    bool
	}{
		{rm, "force", "rmx -rf build", true},
		{rm, "force", "gnux rm --force x", true},
		{rm, "force", "rmx -r build", false},
		{rm, "recursive", "lsx -R . | rmx x", false},
		// "-" is standard input, not a flag without a short form.
		{rm, "one-file-system", "printf x | rmx -", false},
		// What follows envx's command is that command's.
		{env, "ignore-environment", "envx lsx -i", false},
		{env, "ignore-environment", "envx -i lsx", true},
	} {
		if got := passes(c.spec, c.command, flag(c.spec, c.flag)); got != c.want {
			t.Errorf("passes(%s --%s, %q) = %v, want %v", c.spec.Name(), c.flag, c.command, got, c.want)
		}
	}
}

func TestHandwrittenFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "CHANGELOG.md")
	md := "# Changelog\n\nEvery change, newest first.\n\n## [Unreleased]\n\n- See [the guide](documentation/guides/README.md).\n"
	if err := os.WriteFile(file, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	// Read through a repository-relative path, as gendocs does from the root.
	t.Chdir(dir)
	page, err := HandwrittenFile("CHANGELOG.md", "changelog")
	if err != nil {
		t.Fatalf("HandwrittenFile: %v", err)
	}
	if page.Dir != "" || page.Name != "changelog" || page.Title != "Changelog" || page.Description != "Every change, newest first." {
		t.Errorf("page = %+v", page)
	}
	if strings.Contains(page.Markdown, "Every change, newest first.") {
		t.Errorf("the summary should not repeat in the body: %q", page.Markdown)
	}
	if want := "](https://github.com/braswelljr/gnux/blob/main/documentation/guides/README.md)"; !strings.Contains(page.Markdown, want) {
		t.Errorf("link not rewritten to GitHub: %q", page.Markdown)
	}
}
