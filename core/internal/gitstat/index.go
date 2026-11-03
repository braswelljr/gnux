package gitstat

import (
	"crypto/sha1" //nolint:gosec // git's object names, not a security use
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/braswelljr/gnux/core/fsys"
)

// The object types held in the top bits of a git mode.
const (
	modeSymlink = 0o120000
	modeGitlink = 0o160000
	modeTree    = 0o040000
)

// indexEntry is what the index records about one tracked path.
type indexEntry struct {
	mode      uint32
	size      uint32
	mtimeSec  uint32
	mtimeNsec uint32
	hash      string // raw object name
}

// sameTime reports whether t is the modification time the index recorded.
// An index written where nanoseconds are not kept compares seconds only.
func (e *indexEntry) sameTime(t time.Time) bool {
	if uint32(t.Unix()) != e.mtimeSec {
		return false
	}
	return e.mtimeNsec == 0 || uint32(t.Nanosecond()) == e.mtimeNsec
}

// readIndex loads the index (format versions 2, 3 and 4). A repository
// with no index yet has nothing tracked.
func (r *Repo) readIndex() error {
	r.index = map[string]*indexEntry{}
	r.conflicts = map[string]bool{}
	file := filepath.Join(r.gitDir, "index")
	info, err := fsys.Stat(file)
	if err != nil {
		return nil
	}
	r.indexMtime = info.ModTime()
	data, err := fsys.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	if len(data) < 12 || string(data[:4]) != "DIRC" {
		return fmt.Errorf("%s: %w", file, errCorrupt)
	}
	version := binary.BigEndian.Uint32(data[4:8])
	if version < 2 || version > 4 {
		return fmt.Errorf("%s: index version %d: %w", file, version, errCorrupt)
	}
	count := int(binary.BigEndian.Uint32(data[8:12]))

	pos := 12
	var prev []byte
	seen := make(map[string]bool, count)
	for range count {
		start := pos
		fixed := 40 + r.hashLen + 2
		if pos+fixed > len(data) {
			return fmt.Errorf("%s: %w", file, errCorrupt)
		}
		e := &indexEntry{
			mtimeSec:  binary.BigEndian.Uint32(data[pos+8:]),
			mtimeNsec: binary.BigEndian.Uint32(data[pos+12:]),
			mode:      binary.BigEndian.Uint32(data[pos+24:]),
			size:      binary.BigEndian.Uint32(data[pos+36:]),
			hash:      string(data[pos+40 : pos+40+r.hashLen]),
		}
		flags := binary.BigEndian.Uint16(data[pos+40+r.hashLen:])
		pos += fixed
		if version >= 3 && flags&0x4000 != 0 {
			pos += 2
		}
		stage := (flags >> 12) & 3

		var name []byte
		if version == 4 {
			strip, n := varint(data[pos:])
			if n == 0 || strip > len(prev) {
				return fmt.Errorf("%s: %w", file, errCorrupt)
			}
			pos += n
			suffix := trimNUL(data[pos:])
			name = append(slices.Clone(prev[:len(prev)-strip]), suffix...)
			pos += len(suffix) + 1
		} else {
			name = trimNUL(data[pos:])
			// Entries are padded with 1 to 8 NULs to a multiple of 8 bytes.
			pos = start + (pos-start+len(name)+8)&^7
		}
		prev = name
		if pos > len(data) {
			return fmt.Errorf("%s: %w", file, errCorrupt)
		}

		p := string(name)
		// A sparse index's directory entries stand for a whole tree that is
		// not checked out; nothing under them can be listed.
		if e.mode == modeTree {
			continue
		}
		if !seen[p] {
			seen[p] = true
			r.paths = append(r.paths, p)
		}
		if stage == 0 {
			r.index[p] = e
		} else {
			r.conflicts[p] = true
		}
	}
	slices.Sort(r.paths)
	return nil
}

// varint decodes the offset encoding git uses in index v4 and in pack
// files' delta offsets, returning the value and the bytes it took (0 on
// truncated input).
func varint(b []byte) (int, int) {
	if len(b) == 0 {
		return 0, 0
	}
	c := b[0]
	v := int(c & 0x7f)
	i := 1
	for c&0x80 != 0 {
		if i >= len(b) {
			return 0, 0
		}
		c = b[i]
		i++
		v = ((v + 1) << 7) | int(c&0x7f)
	}
	return v, i
}

// blobHash is the object name git gives a blob with this content.
func blobHash(hashLen int, content []byte) string {
	header := fmt.Appendf(nil, "blob %d\x00", len(content))
	if hashLen == 32 {
		h := sha256.New()
		h.Write(header)
		h.Write(content)
		return string(h.Sum(nil))
	}
	h := sha1.New() //nolint:gosec // git's object names, not a security use
	h.Write(header)
	h.Write(content)
	return string(h.Sum(nil))
}
