//go:build windows

package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// On Windows the sandbox is an AppContainer, the isolation Windows gives
// browser renderers and store apps: a process in one can open a file only
// when the file's access list grants the container (or all of them, as the
// system's own directories do for reading). gnux grants its container the
// policy's directories, then runs the same command line again inside it and
// waits for it, passing its standard streams and its exit status through.
// The grants stay, as an entry for that container on each directory (which
// "icacls DIR /remove *SID" takes away), so that the next run with the same
// policy starts at once; a container of another policy gets none of them.

// marker is set in the environment of the gnux run inside the container, to
// the policy's identifier, so that it knows it is (and checks it is)
// confined.
const marker = "GNUX_SANDBOX"

const (
	// PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES, which puts a new process
	// in an AppContainer.
	procThreadAttributeSecurityCapabilities = 0x00020009
	// EXTENDED_STARTUPINFO_PRESENT says the startup information carries an
	// attribute list.
	extendedStartupInfoPresent = 0x00080000
	// HRESULT_FROM_WIN32(ERROR_ALREADY_EXISTS), from a profile made earlier.
	hresultAlreadyExists = 0x800700B7
)

// The rights the container is granted.
const (
	readRights = windows.FILE_GENERIC_READ | windows.FILE_GENERIC_EXECUTE
	// FILE_ALL_ACCESS: everything a file or directory allows.
	writeRights = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0x1FF
)

var (
	userenv                                    = windows.NewLazySystemDLL("userenv.dll")
	procCreateAppContainerProfile              = userenv.NewProc("CreateAppContainerProfile")
	procDeriveAppContainerSidFromAppContainerN = userenv.NewProc("DeriveAppContainerSidFromAppContainerName")
)

// securityCapabilities is SECURITY_CAPABILITIES: the container, and no
// capabilities (no network, no user libraries, no devices).
type securityCapabilities struct {
	AppContainerSid *windows.SID
	Capabilities    *windows.SIDAndAttributes
	CapabilityCount uint32
	Reserved        uint32
}

// enter runs the command line again in the policy's AppContainer and exits
// with its status; inside the container it returns at once (and Enter then
// checks the confinement).
func enter(p Policy) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("sandbox: %w", err)
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	// Inside the container its paths cannot be looked up any more (it cannot
	// read C:\Users), so the policy's identifier could not be worked out
	// again to compare: the marker says this is the run inside, and Enter's
	// check then proves the confinement (or refuses, should someone have set
	// the marker by hand outside it).
	if os.Getenv(marker) != "" {
		_ = os.Unsetenv(marker)
		return nil
	}
	id := policyID(p, exe)

	sid, err := containerSID("gnux.sandbox." + id)
	if err != nil {
		return err
	}
	defer windows.FreeSid(sid)
	for _, g := range []struct {
		paths   []string
		rights  uint32
		inherit uint32
	}{
		{p.Write, writeRights, windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT},
		{p.Read, readRights, windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT},
		{[]string{exe}, readRights, windows.NO_INHERITANCE},
	} {
		for _, path := range g.paths {
			if err := grant(path, sid, g.rights, g.inherit); err != nil {
				return err
			}
		}
	}
	code, err := runContained(sid, exe, id)
	if err != nil {
		return err
	}
	os.Exit(int(code))
	return nil // not reached
}

// policyID names a policy: the same directories always give the same
// container, and other directories another one.
func policyID(p Policy, exe string) string {
	write, read := append([]string(nil), p.Write...), append([]string(nil), p.Read...)
	sort.Strings(write)
	sort.Strings(read)
	key := strings.ToLower(strings.Join(write, "\x00") + "\x01" + strings.Join(read, "\x00") + "\x01" + exe)
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:8])
}

// containerSID makes the AppContainer profile called name, or finds the one
// made before, and returns its SID, which the caller frees.
func containerSID(name string) (*windows.SID, error) {
	if err := procCreateAppContainerProfile.Find(); err != nil {
		return nil, fmt.Errorf("sandbox: %w: this Windows has no AppContainers (Windows 8 or later)", ErrUnsupported)
	}
	n, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	var sid *windows.SID
	hr, _, _ := procCreateAppContainerProfile.Call(uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(n)), 0, 0, uintptr(unsafe.Pointer(&sid)))
	switch uint32(hr) {
	case 0:
		return sid, nil
	case hresultAlreadyExists:
		hr, _, _ = procDeriveAppContainerSidFromAppContainerN.Call(uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(&sid)))
		if hr == 0 {
			return sid, nil
		}
	}
	return nil, fmt.Errorf("sandbox: making the AppContainer: HRESULT %#x", uint32(hr))
}

// grant gives sid rights on path (and, with inherit, beneath it), unless an
// entry for it already does.
func grant(path string, sid *windows.SID, rights, inherit uint32) error {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("sandbox: %s: %w", path, err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("sandbox: %s: %w", path, err)
	}
	if granted(dacl, sid, rights) {
		return nil
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.ACCESS_MASK(rights),
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       inherit,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}}, dacl)
	if err != nil {
		return fmt.Errorf("sandbox: %s: %w", path, err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		return fmt.Errorf("sandbox: granting the sandbox %s: %w", path, err)
	}
	return nil
}

// granted reports whether dacl already allows sid every one of rights.
func granted(dacl *windows.ACL, sid *windows.SID, rights uint32) bool {
	if dacl == nil {
		return false
	}
	for i := range uint32(dacl.AceCount) {
		var ace *windows.ACCESS_ALLOWED_ACE
		if windows.GetAce(dacl, i, &ace) != nil || ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			continue
		}
		if uint32(ace.Mask)&rights == rights && (*windows.SID)(unsafe.Pointer(&ace.SidStart)).Equals(sid) {
			return true
		}
	}
	return false
}

// runContained runs exe with this process's arguments in the container sid,
// with its standard streams, in a job that ends it should this process end,
// and returns its exit status.
func runContained(sid *windows.SID, exe, id string) (uint32, error) {
	args := append([]string{exe}, os.Args[1:]...)
	cmdline, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(args))
	if err != nil {
		return 0, err
	}
	env, err := environmentBlock(append(os.Environ(), marker+"="+id))
	if err != nil {
		return 0, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	dir, err := windows.UTF16PtrFromString(cwd)
	if err != nil {
		return 0, err
	}

	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return 0, fmt.Errorf("sandbox: %w", err)
	}
	defer attrs.Delete()
	caps := securityCapabilities{AppContainerSid: sid}
	if err := attrs.Update(procThreadAttributeSecurityCapabilities, unsafe.Pointer(&caps), unsafe.Sizeof(caps)); err != nil {
		return 0, fmt.Errorf("sandbox: %w", err)
	}

	var si windows.StartupInfoEx
	si.Cb = uint32(unsafe.Sizeof(si))
	si.ProcThreadAttributeList = attrs.List()
	si.Flags = windows.STARTF_USESTDHANDLES
	for _, h := range []struct {
		which uint32
		into  *windows.Handle
	}{{windows.STD_INPUT_HANDLE, &si.StdInput}, {windows.STD_OUTPUT_HANDLE, &si.StdOutput}, {windows.STD_ERROR_HANDLE, &si.StdErr}} {
		if handle, err := windows.GetStdHandle(h.which); err == nil && handle != windows.InvalidHandle {
			_ = windows.SetHandleInformation(handle, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT)
			*h.into = handle
		}
	}

	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("sandbox: %w", err)
	}
	defer windows.CloseHandle(job)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return 0, fmt.Errorf("sandbox: %w", err)
	}

	var pi windows.ProcessInformation
	flags := uint32(extendedStartupInfoPresent | windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_SUSPENDED)
	if err := windows.CreateProcess(nil, cmdline, nil, nil, true, flags, &env[0], dir, &si.StartupInfo, &pi); err != nil {
		return 0, fmt.Errorf("sandbox: starting gnux in the AppContainer: %w", err)
	}
	defer windows.CloseHandle(pi.Process)
	defer windows.CloseHandle(pi.Thread)
	if err := windows.AssignProcessToJobObject(job, pi.Process); err != nil {
		_ = windows.TerminateProcess(pi.Process, 1)
		return 0, fmt.Errorf("sandbox: %w", err)
	}
	// Ctrl+C reaches both processes; the contained one decides what it means.
	signal.Ignore(os.Interrupt)
	if _, err := windows.ResumeThread(pi.Thread); err != nil {
		_ = windows.TerminateProcess(pi.Process, 1)
		return 0, fmt.Errorf("sandbox: %w", err)
	}
	if _, err := windows.WaitForSingleObject(pi.Process, windows.INFINITE); err != nil {
		return 0, fmt.Errorf("sandbox: %w", err)
	}
	var code uint32
	if err := windows.GetExitCodeProcess(pi.Process, &code); err != nil {
		return 0, fmt.Errorf("sandbox: %w", err)
	}
	return code, nil
}

// environmentBlock is env as CreateProcess takes it: NUL-separated UTF-16,
// ended by an empty entry.
func environmentBlock(env []string) ([]uint16, error) {
	var block []uint16
	for _, kv := range env {
		u, err := windows.UTF16FromString(kv)
		if err != nil {
			return nil, errors.New("sandbox: an environment variable holds a NUL")
		}
		block = append(block, u...)
	}
	return append(block, 0), nil
}
