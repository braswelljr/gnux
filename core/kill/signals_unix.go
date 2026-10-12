//go:build unix

package kill

import (
	"os"
	"slices"
	"syscall"
	"unicode"
)

// signals is the table of signals every Unix platform gnux supports defines.
// Their numbers differ between systems (USR1 is 10 on Linux, 30 on macOS), so
// they come from package syscall.
var signals = func() []Signal {
	named := []struct {
		name string
		sig  syscall.Signal
	}{
		{"HUP", syscall.SIGHUP}, {"INT", syscall.SIGINT}, {"QUIT", syscall.SIGQUIT},
		{"ILL", syscall.SIGILL}, {"TRAP", syscall.SIGTRAP}, {"ABRT", syscall.SIGABRT},
		{"BUS", syscall.SIGBUS}, {"FPE", syscall.SIGFPE}, {"KILL", syscall.SIGKILL},
		{"USR1", syscall.SIGUSR1}, {"SEGV", syscall.SIGSEGV}, {"USR2", syscall.SIGUSR2},
		{"PIPE", syscall.SIGPIPE}, {"ALRM", syscall.SIGALRM}, {"TERM", syscall.SIGTERM},
		{"CHLD", syscall.SIGCHLD}, {"CONT", syscall.SIGCONT}, {"STOP", syscall.SIGSTOP},
		{"TSTP", syscall.SIGTSTP}, {"TTIN", syscall.SIGTTIN}, {"TTOU", syscall.SIGTTOU},
		{"URG", syscall.SIGURG}, {"XCPU", syscall.SIGXCPU}, {"XFSZ", syscall.SIGXFSZ},
		{"VTALRM", syscall.SIGVTALRM}, {"PROF", syscall.SIGPROF}, {"WINCH", syscall.SIGWINCH},
		{"IO", syscall.SIGIO}, {"SYS", syscall.SIGSYS},
	}
	out := make([]Signal, 0, len(named))
	for _, n := range named {
		out = append(out, Signal{Number: int(n.sig), Name: n.name, Description: describe(n.sig)})
	}
	slices.SortStableFunc(out, func(a, b Signal) int { return a.Number - b.Number })
	return out
}()

// describe returns the signal's description in strsignal(3)'s capitalization
// ("Terminated", "User defined signal 1").
func describe(sig syscall.Signal) string {
	r := []rune(sig.String())
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}

func send(pid, sig int) error {
	return syscall.Kill(pid, syscall.Signal(sig))
}

func signalProcess(p *os.Process, sig int, group bool) error {
	if group {
		return syscall.Kill(-p.Pid, syscall.Signal(sig))
	}
	return p.Signal(syscall.Signal(sig))
}

func exitStatus(state *os.ProcessState) int {
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return state.ExitCode()
}
