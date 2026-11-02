package uniq

import (
	"os"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// gnux's terminal view of -c (BR-13): each count, still in GNU's 7-column
// field, is colored by its share of the largest count, and a bar of that
// share stands between it and the line, so sort | uniq -c reads as a
// histogram.

// viewLimit is how many groups the view holds to find the largest count.
// Past it, the groups held are drawn against the largest count among them,
// and so is every group after, as it comes, so a long input still streams.
const viewLimit = 4096

// barWidth is the width of a count's bar.
const barWidth = 20

type countView struct {
	theme   *visual.Theme
	unicode bool
	rows    []countRow
	max     int64
	frozen  bool // past viewLimit: drawing as the groups come
}

type countRow struct {
	n    int64
	line []byte
}

func (u *Uniq) prepareView(toStdout bool) {
	u.view = nil
	if !u.Pretty || !u.Count || u.ZeroTerminated || !toStdout {
		return
	}
	getenv := u.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	u.view = &countView{
		theme:   visual.Use(u.Color, u.Theme, getenv, u.diag.Warnf),
		unicode: visual.UTF8(getenv),
	}
}

// add takes one counted group.
func (u *Uniq) add(n int64, line []byte) {
	v := u.view
	if v.frozen {
		u.drawRow(n, line)
		return
	}
	v.max = max(v.max, n)
	v.rows = append(v.rows, countRow{n, append([]byte(nil), line...)})
	if len(v.rows) == viewLimit {
		u.drawView()
		v.frozen = true
	}
}

// drawView draws the groups held.
func (u *Uniq) drawView() {
	v := u.view
	if v == nil {
		return
	}
	for _, r := range v.rows {
		u.drawRow(r.n, r.line)
	}
	v.rows = nil
}

func (u *Uniq) drawRow(n int64, line []byte) {
	v := u.view
	fraction := 1.0
	if v.max > 0 {
		fraction = min(float64(n)/float64(v.max), 1)
	}
	count := string(core.AppendInt(nil, n, 7))
	_, _ = u.out.WriteString(v.theme.Paint(shareKey(fraction), count))
	_ = u.out.WriteByte(' ')
	_, _ = u.out.WriteString(v.theme.Progress(fraction, barWidth, v.unicode))
	_ = u.out.WriteByte(' ')
	_, _ = u.out.Write(line)
	_ = u.out.WriteByte(u.delim)
}

// shareKey puts a share of the largest count on the size gradient, from
// the smallest step to the largest.
func shareKey(fraction float64) string {
	switch {
	case fraction >= 0.8:
		return visual.SizeTera
	case fraction >= 0.6:
		return visual.SizeGiga
	case fraction >= 0.4:
		return visual.SizeMega
	case fraction >= 0.2:
		return visual.SizeKilo
	}
	return visual.SizeByte
}
