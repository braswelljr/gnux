package chmod

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Unix permission bits, as chmod(1) spells them.
const (
	bitSetuid = 0o4000
	bitSetgid = 0o2000
	bitSticky = 0o1000
	bitsAll   = 0o7777
	bitsSetID = bitSetuid | bitSetgid
)

var errInvalidMode = errors.New("invalid mode")

// change is a parsed MODE operand, applied to each file in turn: a symbolic
// mode depends on the file's current mode and type, so it cannot be reduced
// to one number up front.
type change struct {
	text  string
	octal bool
	value uint32 // the bits of an octal mode
}

// parseChange validates a MODE operand. Symbolic modes are checked by
// applying them once to an empty mode.
func parseChange(s string) (change, error) {
	if s == "" {
		return change{}, errInvalidMode
	}
	if strings.Trim(s, "01234567") == "" {
		n, err := strconv.ParseUint(s, 8, 32)
		if err != nil || n > bitsAll {
			return change{}, errInvalidMode
		}
		return change{text: s, octal: true, value: uint32(n)}, nil
	}
	c := change{text: s}
	if _, err := c.apply(0, false, 0); err != nil {
		return change{}, err
	}
	return c, nil
}

// mentionsSetID reports whether the mode names the set-user-ID or
// set-group-ID bits itself, which is what lets it change them on a directory.
func (c change) mentionsSetID() bool {
	if c.octal {
		return len(c.text) > 4 || c.value&bitsSetID != 0
	}
	return strings.ContainsRune(c.text, 's')
}

// apply returns the mode that results from applying c to cur. dir selects the
// directory rules: X grants search permission, and, as in GNU chmod, the
// set-user-ID and set-group-ID bits of a directory are kept unless the mode
// names them. umask is honored by clauses that name no user class.
func (c change) apply(cur uint32, dir bool, umask uint32) (uint32, error) {
	var mode uint32
	if c.octal {
		mode = c.value
	} else {
		mode = cur
		for clause := range strings.SplitSeq(c.text, ",") {
			var err error
			if mode, err = applyClause(mode, clause, dir, umask); err != nil {
				return 0, err
			}
		}
	}
	if dir && !c.mentionsSetID() {
		mode = mode&^bitsSetID | cur&bitsSetID
	}
	return mode, nil
}

// applyClause applies one "[ugoa]*([-+=]([rwxXst]*|[ugo]))+" clause to mode.
func applyClause(mode uint32, clause string, dir bool, umask uint32) (uint32, error) {
	i := 0

	var who uint32
	for ; i < len(clause) && strings.IndexByte("ugoa", clause[i]) >= 0; i++ {
		switch clause[i] {
		case 'u':
			who |= bitSetuid | 0o700
		case 'g':
			who |= bitSetgid | 0o070
		case 'o':
			who |= bitSticky | 0o007
		case 'a':
			who |= bitsAll
		}
	}

	// With no user class the clause covers everyone, but bits set in the
	// umask are left alone: "+w" does not hand out group or other write.
	keep := uint32(bitsAll)
	if who == 0 {
		who = bitsAll
		keep = ^umask
	}

	if i == len(clause) {
		return 0, errInvalidMode
	}
	for i < len(clause) {
		op := clause[i]
		if op != '+' && op != '-' && op != '=' {
			return 0, errInvalidMode
		}
		i++

		var bits uint32
		if i < len(clause) && strings.IndexByte("ugo", clause[i]) >= 0 {
			// Copy the permissions another class currently has.
			shift := map[byte]uint{'u': 6, 'g': 3, 'o': 0}[clause[i]]
			v := (mode >> shift) & 0o7
			bits = v<<6 | v<<3 | v
			i++
		} else {
			for ; i < len(clause) && strings.IndexByte("rwxXst", clause[i]) >= 0; i++ {
				switch clause[i] {
				case 'r':
					bits |= 0o444
				case 'w':
					bits |= 0o222
				case 'x':
					bits |= 0o111
				case 'X':
					// Search permission for a directory, or for a file that
					// is already executable by someone.
					if dir || mode&0o111 != 0 {
						bits |= 0o111
					}
				case 's':
					bits |= bitsSetID
				case 't':
					bits |= bitSticky
				}
			}
		}

		bits &= who & keep
		switch op {
		case '+':
			mode |= bits
		case '-':
			mode &^= bits
		case '=':
			mode = mode&^(who&keep) | bits
		}
	}
	return mode, nil
}

// unixBits converts an os.FileMode to Unix permission bits.
func unixBits(m os.FileMode) uint32 {
	bits := uint32(m.Perm())
	if m&os.ModeSetuid != 0 {
		bits |= bitSetuid
	}
	if m&os.ModeSetgid != 0 {
		bits |= bitSetgid
	}
	if m&os.ModeSticky != 0 {
		bits |= bitSticky
	}
	return bits
}

// fileMode converts Unix permission bits to the os.FileMode fsys.Chmod expects.
func fileMode(bits uint32) os.FileMode {
	mode := os.FileMode(bits & 0o777)
	if bits&bitSetuid != 0 {
		mode |= os.ModeSetuid
	}
	if bits&bitSetgid != 0 {
		mode |= os.ModeSetgid
	}
	if bits&bitSticky != 0 {
		mode |= os.ModeSticky
	}
	return mode
}

// permString renders bits the way ls -l does, without the type letter:
// "rwxr-sr-t".
func permString(bits uint32) string {
	b := []byte("rwxrwxrwx")
	for i := range 9 {
		if bits&(1<<(8-i)) == 0 {
			b[i] = '-'
		}
	}
	special := func(i int, set bool, lower, upper byte) {
		if !set {
			return
		}
		if b[i] == 'x' {
			b[i] = lower
		} else {
			b[i] = upper
		}
	}
	special(2, bits&bitSetuid != 0, 's', 'S')
	special(5, bits&bitSetgid != 0, 's', 'S')
	special(8, bits&bitSticky != 0, 't', 'T')
	return string(b)
}

// describe renders a mode as GNU chmod's messages do: "0755 (rwxr-xr-x)".
func describe(bits uint32) string {
	return fmt.Sprintf("%04o (%s)", bits&bitsAll, permString(bits))
}
