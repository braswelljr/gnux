# Platform code lives behind build tags, with a fallback that compiles everywhere

Anything that differs by platform goes in a build-tagged file, with a counterpart that compiles on every other target, including `js/wasm`. There is no `runtime.GOOS` branching outside tests and no cgo anywhere, so the binary stays static and cross-compiles from any machine.

## Why

gnux ships for Linux, macOS and Windows and runs in the browser as WebAssembly. A `syscall.Stat_t` reference in a shared file breaks the Windows and wasm builds; a `runtime.GOOS` switch compiles every branch on every platform, so it has the same problem and hides it until someone builds for the other target. cgo would tie each binary to a C toolchain and a libc, and cannot target wasm at all. Build tags make the compiler, not the runtime, choose the code, and a missing fallback fails the build where CI can see it.

## What it looks like

Prefer `unix` / `!unix` to listing operating systems. From [`core/fs_unix.go`](../../core/fs_unix.go):

```go
//go:build unix

// Writable reports whether the calling process may write to path. rm uses this
// to decide whether a file is "write-protected" and warrants a prompt.
func Writable(path string) bool {
	return syscall.Access(path, wOK) == nil
}
```

and its fallback, [`core/fs_other.go`](../../core/fs_other.go):

```go
//go:build !unix

// Writable reports whether path is writable. Off Unix (Windows, WebAssembly)
// there is no access(2); approximate with the owner write bit, which is how
// the read-only attribute surfaces.
```

The same pattern recurs where it is needed: `core/pipe_other.go` / `core/pipe_windows.go`, `core/touch/lchtimes_linux.go` / `lchtimes_other.go`, `core/mkdir/umask_unix.go` / `umask_other.go`, and `core/ls/stat_times_js.go` for WebAssembly. What a fallback cannot do is documented in the tool's `Permissions` note (`--one-file-system` is a no-op off Unix).

Do: `foo_unix.go` with `//go:build unix` and `foo_other.go` with `//go:build !unix`.

Do not: `if runtime.GOOS == "windows" { ... }`, or `import "C"`.

## How it is checked

- The [Build workflow](../../.github/workflows/build.yml) cross-compiles linux and darwin on amd64 and arm64 and windows on amd64, each with `CGO_ENABLED: "0"`.
- The [Test workflow](../../.github/workflows/test.yml) runs the suite on ubuntu, macOS and Windows, and has a "Build for WebAssembly" step (`GOOS=js GOARCH=wasm go build ./tools/wasm`).
- The [Web workflow](../../.github/workflows/web.yml) builds the wasm and runs the playground suite against it.
- That no `runtime.GOOS` branch appears outside tests is review; there is none today.

## Related

- NFR-1, NFR-2, NFR-3, NFR-4, NFR-5, NFR-6 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [The playground never touches the visitor's files](the-playground-never-touches-the-visitors-files.md)
- [Workflows](../workflows/README.md), in particular CI and release.
