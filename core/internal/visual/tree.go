package visual

import "strings"

// Branches are the pieces a tree is drawn with: the branch to a child that
// has siblings after it, the branch to the last child, and the two fillers
// carried down under an ancestor that does or does not have more children.
type Branches struct {
	Mid, Last, Pipe, Blank string
}

var (
	// Unicode draws with box-drawing characters, as eza and tree(1) do.
	Unicode = Branches{Mid: "├── ", Last: "└── ", Pipe: "│   ", Blank: "    "}
	// ASCII is the fallback for a locale that cannot show those.
	ASCII = Branches{Mid: "|-- ", Last: "`-- ", Pipe: "|   ", Blank: "    "}
)

// BranchesFor picks Unicode or ASCII for the environment's locale (UTF8).
func BranchesFor(getenv func(string) string) Branches {
	if UTF8(getenv) {
		return Unicode
	}
	return ASCII
}

// Prefix draws the branches in front of a node. last holds, for each level
// from the top of the tree down to the node itself, whether the node on the
// path at that level is the last of its siblings. A root (no levels) gets no
// prefix. Every piece is the same width, 4 columns per level.
func (b Branches) Prefix(last []bool) string {
	if len(last) == 0 {
		return ""
	}
	var s strings.Builder
	for _, isLast := range last[:len(last)-1] {
		if isLast {
			s.WriteString(b.Blank)
		} else {
			s.WriteString(b.Pipe)
		}
	}
	if last[len(last)-1] {
		s.WriteString(b.Last)
	} else {
		s.WriteString(b.Mid)
	}
	return s.String()
}

// Width is the number of columns Prefix takes for depth levels.
func (b Branches) Width(depth int) int {
	return 4 * depth
}
