//go:build !unix

package kill

import (
	"os"
	"syscall"
)

// signals is the conventional table (the Linux numbering). Off Unix a signal
// can only be named; Send delivers just KILL and TERM, by terminating the
// process.
var signals = []Signal{
	{1, "HUP", "Hangup"}, {2, "INT", "Interrupt"}, {3, "QUIT", "Quit"},
	{4, "ILL", "Illegal instruction"}, {5, "TRAP", "Trace/breakpoint trap"},
	{6, "ABRT", "Aborted"}, {7, "BUS", "Bus error"}, {8, "FPE", "Floating point exception"},
	{9, "KILL", "Killed"}, {10, "USR1", "User defined signal 1"},
	{11, "SEGV", "Segmentation fault"}, {12, "USR2", "User defined signal 2"},
	{13, "PIPE", "Broken pipe"}, {14, "ALRM", "Alarm clock"}, {15, "TERM", "Terminated"},
	{17, "CHLD", "Child exited"}, {18, "CONT", "Continued"}, {19, "STOP", "Stopped (signal)"},
	{20, "TSTP", "Stopped"}, {21, "TTIN", "Stopped (tty input)"},
	{22, "TTOU", "Stopped (tty output)"}, {23, "URG", "Urgent I/O condition"},
	{24, "XCPU", "CPU time limit exceeded"}, {25, "XFSZ", "File size limit exceeded"},
	{26, "VTALRM", "Virtual timer expired"}, {27, "PROF", "Profiling timer expired"},
	{28, "WINCH", "Window changed"}, {29, "IO", "I/O possible"}, {31, "SYS", "Bad system call"},
}

func send(pid, sig int) error {
	if pid <= 0 {
		return ErrUnsupported
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return syscall.ESRCH
	}
	defer p.Release()
	return signalProcess(p, sig, false)
}

func signalProcess(p *os.Process, sig int, _ bool) error {
	switch sig {
	case 0:
		return nil
	case 9, 15:
		return p.Kill()
	}
	return ErrUnsupported
}

func exitStatus(state *os.ProcessState) int {
	return state.ExitCode()
}
