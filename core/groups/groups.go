// Package groups implements the engine behind groupsx, a GNU-compatible
// implementation of the GNU groups command: print the groups a user is in.
// The identities are core/id's, which idx -Gn prints.
package groups

import (
	"bufio"
	"os"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/id"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "groupsx"

// Options holds gnux's terminal view only: GNU groups has no options of
// its own, and the zero value is its output.
type Options struct {
	// Pretty, Color and Theme are gnux's terminal view: with Pretty and
	// Color, each user's primary group is in the theme's "yours" color,
	// the others in its "others" color and a USER operand in the heading
	// color (Theme is a built-in theme; "" takes GNUX_THEME's). The text
	// is GNU's.
	Pretty bool
	Color  bool
	Theme  string
}

// Groups is the engine. Construct it with New.
type Groups struct {
	Options

	// Self returns the identity of the running process; it defaults to
	// id.Self.
	Self func() (id.Identity, error)
	// Lookup returns the identity of a named user; it defaults to
	// id.LookupUser.
	Lookup func(name string) (id.Identity, error)
	// GroupName maps a group ID to its name; it defaults to id.GroupName.
	GroupName func(gid string) (string, bool)
	// Getenv reads GNUX_THEME for the view; nil means os.Getenv.
	Getenv func(string) string

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	theme   *visual.Theme // the view's colors, nil when uncolored
}

// New builds a groups engine writing to s.
func New(opts Options, s core.Streams) *Groups {
	return &Groups{
		Options:   opts,
		Self:      id.Self,
		Lookup:    id.LookupUser,
		GroupName: id.GroupName,
		streams:   s,
		diag:      core.NewDiag(Name, s),
	}
}

// Run prints the names of the groups of the running process or, for each
// USER operand, 'USER : ' and that user's groups. An unknown user is
// reported and the remaining operands still run. It returns core.ErrFailed
// when anything failed, and nil otherwise.
func (g *Groups) Run(args []string) (err error) {
	g.out = core.NewWriter(g.streams.Out)
	defer core.FlushOutput(g.out, g.diag, &err)
	g.theme = nil
	if g.Pretty && g.Color {
		getenv := g.Getenv
		if getenv == nil {
			getenv = os.Getenv
		}
		g.theme = visual.Use(true, g.Theme, getenv, g.diag.Warnf)
	}

	if len(args) == 0 {
		self, err := g.Self()
		if err != nil {
			g.diag.Errorf("cannot get the current user's groups: %s", core.ErrText(err))
			return g.diag.Result()
		}
		g.print(self)
		return g.diag.Result()
	}
	for _, name := range args {
		who, err := g.Lookup(name)
		if err != nil {
			_ = g.out.Flush()
			g.diag.Errorf("'%s': no such user", name)
			continue
		}
		_, _ = g.out.WriteString(g.theme.Paint(visual.Heading, name) + g.theme.Paint(visual.Punctuation, " : "))
		g.print(who)
	}
	return g.diag.Result()
}

// print writes the group names of who on one line. A group with no name is
// reported and printed as its number, as GNU does.
func (g *Groups) print(who id.Identity) {
	for i, gid := range id.GroupList(who) {
		if i > 0 {
			_ = g.out.WriteByte(' ')
		}
		name, ok := g.GroupName(gid)
		if !ok {
			_ = g.out.Flush()
			g.diag.Errorf("cannot find name for group ID %s", gid)
			name = gid
		}
		// gnux's terminal view (BR-13): the primary group stands out.
		key := visual.Others
		if gid == who.GID {
			key = visual.You
		}
		_, _ = g.out.WriteString(g.theme.Paint(key, name))
	}
	_ = g.out.WriteByte('\n')
}
