//go:build !linux && !darwin && !freebsd

package uname

// goarchMachine names Go's architectures the way uname -m does on Linux, for
// the platforms that do not report a machine name of their own.
var goarchMachine = map[string]string{
	"amd64":   "x86_64",
	"386":     "i686",
	"arm64":   "aarch64",
	"arm":     "armv7l",
	"riscv64": "riscv64",
	"ppc64le": "ppc64le",
	"ppc64":   "ppc64",
	"s390x":   "s390x",
	"loong64": "loongarch64",
	"wasm":    "wasm32",
}
