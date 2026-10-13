package find

import (
	"io/fs"
	"strconv"
	"strings"
)

// unixPerm returns the twelve permission bits of mode in the Unix layout:
// rwx for user, group and other, plus setuid, setgid and sticky.
func unixPerm(mode fs.FileMode) uint32 {
	bits := uint32(mode.Perm())
	if mode&fs.ModeSetuid != 0 {
		bits |= 0o4000
	}
	if mode&fs.ModeSetgid != 0 {
		bits |= 0o2000
	}
	if mode&fs.ModeSticky != 0 {
		bits |= 0o1000
	}
	return bits
}

// parsePerm parses the argument of -perm: MODE (exactly these bits), -MODE
// (all of them) or /MODE (any of them), where MODE is octal or symbolic.
// kind is 0, '-' or '/'.
func parsePerm(arg string) (kind byte, bits uint32, ok bool) {
	mode := arg
	if mode != "" && (mode[0] == '-' || mode[0] == '/') {
		kind, mode = mode[0], mode[1:]
	}
	if mode == "" {
		return 0, 0, false
	}
	if mode[0] >= '0' && mode[0] <= '7' {
		n, err := strconv.ParseUint(mode, 8, 32)
		if err != nil || n > 0o7777 {
			return 0, 0, false
		}
		return kind, uint32(n), true
	}
	bits, ok = symbolicMode(mode)
	return kind, bits, ok
}

// symbolicMode evaluates a chmod-style symbolic mode (u=rwx,g+r,o-w) the way
// GNU find does: applied to a file with no permission bits, with no umask.
func symbolicMode(spec string) (uint32, bool) {
	var mode uint32
	for _, clause := range strings.Split(spec, ",") {
		i := 0
		var who uint32
		for ; i < len(clause) && strings.IndexByte("ugoa", clause[i]) >= 0; i++ {
			who |= map[byte]uint32{'u': 0o4700, 'g': 0o2070, 'o': 0o1007, 'a': 0o7777}[clause[i]]
		}
		if who == 0 {
			who = 0o7777
		}
		if i >= len(clause) {
			return 0, false
		}
		for i < len(clause) {
			op := clause[i]
			if op != '+' && op != '-' && op != '=' {
				return 0, false
			}
			i++

			var bits uint32
			if i < len(clause) && strings.IndexByte("ugo", clause[i]) >= 0 {
				// Copy another class's permissions.
				shift := map[byte]uint{'u': 6, 'g': 3, 'o': 0}[clause[i]]
				bits = (mode >> shift & 7) * 0o111
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
						// Execute only where something is already
						// executable; the starting mode is never a
						// directory.
						if mode&0o111 != 0 {
							bits |= 0o111
						}
					case 's':
						bits |= 0o6000
					case 't':
						bits |= 0o1000
					}
				}
			}
			bits &= who

			switch op {
			case '+':
				mode |= bits
			case '-':
				mode &^= bits
			default:
				mode = mode&^who | bits
			}
		}
	}
	return mode, true
}
