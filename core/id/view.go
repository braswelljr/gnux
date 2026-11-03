package id

import (
	"fmt"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core/internal/visual"
)

// panel is gnux's terminal view of the default format (BR-13): a box
// titled with the user's name, one labeled line an ID and its name, the
// effective IDs only where they differ, and every group on a line of its
// own, the primary one in the "yours" color.
func (i *ID) panel(who Identity) string {
	getenv := i.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	t := visual.Use(i.Color, i.Theme, getenv, i.diag.Warnf)
	type row struct{ label, id, name, key string }
	named := func(label, id string, lookup func(string) (string, bool), key string) row {
		name, _ := lookup(id)
		return row{label, id, name, key}
	}
	rows := []row{named("User", who.UID, i.UserName, visual.You), named("Group", who.GID, i.GroupName, visual.You)}
	if who.EUID != who.UID {
		rows = append(rows, named("Effective user", who.EUID, i.UserName, visual.Bad))
	}
	if who.EGID != who.GID {
		rows = append(rows, named("Effective group", who.EGID, i.GroupName, visual.Bad))
	}
	for n, g := range groupList(who) {
		label := ""
		if n == 0 {
			label = "Groups"
		}
		key := visual.Others
		if g == who.GID {
			key = visual.You
		}
		rows = append(rows, named(label, g, i.GroupName, key))
	}

	labelWidth, idWidth := 0, 0
	for _, r := range rows {
		labelWidth, idWidth = max(labelWidth, len(r.label)), max(idWidth, len(r.id))
	}
	lines := make([]string, len(rows))
	for n, r := range rows {
		lines[n] = t.Paint(visual.Punctuation, fmt.Sprintf("%-*s", labelWidth, r.label)) + "  " +
			fmt.Sprintf("%*s", idWidth, r.id) + "  " + t.Paint(r.key, r.name)
		lines[n] = strings.TrimRight(lines[n], " ")
	}
	title := rows[0].name
	if title == "" {
		title = "uid " + who.UID
	}
	return t.Box(title, lines, visual.UTF8(getenv))
}
