//go:build linux

package fsstat

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"
)

// Statfs reports the file system holding path.
func Statfs(path string) (FS, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return FS{}, err
	}
	typ := uint64(st.Type) //nolint:unconvert // int32 on some platforms
	t, ok := linuxTypes[typ]
	if !ok {
		t.name = fmt.Sprintf("UNKNOWN (0x%x)", typ)
	}
	fs := FS{
		Type: typ, TypeName: t.name, Remote: t.remote,
		IOSize: uint64(st.Bsize), BlockSize: uint64(st.Frsize), //nolint:unconvert
		Blocks: st.Blocks, Bfree: st.Bfree, Bavail: int64(st.Bavail), //nolint:unconvert
		Files: st.Files, Ffree: int64(st.Ffree), FilesKnown: true, //nolint:unconvert
		NameMax: uint64(st.Namelen), //nolint:unconvert
		ID:      uint64(uint32(st.Fsid.X__val[0]))<<32 | uint64(uint32(st.Fsid.X__val[1])),
	}
	if fs.BlockSize == 0 {
		fs.BlockSize = fs.IOSize
	}
	return fs, nil
}

// Mounts reads the mount table from /proc/self/mounts.
func Mounts() ([]Mount, error) {
	f, err := fsys.Open("/proc/self/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Mount
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		m := Mount{Source: unescape(fields[0]), Target: unescape(fields[1]), Type: fields[2]}
		m.Dummy = IsDummy(m.Type) || m.Source == "none"
		m.Remote = isRemote(m.Type, m.Source)
		out = append(out, m)
	}
	return out, sc.Err()
}

// unescape decodes the octal escapes (\040 for a space) of the mount table.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
