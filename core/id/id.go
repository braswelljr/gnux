// Package id implements the engine behind idx, a GNU-compatible
// implementation of the GNU/UNIX id command: print the user and group IDs of
// the current process or of the named users.
package id

import (
	"bufio"
	"fmt"
	"os/user"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "idx"

// Options mirrors the GNU id flag surface. The zero value prints every ID in
// the default format.
type Options struct {
	// User (-u) prints only the effective user ID.
	User bool
	// Group (-g) prints only the effective group ID.
	Group bool
	// Groups (-G) prints every group ID.
	Groups bool
	// Name (-n) prints names instead of numbers, with -u, -g or -G.
	Name bool
	// Real (-r) prints the real ID instead of the effective one, with -u,
	// -g or -G.
	Real bool
	// Zero (-z) ends each entry with NUL instead of whitespace, outside the
	// default format.
	Zero bool
}

// Identity is a set of user and group IDs. IDs are strings: numbers on Unix,
// security identifiers (SIDs) on Windows.
type Identity struct {
	UID, GID   string // real IDs
	EUID, EGID string // effective IDs
	Groups     []string
}

// ID is the engine. Construct it with New.
type ID struct {
	Options

	// Self returns the identity of the running process.
	Self func() (Identity, error)
	// Lookup returns the identity of a named (or numbered) user.
	Lookup func(name string) (Identity, error)
	// UserName and GroupName map an ID to a name.
	UserName  func(id string) (string, bool)
	GroupName func(id string) (string, bool)

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
}

// New builds an id engine writing to s.
func New(opts Options, s core.Streams) *ID {
	return &ID{
		Options:   opts,
		Self:      selfIdentity,
		Lookup:    lookupUser,
		UserName:  userName,
		GroupName: groupName,
		streams:   s,
		diag:      core.NewDiag(Name, s),
	}
}

// Run prints the IDs of the process, or of each USER operand in turn. An
// unknown user is reported and the remaining operands still run.
func (i *ID) Run(args []string) (err error) {
	only := 0
	for _, b := range []bool{i.User, i.Group, i.Groups} {
		if b {
			only++
		}
	}
	switch {
	case only > 1:
		return core.Usagef("cannot print \"only\" of more than one choice")
	case only == 0 && (i.Name || i.Real):
		return core.Usagef("cannot print only names or real IDs in default format")
	case only == 0 && i.Zero:
		return core.Usagef("option --zero not permitted in default format")
	}

	i.out = bufio.NewWriter(i.streams.Out)
	defer core.FlushOutput(i.out, i.diag, &err)

	if len(args) == 0 {
		self, err := i.Self()
		if err != nil {
			i.diag.Errorf("cannot get the current user's IDs: %s", core.ErrText(err))
			return i.diag.Result()
		}
		i.print(self)
		return i.diag.Result()
	}
	for _, name := range args {
		who, err := i.Lookup(name)
		if err != nil {
			_ = i.out.Flush()
			i.diag.Errorf("'%s': no such user", name)
			continue
		}
		i.print(who)
	}
	return i.diag.Result()
}

func (i *ID) print(who Identity) {
	end := "\n"
	if i.Zero {
		end = "\x00"
	}
	switch {
	case i.User:
		uid := who.EUID
		if i.Real {
			uid = who.UID
		}
		_, _ = i.out.WriteString(i.userField(uid) + end)
	case i.Group:
		gid := who.EGID
		if i.Real {
			gid = who.GID
		}
		_, _ = i.out.WriteString(i.groupField(gid) + end)
	case i.Groups:
		sep := " "
		if i.Zero {
			sep = "\x00"
		}
		var fields []string
		for _, g := range groupList(who) {
			fields = append(fields, i.groupField(g))
		}
		_, _ = i.out.WriteString(strings.Join(fields, sep) + end)
	default:
		_, _ = i.out.WriteString(i.full(who) + "\n")
	}
}

// userField is a user ID, or its name under -n. A missing name is reported
// and the number printed instead, as GNU does.
func (i *ID) userField(uid string) string {
	if !i.Name {
		return uid
	}
	if name, ok := i.UserName(uid); ok {
		return name
	}
	_ = i.out.Flush()
	i.diag.Errorf("cannot find name for user ID %s", uid)
	return uid
}

func (i *ID) groupField(gid string) string {
	if !i.Name {
		return gid
	}
	if name, ok := i.GroupName(gid); ok {
		return name
	}
	_ = i.out.Flush()
	i.diag.Errorf("cannot find name for group ID %s", gid)
	return gid
}

// full renders the default format:
// uid=N(name) gid=N(name) [euid=N(name)] [egid=N(name)] groups=N(name),...
func (i *ID) full(who Identity) string {
	named := func(id string, lookup func(string) (string, bool)) string {
		if name, ok := lookup(id); ok {
			return fmt.Sprintf("%s(%s)", id, name)
		}
		return id
	}
	var b strings.Builder
	b.WriteString("uid=" + named(who.UID, i.UserName))
	b.WriteString(" gid=" + named(who.GID, i.GroupName))
	if who.EUID != who.UID {
		b.WriteString(" euid=" + named(who.EUID, i.UserName))
	}
	if who.EGID != who.GID {
		b.WriteString(" egid=" + named(who.EGID, i.GroupName))
	}
	if groups := groupList(who); len(groups) > 0 {
		var fields []string
		for _, g := range groups {
			fields = append(fields, named(g, i.GroupName))
		}
		b.WriteString(" groups=" + strings.Join(fields, ","))
	}
	return b.String()
}

// groupList is the real group, then the effective one when it differs, then
// the supplementary groups, each once, as GNU id lists them.
func groupList(who Identity) []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range append([]string{who.GID, who.EGID}, who.Groups...) {
		if g == "" || seen[g] {
			continue
		}
		seen[g] = true
		out = append(out, g)
	}
	return out
}

// lookupUser resolves a user by name, then by number.
func lookupUser(name string) (Identity, error) {
	u, err := user.Lookup(name)
	if err != nil {
		var idErr error
		u, idErr = user.LookupId(strings.TrimPrefix(name, "+"))
		if idErr != nil {
			return Identity{}, err
		}
	}
	groups, _ := u.GroupIds()
	return Identity{UID: u.Uid, GID: u.Gid, EUID: u.Uid, EGID: u.Gid, Groups: groups}, nil
}

func userName(uid string) (string, bool) {
	u, err := user.LookupId(uid)
	if err != nil || u.Username == "" {
		return "", false
	}
	return u.Username, true
}

func groupName(gid string) (string, bool) {
	g, err := user.LookupGroupId(gid)
	if err != nil || g.Name == "" {
		return "", false
	}
	return g.Name, true
}
