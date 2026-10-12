//go:build windows

package nice

import "golang.org/x/sys/windows"

// getNiceness reports the niceness the calling process's priority class
// stands for.
func getNiceness() (int, error) {
	class, err := windows.GetPriorityClass(windows.CurrentProcess())
	if err != nil {
		return 0, err
	}
	return classNiceness(class), nil
}

// setNiceness gives process pid the priority class nearest to niceness.
func setNiceness(pid, niceness int) error {
	h, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(pid))
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	return windows.SetPriorityClass(h, priorityClass(niceness))
}
