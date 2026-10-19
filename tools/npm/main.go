// Command npm assembles the npm packages of a gnux release from its
// archives, for the release workflow to publish.
//
// Usage:
//
//	go run ./tools/npm -version vX.Y.Z -from DIR -out DIR
//
// -from holds the release archives (gnux-<tag>-<os>-<arch>.tar.gz, or .zip on
// Windows), or bare binaries named gnux-<os>-<arch> (gnux-windows-amd64.exe),
// as `make npm/pack` builds them. -out receives one directory per package,
// ready for `npm publish`:
//
//   - gnux: what a project installs. Plain JavaScript: a launcher for gnux and
//     one for every tool under its own name (rmx, cpx, ...), never under the
//     GNU name, so that rm in a package script is still the system's rm.
//   - gnux-<platform>-<arch>: one per release platform, holding that
//     platform's binary and nothing else. The gnux package lists them all as
//     optional dependencies, each limited to its os and cpu, so npm installs
//     only the one the machine needs, without a download script.
//
// Every package gets the same version, the tag without its v. A platform
// with no archive in -from is an error, so a release never publishes a
// launcher that points at a package that does not exist.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/braswelljr/gnux/cmd"
)

// platform is one release target, named both ways.
type platform struct {
	goos, goarch string // as Go and the archives name it
	os, cpu      string // as Node's process.platform and process.arch name it
}

// platforms are the release targets of .goreleaser.yaml.
var platforms = []platform{
	{"darwin", "amd64", "darwin", "x64"},
	{"darwin", "arm64", "darwin", "arm64"},
	{"freebsd", "amd64", "freebsd", "x64"},
	{"freebsd", "arm64", "freebsd", "arm64"},
	{"linux", "amd64", "linux", "x64"},
	{"linux", "arm64", "linux", "arm64"},
	{"linux", "riscv64", "linux", "riscv64"},
	{"windows", "amd64", "win32", "x64"},
	{"windows", "arm64", "win32", "arm64"},
}

// pkgName is the npm name of a platform's package.
func (p platform) pkgName() string { return "gnux-" + p.os + "-" + p.cpu }

// exe is the binary's file name on the platform.
func (p platform) exe() string {
	if p.goos == "windows" {
		return "gnux.exe"
	}
	return "gnux"
}

func main() {
	version := flag.String("version", "", "release version, with or without the leading v")
	from := flag.String("from", "", "directory holding the release archives or binaries")
	out := flag.String("out", "", "directory to write the packages into")
	flag.Parse()
	if *version == "" || *from == "" || *out == "" || flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "usage: npm -version vX.Y.Z -from DIR -out DIR")
		os.Exit(2)
	}
	var tools []string
	for _, s := range cmd.Tools() {
		tools = append(tools, s.Name())
	}
	if err := Build(strings.TrimPrefix(*version, "v"), *from, *out, tools); err != nil {
		fmt.Fprintf(os.Stderr, "npm: %v\n", err)
		os.Exit(1)
	}
}

// Build writes every package for version into out, taking the binaries from
// from and naming a launcher after each of tools.
func Build(version, from, out string, tools []string) error {
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	for _, p := range platforms {
		if err := buildPlatform(p, version, from, filepath.Join(out, p.pkgName())); err != nil {
			return fmt.Errorf("%s: %w", p.pkgName(), err)
		}
	}
	return buildMain(version, filepath.Join(out, "gnux"), tools)
}

// buildPlatform writes the package that carries one platform's binary.
func buildPlatform(p platform, version, from, dir string) error {
	binary, err := findBinary(p, from)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", p.exe()), binary, 0o755); err != nil {
		return err
	}
	manifest := manifest{
		Name:        p.pkgName(),
		Version:     version,
		Description: fmt.Sprintf("The gnux binary for %s on %s; installed by the gnux package.", p.goos, p.goarch),
		License:     "MIT",
		Repository:  repository,
		Homepage:    homepage,
		OS:          []string{p.os},
		CPU:         []string{p.cpu},
		Files:       []string{"bin"},
		// Yarn Plug'n'Play must leave the binary on disk to run it.
		PreferUnplugged: true,
	}
	readme := fmt.Sprintf("# %s\n\nThe gnux binary for %s on %s. Install [gnux](https://www.npmjs.com/package/gnux) instead: it depends on this package and runs it.\n", p.pkgName(), p.goos, p.goarch)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "package.json"), manifest)
}

// manifest is a package.json, its fields in the order npm writes them.
type manifest struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Description          string            `json:"description"`
	Keywords             []string          `json:"keywords,omitempty"`
	License              string            `json:"license"`
	Repository           string            `json:"repository"`
	Homepage             string            `json:"homepage"`
	Bugs                 string            `json:"bugs,omitempty"`
	OS                   []string          `json:"os,omitempty"`
	CPU                  []string          `json:"cpu,omitempty"`
	Main                 string            `json:"main,omitempty"`
	Types                string            `json:"types,omitempty"`
	Bin                  map[string]string `json:"bin,omitempty"`
	Files                []string          `json:"files"`
	Engines              map[string]string `json:"engines,omitempty"`
	OptionalDependencies map[string]string `json:"optionalDependencies,omitempty"`
	PreferUnplugged      bool              `json:"preferUnplugged,omitempty"`
}

const (
	homepage   = "https://gnux.vercel.app"
	repository = "github:braswelljr/gnux"
)

// buildMain writes the gnux package: the launchers and the manifest that
// depends on every platform package.
func buildMain(version, dir string, tools []string) error {
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		return err
	}
	optional := map[string]string{}
	table := map[string]string{}
	for _, p := range platforms {
		optional[p.pkgName()] = version
		table[p.os+"-"+p.cpu] = p.pkgName()
	}
	bins := map[string]string{"gnux": "bin/gnux.js"}
	if err := writeLauncher(dir, "gnux", ""); err != nil {
		return err
	}
	for _, t := range tools {
		bins[t] = "bin/" + t + ".js"
		if err := writeLauncher(dir, t, t); err != nil {
			return err
		}
	}
	tableJSON, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return err
	}
	index := strings.Replace(indexJS, "PLATFORMS", string(tableJSON), 1)
	if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte(index), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "index.d.ts"), []byte(indexDTS), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readmeMain), 0o644); err != nil {
		return err
	}
	if license, err := os.ReadFile("LICENSE"); err == nil {
		if err := os.WriteFile(filepath.Join(dir, "LICENSE"), license, 0o644); err != nil {
			return err
		}
	}
	return writeJSON(filepath.Join(dir, "package.json"), manifest{
		Name:                 "gnux",
		Version:              version,
		Description:          "The GNU core utilities (rm, cp, mv, mkdir, ls, find, sed, ...) as one fast, cross-platform binary: GNU flags and messages on Linux, macOS and Windows, for package scripts and CI.",
		Keywords:             []string{"coreutils", "gnu", "rm", "rimraf", "mkdirp", "cpy", "shx", "cross-platform", "cli", "scripts"},
		License:              "MIT",
		Repository:           repository,
		Homepage:             homepage,
		Bugs:                 "https://github.com/braswelljr/gnux/issues",
		Main:                 "index.js",
		Types:                "index.d.ts",
		Bin:                  bins,
		Files:                []string{"bin", "index.js", "index.d.ts", "LICENSE"},
		Engines:              map[string]string{"node": ">=18"},
		OptionalDependencies: optional,
	})
}

// writeLauncher writes bin/<name>.js, which runs tool (gnux itself when
// tool is empty) with the launcher's arguments.
func writeLauncher(dir, name, tool string) error {
	arg := "null"
	if tool != "" {
		arg = "'" + tool + "'"
	}
	script := "#!/usr/bin/env node\n'use strict';\nrequire('../index.js').main(" + arg + ");\n"
	return os.WriteFile(filepath.Join(dir, "bin", name+".js"), []byte(script), 0o755)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// findBinary returns the gnux binary for p from dir: a bare
// gnux-<os>-<arch>[.exe], or the one inside the platform's release archive.
func findBinary(p platform, dir string) ([]byte, error) {
	bare := filepath.Join(dir, "gnux-"+p.goos+"-"+p.goarch)
	if p.goos == "windows" {
		bare += ".exe"
	}
	if data, err := os.ReadFile(bare); err == nil {
		return data, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	suffix := "-" + p.goos + "-" + p.goarch
	for _, name := range names {
		path := filepath.Join(dir, name)
		switch {
		case strings.HasSuffix(name, suffix+".tar.gz"):
			return fromTarGz(path, p.exe())
		case strings.HasSuffix(name, suffix+".zip"):
			return fromZip(path, p.exe())
		}
	}
	return nil, fmt.Errorf("no archive or binary for %s/%s in %s", p.goos, p.goarch, dir)
}

// fromTarGz reads the file called exe from a .tar.gz archive.
func fromTarGz(path, exe string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s has no %s", path, exe)
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == exe {
			return io.ReadAll(tr)
		}
	}
}

// fromZip reads the file called exe from a .zip archive.
func fromZip(path, exe string) ([]byte, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	for _, f := range z.File {
		if filepath.Base(f.Name) == exe && !f.FileInfo().IsDir() {
			r, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer r.Close()
			return io.ReadAll(r)
		}
	}
	return nil, fmt.Errorf("%s has no %s", path, exe)
}

// indexJS finds the platform's binary and runs it. PLATFORMS is replaced by
// the table of platform packages.
const indexJS = `'use strict';
// The gnux launcher: finds the binary that npm installed for this platform
// (one of the optional dependencies) and runs it.

const { spawnSync } = require('node:child_process');

const PACKAGES = PLATFORMS;

/**
 * The path of the gnux binary for this machine. GNUX_BINARY, when set, is
 * used instead, for a gnux built or installed some other way.
 */
function binaryPath() {
  if (process.env.GNUX_BINARY) {
    return process.env.GNUX_BINARY;
  }
  const key = process.platform + '-' + process.arch;
  const pkg = PACKAGES[key];
  if (!pkg) {
    throw new Error('gnux: no prebuilt binary for ' + key + '; build it with "go install github.com/braswelljr/gnux@latest" and set GNUX_BINARY to its path');
  }
  const exe = process.platform === 'win32' ? 'gnux.exe' : 'gnux';
  try {
    return require.resolve(pkg + '/bin/' + exe);
  } catch {
    throw new Error('gnux: the package ' + pkg + ' is not installed. It is an optional dependency of gnux, so it is left out when optional dependencies are (npm --omit=optional, or --no-optional); install it, or reinstall gnux with them.');
  }
}

/**
 * Runs a tool (rmx, cpx, ...) or, with tool null, gnux itself, with args, and
 * returns its exit status. Output goes to the parent's streams unless options
 * say otherwise; options are child_process.spawnSync's.
 */
function run(tool, args, options) {
  const argv = tool ? [tool].concat(args || []) : (args || []);
  const result = spawnSync(binaryPath(), argv, Object.assign({ stdio: 'inherit' }, options));
  if (result.error) {
    throw result.error;
  }
  if (result.signal) {
    return 128 + (require('node:os').constants.signals[result.signal] || 0);
  }
  return result.status;
}

/** What each launcher in bin/ calls: run the tool with the command line. */
function main(tool) {
  let status;
  try {
    status = run(tool, process.argv.slice(2));
  } catch (err) {
    process.stderr.write((err && err.message ? err.message : String(err)) + '\n');
    process.exit(127);
  }
  process.exit(status);
}

module.exports = { binaryPath, run, main };
`

const indexDTS = `import type { SpawnSyncOptions } from 'node:child_process';

/** The path of the gnux binary for this machine (or GNUX_BINARY). */
export function binaryPath(): string;

/**
 * Runs a tool (rmx, cpx, ...), or gnux itself when tool is null, and returns
 * its exit status.
 */
export function run(tool: string | null, args?: string[], options?: SpawnSyncOptions): number;

/** Runs tool with the process's command line and exits with its status. */
export function main(tool: string | null): never;
`

const readmeMain = "# gnux\n\n" +
	"The GNU core utilities as one cross-platform binary: `rmx`, `cpx`, `mvx`, `mkdirx`, `lsx`, `findx`, `sedx` and about ninety more, with GNU's flags, messages and exit statuses on Linux, macOS and Windows alike. This package installs the native binary for your machine and puts every tool on the PATH of your package scripts.\n\n" +
	"```bash\nnpm install --save-dev gnux\n```\n\n" +
	"## In package scripts\n\n" +
	"One dependency replaces rimraf, mkdirp, cpy-cli, del-cli and shx, and the commands are the ones you already know:\n\n" +
	"```json\n{\n  \"scripts\": {\n    \"clean\": \"rmx -rf dist coverage\",\n    \"prebuild\": \"mkdirx -p dist/assets\",\n    \"build:assets\": \"cpx -r public/. dist/\",\n    \"postbuild\": \"lsx -l dist\"\n  }\n}\n```\n\n" +
	"| Instead of | Write |\n|---|---|\n| `rimraf dist` | `rmx -rf dist` |\n| `mkdirp dist/a/b` | `mkdirx -p dist/a/b` |\n| `cpy 'src/**/*.css' dist` | `cpx --parents src/*.css dist` |\n| `shx cp -r a b` | `cpx -r a b` |\n| `cross-env NODE_ENV=production node build.js` | `envx NODE_ENV=production node build.js` |\n\n" +
	"Each tool is named after its GNU tool with an `x` added, so it never hides the system's own `rm` or `cp`. `gnux --list` lists them all, and `gnux help rmx` shows one's manual.\n\n" +
	"## From JavaScript\n\n" +
	"```js\nconst gnux = require('gnux');\n\nconst status = gnux.run('rmx', ['-rf', 'dist']);  // the tool's exit status\nconst path = gnux.binaryPath();                   // the binary, to run yourself\n```\n\n" +
	"## How it installs\n\n" +
	"The binary comes in a package of its own for each platform (`gnux-linux-x64`, `gnux-darwin-arm64`, `gnux-win32-x64`, ...), listed here as optional dependencies, so npm downloads only the one your machine needs and runs no install script. Installing with optional dependencies left out (`--omit=optional`) leaves gnux without a binary; set `GNUX_BINARY` to a gnux you installed another way to use that one instead.\n\n" +
	"Documentation: https://gnux.vercel.app/docs. Source: https://github.com/braswelljr/gnux.\n"
