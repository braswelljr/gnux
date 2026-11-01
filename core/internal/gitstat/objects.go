package gitstat

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"
)

// treeEntry is one path of HEAD's tree.
type treeEntry struct {
	mode uint32
	hash string
}

// object is a decoded git object.
type object struct {
	kind string // "commit", "tree", "blob", "tag"
	data []byte
}

// objectStore reads objects from a repository's objects directory: loose
// objects first, then each pack file.
type objectStore struct {
	dir     string
	hashLen int
	packs   []*pack
	loaded  bool
	cache   map[string]object
}

// pack is one pack file with its index.
type pack struct {
	idx     []byte
	version int // of the .idx file: 1 or 2
	count   int
	file    *os.File
	size    int64
	byPos   map[int64]object
}

func (s *objectStore) close() {
	for _, p := range s.packs {
		_ = p.file.Close()
	}
	s.packs = nil
}

// read returns the object named by hash (raw bytes).
func (s *objectStore) read(hash string) (object, error) {
	if o, ok := s.cache[hash]; ok {
		return o, nil
	}
	o, err := s.readLoose(hash)
	if err != nil {
		o, err = s.readPacked(hash)
	}
	if err == nil && o.kind != "blob" {
		s.cache[hash] = o
	}
	return o, err
}

func (s *objectStore) readLoose(hash string) (object, error) {
	name := hex.EncodeToString([]byte(hash))
	data, err := fsys.ReadFile(filepath.Join(s.dir, name[:2], name[2:]))
	if err != nil {
		return object{}, err
	}
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return object{}, err
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return object{}, err
	}
	header, body, ok := bytes.Cut(raw, []byte{0})
	kind, _, _ := strings.Cut(string(header), " ")
	if !ok {
		return object{}, errCorrupt
	}
	return object{kind: kind, data: body}, nil
}

func (s *objectStore) loadPacks() {
	if s.loaded {
		return
	}
	s.loaded = true
	dir := filepath.Join(s.dir, "pack")
	entries, _ := fsys.ReadDir(dir)
	for _, de := range entries {
		base, ok := strings.CutSuffix(de.Name(), ".idx")
		if !ok {
			continue
		}
		idx, err := fsys.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			continue
		}
		f, err := fsys.Open(filepath.Join(dir, base+".pack"))
		if err != nil {
			continue
		}
		info, err := f.Stat()
		if err != nil {
			_ = f.Close()
			continue
		}
		p := &pack{idx: idx, file: f, size: info.Size(), byPos: map[int64]object{}}
		if p.parse(s.hashLen) {
			s.packs = append(s.packs, p)
		} else {
			_ = f.Close()
		}
	}
}

// parse checks the index's layout and reads its object count.
func (p *pack) parse(hashLen int) bool {
	p.version = 1
	fanout := 0
	if len(p.idx) >= 8 && string(p.idx[:4]) == "\xfftOc" {
		if binary.BigEndian.Uint32(p.idx[4:8]) != 2 {
			return false
		}
		p.version, fanout = 2, 8
	}
	if len(p.idx) < fanout+1024 {
		return false
	}
	p.count = int(binary.BigEndian.Uint32(p.idx[fanout+255*4:]))
	// v1: an offset and a name per object; v2: a name, a CRC and an offset.
	need := fanout + 1024 + p.count*(hashLen+4)
	if p.version == 2 {
		need += p.count * 4
	}
	return len(p.idx) >= need
}

// offset looks hash up in the pack's index.
func (p *pack) offset(hash string, hashLen int) (int64, bool) {
	fanout := 0
	if p.version == 2 {
		fanout = 8
	}
	first := hash[0]
	lo := 0
	if first > 0 {
		lo = int(binary.BigEndian.Uint32(p.idx[fanout+int(first-1)*4:]))
	}
	hi := int(binary.BigEndian.Uint32(p.idx[fanout+int(first)*4:]))
	names := fanout + 1024
	stride, skip := hashLen, 0
	if p.version == 1 {
		stride, skip = hashLen+4, 4
	}
	for lo < hi {
		mid := (lo + hi) / 2
		at := names + mid*stride + skip
		switch c := strings.Compare(string(p.idx[at:at+hashLen]), hash); {
		case c == 0:
			return p.offsetAt(mid, hashLen), true
		case c < 0:
			lo = mid + 1
		default:
			hi = mid
		}
	}
	return 0, false
}

func (p *pack) offsetAt(i, hashLen int) int64 {
	fanout := 1024
	if p.version == 1 {
		return int64(binary.BigEndian.Uint32(p.idx[fanout+i*(hashLen+4):]))
	}
	offsets := 8 + fanout + p.count*hashLen + p.count*4
	off := binary.BigEndian.Uint32(p.idx[offsets+i*4:])
	if off&0x80000000 == 0 {
		return int64(off)
	}
	large := offsets + p.count*4 + int(off&0x7fffffff)*8
	if large+8 > len(p.idx) {
		return -1
	}
	return int64(binary.BigEndian.Uint64(p.idx[large:]))
}

func (s *objectStore) readPacked(hash string) (object, error) {
	s.loadPacks()
	for _, p := range s.packs {
		if off, ok := p.offset(hash, s.hashLen); ok && off >= 0 {
			return s.readAt(p, off, 0)
		}
	}
	return object{}, fmt.Errorf("object %x: %w", hash, os.ErrNotExist)
}

var packKinds = [...]string{1: "commit", 2: "tree", 3: "blob", 4: "tag"}

// readAt decodes the object at off in p, resolving deltas.
func (s *objectStore) readAt(p *pack, off int64, depth int) (object, error) {
	if o, ok := p.byPos[off]; ok {
		return o, nil
	}
	if depth > 64 || off < 0 || off >= p.size {
		return object{}, errCorrupt
	}
	br := bufio.NewReader(io.NewSectionReader(p.file, off, p.size-off))
	c, err := br.ReadByte()
	if err != nil {
		return object{}, err
	}
	kind := int(c>>4) & 7
	for c&0x80 != 0 {
		if c, err = br.ReadByte(); err != nil {
			return object{}, err
		}
	}

	var base object
	switch kind {
	case 6: // offset delta
		var buf [16]byte
		n := 0
		for n < len(buf) {
			if buf[n], err = br.ReadByte(); err != nil {
				return object{}, err
			}
			n++
			if buf[n-1]&0x80 == 0 {
				break
			}
		}
		back, used := varint(buf[:n])
		if used == 0 {
			return object{}, errCorrupt
		}
		base, err = s.readAt(p, off-int64(back), depth+1)
	case 7: // reference delta
		ref := make([]byte, s.hashLen)
		if _, err = io.ReadFull(br, ref); err != nil {
			return object{}, err
		}
		base, err = s.read(string(ref))
	default:
		if kind >= len(packKinds) || packKinds[kind] == "" {
			return object{}, errCorrupt
		}
	}
	if err != nil {
		return object{}, err
	}

	zr, err := zlib.NewReader(br)
	if err != nil {
		return object{}, err
	}
	data, err := io.ReadAll(zr)
	if err != nil {
		return object{}, err
	}
	o := object{kind: base.kind, data: data}
	if kind == 6 || kind == 7 {
		if o.data, err = applyDelta(base.data, data); err != nil {
			return object{}, err
		}
	} else {
		o.kind = packKinds[kind]
	}
	if o.kind != "blob" {
		p.byPos[off] = o
	}
	return o, nil
}

// deltaSize reads the little-endian size at the start of a delta.
func deltaSize(d []byte) (int, []byte) {
	size, shift := 0, 0
	for len(d) > 0 {
		c := d[0]
		d = d[1:]
		size |= int(c&0x7f) << shift
		shift += 7
		if c&0x80 == 0 {
			break
		}
	}
	return size, d
}

// applyDelta rebuilds an object from its base and a delta.
func applyDelta(base, delta []byte) ([]byte, error) {
	srcSize, d := deltaSize(delta)
	dstSize, d := deltaSize(d)
	if srcSize != len(base) {
		return nil, errCorrupt
	}
	out := make([]byte, 0, dstSize)
	for len(d) > 0 {
		op := d[0]
		d = d[1:]
		if op&0x80 == 0 {
			n := int(op)
			if n == 0 || n > len(d) {
				return nil, errCorrupt
			}
			out = append(out, d[:n]...)
			d = d[n:]
			continue
		}
		var off, size int
		for i := range 4 {
			if op&(1<<i) != 0 {
				if len(d) == 0 {
					return nil, errCorrupt
				}
				off |= int(d[0]) << (8 * i)
				d = d[1:]
			}
		}
		for i := range 3 {
			if op&(0x10<<i) != 0 {
				if len(d) == 0 {
					return nil, errCorrupt
				}
				size |= int(d[0]) << (8 * i)
				d = d[1:]
			}
		}
		if size == 0 {
			size = 0x10000
		}
		if off+size > len(base) {
			return nil, errCorrupt
		}
		out = append(out, base[off:off+size]...)
	}
	if len(out) != dstSize {
		return nil, errCorrupt
	}
	return out, nil
}

// readHead loads HEAD's tree, once. With no commit yet the tree is empty;
// when HEAD cannot be read (a reftable repository, a missing object) the
// head map stays nil and every staged status is clean.
func (r *Repo) readHead() {
	if r.headRead {
		return
	}
	r.headRead = true
	if r.reftable {
		return
	}
	commit, ok := r.resolveHead()
	if !ok {
		return
	}
	head := map[string]treeEntry{}
	if commit != "" {
		o, err := r.objects.read(commit)
		if err != nil || o.kind != "commit" {
			return
		}
		line, _, _ := strings.Cut(string(o.data), "\n")
		tree, ok := strings.CutPrefix(line, "tree ")
		raw, err := hex.DecodeString(tree)
		if !ok || err != nil || r.walkTree(string(raw), "", head) != nil {
			return
		}
	}
	r.head = head
	for p := range head {
		if _, inIndex := r.index[p]; !inIndex && !r.conflicts[p] {
			r.headOnly = append(r.headOnly, p)
		}
	}
	slices.Sort(r.headOnly)
}

// resolveHead returns HEAD's commit (raw), "" for a branch with no commit
// yet, and false when HEAD cannot be read.
func (r *Repo) resolveHead() (string, bool) {
	data, err := fsys.ReadFile(filepath.Join(r.gitDir, "HEAD"))
	if err != nil {
		return "", false
	}
	head := strings.TrimSpace(string(data))
	for range 8 { // a symbolic ref may name another one
		ref, symbolic := strings.CutPrefix(head, "ref:")
		if !symbolic {
			raw, err := hex.DecodeString(head)
			return string(raw), err == nil && len(raw) == r.hashLen
		}
		ref = strings.TrimSpace(ref)
		next, ok := r.readRef(ref)
		if !ok {
			return "", true // unborn branch
		}
		head = next
	}
	return "", false
}

// readRef reads a reference from its loose file or from packed-refs.
func (r *Repo) readRef(name string) (string, bool) {
	for _, dir := range []string{r.gitDir, r.commonDir} {
		if data, err := fsys.ReadFile(filepath.Join(dir, filepath.FromSlash(name))); err == nil {
			return strings.TrimSpace(string(data)), true
		}
	}
	data, err := fsys.ReadFile(filepath.Join(r.commonDir, "packed-refs"))
	if err != nil {
		return "", false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if hash, ref, ok := strings.Cut(strings.TrimSpace(line), " "); ok && ref == name {
			return hash, true
		}
	}
	return "", false
}

// walkTree flattens a tree object into out, paths joined under prefix.
func (r *Repo) walkTree(hash, prefix string, out map[string]treeEntry) error {
	o, err := r.objects.read(hash)
	if err != nil {
		return err
	}
	if o.kind != "tree" {
		return errCorrupt
	}
	d := o.data
	for len(d) > 0 {
		sp := bytes.IndexByte(d, ' ')
		nul := bytes.IndexByte(d, 0)
		if sp < 0 || nul < sp || nul+1+r.hashLen > len(d) {
			return errCorrupt
		}
		mode, err := strconv.ParseUint(string(d[:sp]), 8, 32)
		if err != nil {
			return errCorrupt
		}
		name := path.Join(prefix, string(d[sp+1:nul]))
		child := string(d[nul+1 : nul+1+r.hashLen])
		d = d[nul+1+r.hashLen:]
		if mode == modeTree {
			if err := r.walkTree(child, name, out); err != nil {
				return err
			}
			continue
		}
		out[name] = treeEntry{mode: uint32(mode), hash: child}
	}
	return nil
}
