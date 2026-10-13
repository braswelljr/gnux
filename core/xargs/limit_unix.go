//go:build unix

package xargs

// systemArgMax is the command-line size, in bytes, taken as the system's upper
// limit. The real limit is not available without cgo, so this is the smallest
// ARG_MAX among the Unix systems gnux is released for (FreeBSD's 512 KiB;
// macOS allows 1 MiB and Linux 2 MiB), which keeps a command line built to it
// runnable on all of them.
const systemArgMax = 512 << 10

// ttyName is the file the answer to a -p prompt is read from: standard input
// carries the items, so it cannot also carry the answers.
const ttyName = "/dev/tty"
