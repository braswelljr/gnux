//go:build windows

package utmp

import (
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// DefaultFile is empty: Windows keeps sessions, not a utmp file.
const DefaultFile = ""

var (
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	getTickCount64   = kernel32.NewProc("GetTickCount64")
	wtsapi32         = windows.NewLazySystemDLL("wtsapi32.dll")
	wtsQuerySession  = wtsapi32.NewProc("WTSQuerySessionInformationW")
	wtsUserName      = uintptr(5)
	wtsSessionInfo   = uintptr(24)
	wtsActive        = uint32(0)
	wtsDisconnected  = uint32(4)
	wtsinfoLogonTime = 200 // offset of LogonTime in WTSINFOW
)

// Boot returns the time Windows started: now minus GetTickCount64.
func Boot() (time.Time, error) {
	if err := getTickCount64.Find(); err != nil {
		return time.Time{}, err
	}
	ms, _, _ := getTickCount64.Call()
	return time.Now().Add(-time.Duration(ms) * time.Millisecond).Truncate(time.Second), nil
}

// LoadAvg is not kept by Windows.
func LoadAvg() ([]float64, error) { return nil, ErrUnsupported }

// ReadFile is not available: Windows has no utmp files.
func ReadFile(string) ([]Entry, error) { return nil, ErrUnsupported }

// Current lists the signed-in Windows sessions (console and Remote Desktop)
// as user records, with a boot record first.
func Current() ([]Entry, error) {
	var sessions *windows.WTS_SESSION_INFO
	var count uint32
	if err := windows.WTSEnumerateSessions(0, 0, 1, &sessions, &count); err != nil {
		return withBoot(nil), nil
	}
	defer windows.WTSFreeMemory(uintptr(unsafe.Pointer(sessions)))
	var out []Entry
	for _, s := range unsafe.Slice(sessions, count) {
		if s.State != wtsActive && s.State != wtsDisconnected {
			continue
		}
		user := queryString(s.SessionID, wtsUserName)
		if user == "" {
			continue
		}
		e := Entry{Type: UserProcess, User: user, Line: windows.UTF16PtrToString(s.WindowStationName)}
		if e.Line == "" {
			e.Line = "session" + strconv.FormatUint(uint64(s.SessionID), 10)
		}
		e.Time = logonTime(s.SessionID)
		out = append(out, e)
	}
	return withBoot(out), nil
}

func query(session uint32, class uintptr) ([]byte, bool) {
	var buf *byte
	var n uint32
	r, _, _ := wtsQuerySession.Call(0, uintptr(session), class, uintptr(unsafe.Pointer(&buf)), uintptr(unsafe.Pointer(&n)))
	if r == 0 || buf == nil {
		return nil, false
	}
	defer windows.WTSFreeMemory(uintptr(unsafe.Pointer(buf)))
	return append([]byte(nil), unsafe.Slice(buf, n)...), true
}

func queryString(session uint32, class uintptr) string {
	b, ok := query(session, class)
	if !ok || len(b) < 2 {
		return ""
	}
	u := unsafe.Slice((*uint16)(unsafe.Pointer(&b[0])), len(b)/2)
	return windows.UTF16ToString(u)
}

// logonTime reads LogonTime from the session's WTSINFOW, or zero.
func logonTime(session uint32) time.Time {
	b, ok := query(session, wtsSessionInfo)
	if !ok || len(b) < wtsinfoLogonTime+8 {
		return time.Time{}
	}
	ft := *(*windows.Filetime)(unsafe.Pointer(&b[wtsinfoLogonTime]))
	if ft.Nanoseconds() <= 0 {
		return time.Time{}
	}
	return time.Unix(0, ft.Nanoseconds())
}
