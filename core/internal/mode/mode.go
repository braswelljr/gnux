// Package mode reads the MODE argument of mkdir -m, mkfifo -m and mknod -m:
// an octal number or chmod's symbolic clauses, applied to the mode a new
// file starts with.
package mode

import (
	"errors"
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
)

// ErrInvalid is a MODE that is not a valid octal or symbolic mode.
var ErrInvalid = errors.New("invalid mode")

// Parse turns a chmod-style MODE into Unix permission bits for a new file.
// Octal modes are taken literally; symbolic modes are applied to base, the
// point of departure the tool documents (a=rwx for mkdir, a=rw for mkfifo).
// X adds search permission for a directory, or when base already has some.
// umask is only called for clauses that name no user class.
func Parse(s string, base uint32, dir bool, umask func() uint32) (uint32, error) {
	if s == "" {
		return 0, ErrInvalid
	}
	if strings.Trim(s, "01234567") == "" {
		n, err := strconv.ParseUint(s, 8, 32)
		if err != nil || n > bitsAll {
			return 0, ErrInvalid
		}
		return uint32(n), nil
	}

	mode := base
	for clause := range strings.SplitSeq(s, ",") {
		var err error
		if mode, err = applyClause(mode, clause, dir, umask); err != nil {
			return 0, err
		}
	}
	return mode, nil
}

// applyClause applies one "[ugoa]*([-+=]([rwxXst]*|[ugo]))+" clause to mode.
func applyClause(mode uint32, clause string, dir bool, umask func() uint32) (uint32, error) {
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
	// umask are left alone - "+w" does not hand out group or other write.
	keep := uint32(bitsAll)
	if who == 0 {
		who = bitsAll
		keep = ^umask()
	}

	if i == len(clause) {
		return 0, ErrInvalid
	}
	for i < len(clause) {
		op := clause[i]
		if op != '+' && op != '-' && op != '=' {
			return 0, ErrInvalid
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
					// Search permission for a directory, or for a file
					// that someone may already search.
					if dir || mode&0o111 != 0 {
						bits |= 0o111
					}
				case 's':
					bits |= bitSetuid | bitSetgid
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
			mode = mode&^who | bits
		}
	}
	return mode, nil
}

// FileMode converts Unix permission bits to the os.FileMode fsys.Chmod expects.
func FileMode(bits uint32) os.FileMode {
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
