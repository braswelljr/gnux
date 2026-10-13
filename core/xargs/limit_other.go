//go:build !unix

package xargs

// systemArgMax is the command-line size, in bytes, taken as the system's upper
// limit: Windows refuses to start a program whose command line is longer than
// 32767 characters. The same value serves js/wasm, where only gnux's own
// tools can run and nothing imposes a limit.
const systemArgMax = 32767

// ttyName is the file the answer to a -p prompt is read from: standard input
// carries the items, so it cannot also carry the answers. It is the Windows
// console input; where there is no console, opening it fails and -p reports
// that.
const ttyName = "CONIN$"
