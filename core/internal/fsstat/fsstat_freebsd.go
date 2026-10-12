//go:build freebsd

package fsstat

import "syscall"

func fromStatfs(st *syscall.Statfs_t) FS {
	name := cString(st.Fstypename[:])
	return FS{
		Type: uint64(st.Type), TypeName: name, Remote: remoteTypes[name],
		IOSize: st.Iosize, BlockSize: st.Bsize,
		Blocks: st.Blocks, Bfree: st.Bfree, Bavail: st.Bavail,
		Files: st.Files, Ffree: st.Ffree, FilesKnown: true,
		NameMax: uint64(st.Namemax),
		ID:      uint64(uint32(st.Fsid.Val[0]))<<32 | uint64(uint32(st.Fsid.Val[1])),
	}
}
