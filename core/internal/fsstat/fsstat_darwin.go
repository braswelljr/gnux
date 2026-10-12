//go:build darwin

package fsstat

import "syscall"

func fromStatfs(st *syscall.Statfs_t) FS {
	name := cString(st.Fstypename[:])
	return FS{
		Type: uint64(st.Type), TypeName: name, Remote: remoteTypes[name],
		IOSize: uint64(st.Iosize), BlockSize: uint64(st.Bsize),
		Blocks: st.Blocks, Bfree: st.Bfree, Bavail: int64(st.Bavail),
		Files: st.Files, Ffree: int64(st.Ffree), FilesKnown: true,
		// darwin's statfs has no name limit; APFS and HFS+ allow 255 bytes.
		NameMax: 255,
		ID:      uint64(uint32(st.Fsid.Val[0]))<<32 | uint64(uint32(st.Fsid.Val[1])),
	}
}
