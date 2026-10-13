package find

import (
	"os/user"
	"strconv"
)

// idNames caches the names behind numeric owners and groups: a tree usually
// has a handful of them, and a lookup can be a network round trip.
type idNames struct {
	users, groups map[int64]string
}

// lookup returns the name of id, or "" when it has none.
func (n *idNames) lookup(id int64, isUser bool) string {
	cache := &n.groups
	if isUser {
		cache = &n.users
	}
	if *cache == nil {
		*cache = map[int64]string{}
	}
	if name, ok := (*cache)[id]; ok {
		return name
	}
	name := ""
	text := strconv.FormatInt(id, 10)
	if isUser {
		if u, err := user.LookupId(text); err == nil {
			name = u.Username
		}
	} else if g, err := user.LookupGroupId(text); err == nil {
		name = g.Name
	}
	(*cache)[id] = name
	return name
}

// known reports whether id has a name (-nouser and -nogroup are its
// negation).
func (n *idNames) known(id int64, isUser bool) bool {
	return n.lookup(id, isUser) != ""
}

// display returns what %u, %g and -ls print: the name, or the number when
// there is none. An owner the platform does not record (-1) prints as 0.
func (n *idNames) display(id int64, isUser bool) string {
	if id < 0 {
		return "0"
	}
	if name := n.lookup(id, isUser); name != "" {
		return name
	}
	return strconv.FormatInt(id, 10)
}

// lookupID resolves the argument of -user or -group: a name first, then a
// number, as GNU does.
func lookupID(arg string, isUser bool) (int64, bool) {
	if isUser {
		if u, err := user.Lookup(arg); err == nil {
			if id, err := strconv.ParseInt(u.Uid, 10, 64); err == nil {
				return id, true
			}
		}
	} else if g, err := user.LookupGroup(arg); err == nil {
		if id, err := strconv.ParseInt(g.Gid, 10, 64); err == nil {
			return id, true
		}
	}
	id, err := strconv.ParseUint(arg, 10, 31)
	if err != nil {
		return 0, false
	}
	return int64(id), true
}
