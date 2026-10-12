// Package ownership is the engine shared by chownx and chgrpx: changing the
// owner and group of files, with GNU chown's reporting, traversal and
// symbolic link rules. chgrp GROUP is chown :GROUP with group-only messages,
// so both tools are thin wrappers around Run.
package ownership

import (
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/braswelljr/gnux/core"
)

// Traversal says which symbolic links a recursive run follows (-H, -L, -P).
type Traversal int

const (
	// TraversePhysical (-P) follows no symbolic link: each link met, operand
	// or not, is changed itself. It is the default, as in GNU.
	TraversePhysical Traversal = iota
	// TraverseCommandLine (-H) follows a symbolic link given as an operand.
	TraverseCommandLine
	// TraverseLogical (-L) follows every symbolic link.
	TraverseLogical
)

// Unset marks an ID that is not to be changed.
const Unset = -1

// Options is the shared flag surface of chown and chgrp.
type Options struct {
	Changes        bool      // -c
	Verbose        bool      // -v
	Silent         bool      // -f
	Recursive      bool      // -R
	Traversal      Traversal // -H, -L, -P
	NoDereference  bool      // -h: change a symbolic link itself
	Dereference    bool      // --dereference given explicitly
	NoPreserveRoot bool      // --no-preserve-root
}

// Request is what to change: the new IDs (Unset to keep), and the --from
// filter (Unset for "any").
type Request struct {
	UID, GID         int
	FromUID, FromGID int
	// UserNamed reports that the request names a user, which selects the
	// "ownership" wording over the "group" wording in messages.
	UserNamed bool
}

// Hooks are the engine's injection points.
type Hooks struct {
	// Chown changes the owner and group of path, following a symbolic link
	// when follow is true. It defaults to os.Chown / os.Lchown.
	Chown func(path string, uid, gid int, follow bool) error
}

// Engine changes ownership. Build it with New.
type Engine struct {
	Options
	Hooks

	diag   *core.Diag
	req    Request
	failed bool
	names  map[string]string
}

// New builds an engine whose diagnostics are prefixed with prog.
func New(prog string, opts Options, s core.Streams) *Engine {
	return &Engine{
		Options: opts,
		Hooks:   Hooks{Chown: chown},
		diag:    core.NewDiag(prog, s),
		names:   map[string]string{},
	}
}

// Diag returns the engine's diagnostics writer, for the wrappers' own
// messages.
func (e *Engine) Diag() *core.Diag {
	return e.diag
}

// CheckOptions refuses the combinations GNU refuses.
func (e *Engine) CheckOptions() error {
	if e.Recursive && e.Traversal == TraversePhysical && e.Dereference && !e.NoDereference {
		return core.Usagef("-R --dereference requires either -H or -L")
	}
	return nil
}

// Run applies req to every file. The returned error is core.ErrFailed if
// any operand failed.
func (e *Engine) Run(req Request, files []string) error {
	e.req = req
	for _, f := range files {
		e.operand(f)
	}
	if e.failed || e.diag.Failed() {
		return core.ErrFailed
	}
	return nil
}

func (e *Engine) errorf(format string, a ...any) {
	e.failed = true
	if !e.Silent {
		e.diag.Errorf(format, a...)
	}
}

// followOperand reports whether a symbolic link given as an operand is
// followed: always without -R (unless -h), and with -R only under -H or -L.
func (e *Engine) followOperand() bool {
	if e.NoDereference {
		return false
	}
	return !e.Recursive || e.Traversal != TraversePhysical
}

func (e *Engine) operand(path string) {
	info, err := os.Lstat(path)
	if err != nil {
		e.errorf("cannot access '%s': %s", path, core.ErrText(err))
		return
	}
	// An operand is changed through a symbolic link unless -h (or -R
	// without -H/-L) says otherwise; for any other file the two are the same.
	follow := e.followOperand()
	if info.Mode()&os.ModeSymlink != 0 && follow {
		target, err := os.Stat(path)
		if err != nil {
			e.errorf("cannot dereference '%s': %s", path, core.ErrText(err))
			return
		}
		info = target
	}
	if e.Recursive && info.IsDir() && !e.NoPreserveRoot && core.IsRoot(path) {
		if filepath.Clean(path) == "/" {
			e.errorf("it is dangerous to operate recursively on '%s'", path)
		} else {
			e.errorf("it is dangerous to operate recursively on '%s' (same as '/')", path)
		}
		e.errorf("use --no-preserve-root to override this failsafe")
		return
	}
	e.visit(path, info, follow, nil)
}

// visit changes path and, when recursing into a directory, its contents.
func (e *Engine) visit(path string, info os.FileInfo, follow bool, ancestors []os.FileInfo) {
	e.change(path, info, follow)
	if !e.Recursive || !info.IsDir() {
		return
	}
	for _, a := range ancestors {
		if os.SameFile(a, info) {
			e.errorf("WARNING: Circular directory structure.\nThis almost certainly means that you have a corrupted file system.\nNOTIFY YOUR SYSTEM MANAGER.\nThe following directory is part of the cycle:\n  '%s'", path)
			return
		}
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		e.errorf("cannot read directory '%s': %s", path, core.ErrText(err))
		return
	}
	ancestors = append(ancestors, info)
	for _, ent := range entries {
		child := filepath.Join(path, ent.Name())
		ci, err := os.Lstat(child)
		if err != nil {
			e.errorf("cannot access '%s': %s", child, core.ErrText(err))
			continue
		}
		childFollow := false
		if ci.Mode()&os.ModeSymlink != 0 && e.Traversal == TraverseLogical && !e.NoDereference {
			if target, err := os.Stat(child); err == nil {
				ci, childFollow = target, true
			}
		}
		e.visit(child, ci, childFollow, ancestors)
	}
}

// change applies the request to one file and reports it.
func (e *Engine) change(path string, info os.FileInfo, follow bool) {
	oldUID, oldGID, known := ownerOf(info)

	// --from: only files currently owned as stated are changed.
	if e.req.FromUID != Unset || e.req.FromGID != Unset {
		if !known || !matches(e.req.FromUID, oldUID) || !matches(e.req.FromGID, oldGID) {
			if e.Verbose {
				e.diag.Printf("%s of '%s' retained", e.noun(), path)
			}
			return
		}
	}

	uid, gid := e.req.UID, e.req.GID
	if uid != Unset || gid != Unset {
		if err := e.Chown(path, uid, gid, follow); err != nil {
			if e.Verbose && known {
				e.diag.Printf("failed to change %s of '%s' from %s to %s", e.noun(), path,
					e.spec(oldUID, oldGID), e.spec(pick(uid, oldUID), pick(gid, oldGID)))
			}
			e.errorf("changing %s of '%s': %s", e.noun(), path, errText(err))
			return
		}
	}

	newUID, newGID := pick(uid, oldUID), pick(gid, oldGID)
	switch {
	case !known:
	case newUID != oldUID || newGID != oldGID:
		if e.Verbose || e.Changes {
			e.diag.Printf("changed %s of '%s' from %s to %s", e.noun(), path,
				e.spec(oldUID, oldGID), e.spec(newUID, newGID))
		}
	case e.Verbose:
		e.diag.Printf("%s of '%s' retained as %s", e.noun(), path, e.spec(newUID, newGID))
	}
}

// noun is "ownership" when a user is named, "group" otherwise.
func (e *Engine) noun() string {
	if e.req.UserNamed {
		return "ownership"
	}
	return "group"
}

// spec renders IDs as messages show them: "user:group" when the request
// names both, the user alone, or the group alone.
func (e *Engine) spec(uid, gid int) string {
	switch {
	case e.req.UserNamed && e.req.GID != Unset:
		return e.userName(uid) + ":" + e.groupName(gid)
	case e.req.UserNamed:
		return e.userName(uid)
	default:
		return e.groupName(gid)
	}
}

func (e *Engine) userName(uid int) string {
	key := "u" + strconv.Itoa(uid)
	if n, ok := e.names[key]; ok {
		return n
	}
	n := strconv.Itoa(uid)
	if u, err := user.LookupId(n); err == nil {
		n = u.Username
	}
	e.names[key] = n
	return n
}

func (e *Engine) groupName(gid int) string {
	key := "g" + strconv.Itoa(gid)
	if n, ok := e.names[key]; ok {
		return n
	}
	n := strconv.Itoa(gid)
	if g, err := user.LookupGroupId(n); err == nil {
		n = g.Name
	}
	e.names[key] = n
	return n
}

func matches(want, have int) bool {
	return want == Unset || want == have
}

func pick(want, have int) int {
	if want == Unset {
		return have
	}
	return want
}

// errText is core.ErrText, with EPERM spelled as GNU spells it.
func errText(err error) string {
	var perr *fs.PathError
	if errors.As(err, &perr) && errors.Is(perr.Err, syscall.EPERM) {
		return "Operation not permitted"
	}
	return core.ErrText(err)
}

// LookupUser resolves a user name or number to its UID and primary GID.
// A name is tried first, then a number, as in GNU; "+N" is always a number.
func LookupUser(name string) (uid, gid int, err error) {
	if !strings.HasPrefix(name, "+") {
		if u, lerr := user.Lookup(name); lerr == nil {
			uid, _ = strconv.Atoi(u.Uid)
			gid, gerr := strconv.Atoi(u.Gid)
			if gerr != nil {
				gid = Unset
			}
			return uid, gid, nil
		}
	}
	n, perr := strconv.ParseUint(strings.TrimPrefix(name, "+"), 10, 31)
	if perr != nil {
		return 0, 0, errors.New("invalid user")
	}
	gid = Unset
	if u, lerr := user.LookupId(strconv.FormatUint(n, 10)); lerr == nil {
		if g, gerr := strconv.Atoi(u.Gid); gerr == nil {
			gid = g
		}
	}
	return int(n), gid, nil
}

// LookupGroup resolves a group name or number to its GID.
func LookupGroup(name string) (int, error) {
	if !strings.HasPrefix(name, "+") {
		if g, err := user.LookupGroup(name); err == nil {
			return strconv.Atoi(g.Gid)
		}
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(name, "+"), 10, 31)
	if err != nil {
		return 0, errors.New("invalid group")
	}
	return int(n), nil
}

// ParseSpec parses chown's OWNER[:[GROUP]] (or the obsolete OWNER.GROUP) into
// a request. "OWNER:" also sets the group to the owner's login group. The
// error text is GNU's: "invalid user", "invalid group" or "invalid spec".
func ParseSpec(spec string) (uid, gid int, userNamed bool, err error) {
	uid, gid = Unset, Unset
	userPart, groupPart, colon := strings.Cut(spec, ":")
	if !colon && strings.Contains(spec, ".") {
		// The obsolete separator, unless the whole thing names a user.
		if _, _, lerr := LookupUser(spec); lerr != nil {
			userPart, groupPart, colon = strings.Cut(spec, ".")
		}
	}

	if userPart != "" {
		u, loginGID, lerr := LookupUser(userPart)
		if lerr != nil {
			return 0, 0, false, errors.New("invalid user")
		}
		uid, userNamed = u, true
		if colon && groupPart == "" {
			if loginGID == Unset {
				return 0, 0, false, errors.New("invalid spec")
			}
			gid = loginGID
		}
	}
	if groupPart != "" {
		g, lerr := LookupGroup(groupPart)
		if lerr != nil {
			return 0, 0, false, errors.New("invalid group")
		}
		gid = g
	}
	return uid, gid, userNamed, nil
}

// Owner returns the owner and group of the file at path (following a
// symbolic link), for --reference.
func Owner(path string) (uid, gid int, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	uid, gid, ok := ownerOf(info)
	if !ok {
		return 0, 0, errNotSupported
	}
	return uid, gid, nil
}
