// Package common holds app-wide constants shared across packages.
package common

import "runtime/debug"

// AppName is the program name used in diagnostics and prompts.
const AppName = "gnux"

// Version is the build version. It is overridden at link time via
//
//	-ldflags "-X github.com/braswelljr/gnux/internal/common.Version=<tag>"
//
// When it is not, it falls back to the module version recorded in the build
// info, so `go install github.com/braswelljr/gnux@v1.2.3` reports v1.2.3 and
// a build from a checkout reports its VCS-derived pseudo-version.
var Version = "dev"

func init() {
	if Version != "dev" {
		return
	}
	Version = buildInfoVersion(debug.ReadBuildInfo())
}

// buildInfoVersion returns the main module's version from build info, or
// "dev" when there is none worth reporting (tests, `go run`, a GOPATH build).
func buildInfoVersion(info *debug.BuildInfo, ok bool) string {
	if !ok || info == nil {
		return "dev"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	return "dev"
}
