//go:build linux

package sandbox

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// systemRead are what stays readable (and executable) in a sandbox: the
// programs, libraries and configuration a tool or a program it starts needs.
var systemRead = []string{
	"/usr", "/bin", "/sbin", "/lib", "/lib32", "/lib64", "/libx32",
	"/etc", "/opt", "/proc", "/nix/store",
	"/dev/zero", "/dev/random", "/dev/urandom",
}

// systemWrite are the devices that stay writable.
var systemWrite = []string{"/dev/null"}

// Landlock's rights, as the kernel numbers them.
const (
	fileRights = unix.LANDLOCK_ACCESS_FS_EXECUTE | unix.LANDLOCK_ACCESS_FS_WRITE_FILE |
		unix.LANDLOCK_ACCESS_FS_READ_FILE | unix.LANDLOCK_ACCESS_FS_TRUNCATE |
		unix.LANDLOCK_ACCESS_FS_IOCTL_DEV
	readRights = unix.LANDLOCK_ACCESS_FS_EXECUTE | unix.LANDLOCK_ACCESS_FS_READ_FILE |
		unix.LANDLOCK_ACCESS_FS_READ_DIR
)

// rulesetAttr is struct landlock_ruleset_attr as Landlock's first version
// defines it. gnux keeps its own copy rather than x/sys's, whose fields have
// been renamed and added to as Landlock grew: the kernel takes a structure
// of any size it knows (the size goes with it), and this one every Landlock
// kernel knows.
type rulesetAttr struct {
	handledAccessFS uint64
}

// pathBeneathAttr is struct landlock_path_beneath_attr: the kernel reads
// its first 12 bytes (it is packed), which Go lays out the same.
type pathBeneathAttr struct {
	allowedAccess uint64
	parentFd      int32
}

// handledRights are the rights a kernel of Landlock ABI version abi
// understands: all of them are restricted, so that what is not granted is
// denied.
func handledRights(abi int) uint64 {
	rights := uint64(0x1fff) // ABI 1: execute through make-symlink
	if abi >= 2 {
		rights |= unix.LANDLOCK_ACCESS_FS_REFER
	}
	if abi >= 3 {
		rights |= unix.LANDLOCK_ACCESS_FS_TRUNCATE
	}
	if abi >= 5 {
		rights |= unix.LANDLOCK_ACCESS_FS_IOCTL_DEV
	}
	return rights
}

// enter restricts every thread of the process with a Landlock ruleset.
func enter(p Policy) error {
	abi, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if errno != 0 || int(abi) < 1 {
		return fmt.Errorf("sandbox: %w: this kernel has no Landlock (Linux 5.13 or later, with Landlock enabled)", ErrUnsupported)
	}
	handled := handledRights(int(abi))

	attr := rulesetAttr{handledAccessFS: handled}
	fd, _, errno := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if errno != 0 {
		return fmt.Errorf("sandbox: creating the Landlock ruleset: %w", errno)
	}
	ruleset := int(fd)
	defer unix.Close(ruleset)

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("sandbox: %w", err)
	}
	add := func(paths []string, rights uint64) error {
		for _, path := range paths {
			if err := addRule(ruleset, path, rights&handled); err != nil {
				return err
			}
		}
		return nil
	}
	for _, step := range []struct {
		paths  []string
		rights uint64
	}{
		{p.Write, handled},
		{p.Read, readRights},
		{existing(systemRead), readRights},
		{existing(systemWrite), fileRights},
		{existing([]string{exe}), readRights},
	} {
		if err := add(step.paths, step.rights); err != nil {
			return err
		}
	}

	// Landlock binds each thread, so the restriction is applied to all of
	// Go's threads at once; that needs a build without cgo.
	if _, _, errno := syscall.AllThreadsSyscall6(unix.SYS_PRCTL, unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0, 0); errno != 0 {
		return restrictError(errno)
	}
	if _, _, errno := syscall.AllThreadsSyscall(unix.SYS_LANDLOCK_RESTRICT_SELF, uintptr(ruleset), 0, 0); errno != 0 {
		return restrictError(errno)
	}
	return nil
}

func restrictError(errno syscall.Errno) error {
	if errors.Is(errno, syscall.ENOTSUP) {
		return fmt.Errorf("sandbox: %w: this gnux was built with cgo, which keeps it from restricting all of its threads; build it with CGO_ENABLED=0", ErrUnsupported)
	}
	return fmt.Errorf("sandbox: entering the Landlock sandbox: %w", errno)
}

// addRule grants rights beneath path (or on it, for a file, which takes only
// the rights a file can have).
func addRule(ruleset int, path string, rights uint64) error {
	fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("sandbox: %s: %w", path, err)
	}
	defer unix.Close(fd)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return fmt.Errorf("sandbox: %s: %w", path, err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR {
		rights &= fileRights
	}
	rule := pathBeneathAttr{allowedAccess: rights, parentFd: int32(fd)}
	if _, _, errno := unix.Syscall6(unix.SYS_LANDLOCK_ADD_RULE, uintptr(ruleset), unix.LANDLOCK_RULE_PATH_BENEATH, uintptr(unsafe.Pointer(&rule)), 0, 0, 0); errno != 0 {
		return fmt.Errorf("sandbox: allowing %s: %w", path, errno)
	}
	return nil
}
