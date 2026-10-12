// Package fsstat reports file system statistics (statfs) and the mounted
// file systems, for statx -f and dfx. Each platform reports what it has;
// a count the platform does not keep is zero and marked unknown.
package fsstat

import (
	"errors"
	"strings"
)

// ErrUnsupported is returned where the platform cannot report file systems
// (the browser).
var ErrUnsupported = errors.New("operation not supported")

// FS describes one file system.
type FS struct {
	// Type is the system's number for the file system type (a magic number
	// on Linux), and TypeName its name ("ext2/ext3", "apfs", "NTFS").
	Type     uint64
	TypeName string
	// Remote reports a network file system, for dfx -l.
	Remote bool
	// IOSize is the preferred transfer size (stat %s); BlockSize the unit
	// Blocks, Bfree and Bavail count in (stat %S).
	IOSize, BlockSize uint64
	Blocks, Bfree     uint64
	// Bavail is the free blocks available to an unprivileged user; it can be
	// negative on file systems that reserve space.
	Bavail int64
	// Files and Ffree count inodes; FilesKnown is false where the file
	// system keeps no inode counts (Windows).
	Files      uint64
	Ffree      int64
	FilesKnown bool
	// NameMax is the longest file name; 0 when unknown.
	NameMax uint64
	// ID identifies the file system (f_fsid, or the volume serial number).
	ID uint64
}

// Mount is one entry of the mount table.
type Mount struct {
	Source string // the device or remote resource
	Target string // the mount point
	Type   string // the file system type name
	// Dummy marks pseudo file systems (proc, sysfs, ...) that dfx leaves out
	// unless -a.
	Dummy bool
	// Remote marks network file systems, which dfx -l leaves out.
	Remote bool
}

// dummyTypes are gnulib's ME_DUMMY list: the pseudo file systems GNU df
// hides without -a. (It also hides any file system with no blocks, which
// covers cgroup, bpf and the like.)
var dummyTypes = map[string]bool{
	"autofs": true, "proc": true, "subfs": true, "debugfs": true, "devpts": true,
	"fusectl": true, "fuse.portal": true, "mqueue": true, "rpc_pipefs": true,
	"sysfs": true, "devfs": true, "kernfs": true, "ignore": true,
}

// IsDummy reports whether a file system type is a pseudo one.
func IsDummy(typ string) bool { return dummyTypes[typ] }

// remoteTypes are network file systems by name, for platforms that report
// a name rather than a magic number.
var remoteTypes = map[string]bool{
	"nfs": true, "nfs4": true, "smbfs": true, "cifs": true, "afpfs": true,
	"webdav": true, "smb2": true, "afs": true, "9p": true, "ceph": true,
	"fuse.sshfs": true, "coda": true, "ncpfs": true, "smb3": true, "glusterfs": true,
	"fuse.glusterfs": true, "gfs2": true, "ocfs2": true,
	"lustre": true, //nolint:misspell // the Lustre file system
}

// isRemote reports a network file system from its type or its source: a
// "host:path" export or a "//server/share".
func isRemote(typ, source string) bool {
	if remoteTypes[typ] {
		return true
	}
	if i := strings.IndexByte(source, ':'); i > 1 && !strings.HasPrefix(source, "/") {
		return true
	}
	return strings.HasPrefix(source, "//")
}

// cString converts a NUL-terminated C char array.
func cString[T int8 | uint8](b []T) string {
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	out := make([]byte, n)
	for i := range n {
		out[i] = byte(b[i])
	}
	return string(out)
}
