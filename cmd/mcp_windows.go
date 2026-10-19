//go:build windows

package cmd

import (
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// stopGroup makes stopping a call end the programs it started too: Windows
// has no process groups to signal, so the call's descendants are found and
// ended, the deepest first, then the call itself.
func stopGroup(c *exec.Cmd) {
	c.Cancel = func() error {
		killTree(uint32(c.Process.Pid))
		return nil
	}
}

// killTree ends the process pid and every process descended from it.
func killTree(pid uint32) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err == nil {
		children := map[uint32][]uint32{}
		var e windows.ProcessEntry32
		e.Size = uint32(unsafe.Sizeof(e))
		for ok := windows.Process32First(snap, &e) == nil; ok; ok = windows.Process32Next(snap, &e) == nil {
			children[e.ParentProcessID] = append(children[e.ParentProcessID], e.ProcessID)
		}
		_ = windows.CloseHandle(snap)
		var walk func(uint32)
		walk = func(p uint32) {
			for _, child := range children[p] {
				if child != p {
					walk(child)
				}
			}
			terminate(p)
		}
		for _, child := range children[pid] {
			walk(child)
		}
	}
	terminate(pid)
}

func terminate(pid uint32) {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return
	}
	_ = windows.TerminateProcess(h, 1)
	_ = windows.CloseHandle(h)
}
