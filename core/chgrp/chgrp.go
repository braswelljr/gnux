// Package chgrp implements the engine behind chgrpx, a GNU-compatible
// implementation of the GNU/UNIX chgrp command: change the group of files.
package chgrp

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/ownership"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "chgrpx"

// Traversal says which symbolic links a recursive run follows (-H, -L, -P).
type Traversal = ownership.Traversal

// The traversal modes. TraversePhysical, the zero value, is GNU's default.
const (
	TraversePhysical    = ownership.TraversePhysical
	TraverseCommandLine = ownership.TraverseCommandLine
	TraverseLogical     = ownership.TraverseLogical
)

// Options mirrors the GNU chgrp flag surface.
type Options struct {
	// Changes (-c) reports only the files whose group changed.
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
	// Reference (--reference=RFILE) takes the group from RFILE instead of a
	// GROUP operand.
	Reference string
	// NoPreserveRoot (--no-preserve-root) allows a recursive run on '/'.
	NoPreserveRoot bool
}

// Chgrp is the engine. Construct it with New.
type Chgrp struct {
	Options

	// ChownFn changes a file's owner and group (-1 keeps one), following a
	// symbolic link when follow is true. It is the injection point for tests.
	ChownFn func(path string, uid, gid int, follow bool) error

	eng *ownership.Engine
}

// New builds a chgrp engine writing to s.
func New(opts Options, s core.Streams) *Chgrp {
	eng := ownership.New(Name, ownership.Options{
		Changes: opts.Changes, Verbose: opts.Verbose, Silent: opts.Silent,
		Recursive: opts.Recursive, Traversal: opts.Traversal,
		NoDereference: opts.NoDereference, Dereference: opts.Dereference,
		NoPreserveRoot: opts.NoPreserveRoot,
	}, s)
	return &Chgrp{Options: opts, ChownFn: eng.Chown, eng: eng}
}

// Run changes the group of each file: "GROUP FILE..." or, with Reference,
// "FILE...". A failing operand is reported and the rest still run.
func (c *Chgrp) Run(args []string) error {
	if err := c.eng.CheckOptions(); err != nil {
		return err
	}
	c.eng.Chown = c.ChownFn
	diag := c.eng.Diag()

	req := ownership.Request{UID: ownership.Unset, FromUID: ownership.Unset, FromGID: ownership.Unset}
	if c.Reference != "" {
		_, gid, err := ownership.Owner(c.Reference)
		if err != nil {
			diag.Errorf("failed to get attributes of '%s': %s", c.Reference, core.ErrText(err))
			return core.ErrFailed
		}
		req.GID = gid
	} else {
		if len(args) == 0 {
			return core.Usagef("missing operand")
		}
		group := args[0]
		args = args[1:]
		if len(args) == 0 {
			return core.Usagef("missing operand after '%s'", group)
		}
		req.GID = ownership.Unset
		if group != "" {
			gid, err := ownership.LookupGroup(group)
			if err != nil {
				diag.Errorf("invalid group: '%s'", group)
				return core.ErrFailed
			}
			req.GID = gid
		}
	}
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	return c.eng.Run(req, args)
}
