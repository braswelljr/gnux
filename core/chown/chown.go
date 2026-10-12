// Package chown implements the engine behind chownx, a GNU-compatible
// implementation of the GNU/UNIX chown command: change the owner and group of
// files.
package chown

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/ownership"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "chownx"

// Traversal says which symbolic links a recursive run follows (-H, -L, -P).
type Traversal = ownership.Traversal

// The traversal modes. TraversePhysical, the zero value, is GNU's default.
const (
	TraversePhysical    = ownership.TraversePhysical
	TraverseCommandLine = ownership.TraverseCommandLine
	TraverseLogical     = ownership.TraverseLogical
)

// Options mirrors the GNU chown flag surface.
type Options struct {
	// Changes (-c) reports only the files whose ownership changed.
	Changes bool
	// Verbose (-v) reports every file processed.
	Verbose bool
	// Silent (-f) suppresses most error messages.
	Silent bool
	// Recursive (-R) changes directories and their contents.
	Recursive bool
	// Traversal selects the symbolic links a recursive run follows.
	Traversal Traversal
	// NoDereference (-h) changes a symbolic link itself, not its referent.
	NoDereference bool
	// Dereference (--dereference) asks explicitly for the referent; with
	// -R it needs -H or -L.
	Dereference bool
	// Reference (--reference=RFILE) takes the owner and group from RFILE
	// instead of an OWNER[:GROUP] operand.
	Reference string
	// From (--from=CURRENT_OWNER:CURRENT_GROUP) changes only the files
	// currently owned as stated.
	From string
	// NoPreserveRoot (--no-preserve-root) allows a recursive run on '/'.
	NoPreserveRoot bool
}

// Chown is the engine. Construct it with New.
type Chown struct {
	Options

	// ChownFn changes a file's owner and group (-1 keeps one), following a
	// symbolic link when follow is true. It is the injection point for tests.
	ChownFn func(path string, uid, gid int, follow bool) error

	eng *ownership.Engine
}

// New builds a chown engine writing to s.
func New(opts Options, s core.Streams) *Chown {
	eng := ownership.New(Name, ownership.Options{
		Changes: opts.Changes, Verbose: opts.Verbose, Silent: opts.Silent,
		Recursive: opts.Recursive, Traversal: opts.Traversal,
		NoDereference: opts.NoDereference, Dereference: opts.Dereference,
		NoPreserveRoot: opts.NoPreserveRoot,
	}, s)
	return &Chown{Options: opts, ChownFn: eng.Chown, eng: eng}
}

// Run changes the owner of each file: "OWNER[:GROUP] FILE..." or, with
// Reference, "FILE...". A failing operand is reported and the rest still
// run.
func (c *Chown) Run(args []string) error {
	if err := c.eng.CheckOptions(); err != nil {
		return err
	}
	c.eng.Chown = c.ChownFn
	diag := c.eng.Diag()

	req := ownership.Request{UID: ownership.Unset, GID: ownership.Unset, FromUID: ownership.Unset, FromGID: ownership.Unset}
	if c.From != "" {
		uid, gid, _, err := ownership.ParseSpec(c.From)
		if err != nil {
			diag.Errorf("%s: '%s'", err, c.From)
			return core.ErrFailed
		}
		req.FromUID, req.FromGID = uid, gid
	}

	if c.Reference != "" {
		uid, gid, err := ownership.Owner(c.Reference)
		if err != nil {
			diag.Errorf("failed to get attributes of '%s': %s", c.Reference, core.ErrText(err))
			return core.ErrFailed
		}
		req.UID, req.GID, req.UserNamed = uid, gid, true
	} else {
		if len(args) == 0 {
			return core.Usagef("missing operand")
		}
		spec := args[0]
		uid, gid, named, err := ownership.ParseSpec(spec)
		if err != nil {
			diag.Errorf("%s: '%s'", err, spec)
			return core.ErrFailed
		}
		req.UID, req.GID, req.UserNamed = uid, gid, named
		if !named && req.GID == ownership.Unset {
			// "" or ":" changes nothing, but still checks the files exist.
			req.UserNamed = true
		}
		args = args[1:]
		if len(args) == 0 {
			return core.Usagef("missing operand after '%s'", spec)
		}
	}
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	return c.eng.Run(req, args)
}
