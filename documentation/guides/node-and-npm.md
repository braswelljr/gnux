# Node and npm

gnux on npm: one development dependency that puts `rmx`, `cpx`, `mkdirx`, `lsx` and every other tool on the PATH of your package scripts, with GNU's flags and messages on Linux, macOS and Windows alike. It replaces rimraf, mkdirp, cpy-cli, del-cli, shx and cross-env with the commands you already know, and runs the native binary, not JavaScript.

## Install

```bash
npm install --save-dev gnux
pnpm add -D gnux
yarn add -D gnux
bun add -d gnux
```

The `gnux` package is a few kilobytes of JavaScript. The binary comes in a package of its own for each platform, listed as optional dependencies and limited to their operating system and CPU, so the package manager downloads only the one your machine needs and runs no install script:

| Platform | Package |
|----------|---------|
| Linux on x64, arm64, riscv64 | `gnux-linux-x64`, `gnux-linux-arm64`, `gnux-linux-riscv64` |
| macOS on Intel and Apple silicon | `gnux-darwin-x64`, `gnux-darwin-arm64` |
| Windows on x64 and arm64 | `gnux-win32-x64`, `gnux-win32-arm64` |
| FreeBSD on x64 and arm64 | `gnux-freebsd-x64`, `gnux-freebsd-arm64` |

Every package of a release has the same version, the release's own, so `gnux@1.2.0` always runs gnux 1.2.0.

## In package scripts

Each tool is installed under its gnux name, the GNU name with an `x` added, and never under the GNU name: `rm` in a script is still the system's `rm`, and `rmx` is gnux's, the same on every operating system.

```json
{
  "scripts": {
    "clean": "rmx -rf dist coverage .cache",
    "prebuild": "mkdirx -p dist/assets",
    "build": "tsc && cpx -r public/. dist/",
    "postbuild": "lsx -l dist",
    "size": "dux -sh dist",
    "dev": "envx NODE_ENV=development node server.js"
  }
}
```

| Instead of | Write |
|------------|-------|
| `rimraf dist` | `rmx -rf dist` |
| `del-cli "dist/**"` | `rmx -rf dist` |
| `mkdirp dist/a/b` | `mkdirx -p dist/a/b` |
| `make-dir dist/a/b` | `mkdirx -p dist/a/b` |
| `cpy-cli src/index.html dist` | `cpx src/index.html dist` |
| `copyfiles -u 1 "src/**/*.css" dist` | `gnux shell -c 'cd src && cpx --parents */*.css ../dist'` |
| `shx cp -r a b` | `cpx -r a b` |
| `shx mv a b` | `mvx a b` |
| `shx cat a b > c` | `catx a b > c` |
| `cross-env NODE_ENV=production node build.js` | `envx NODE_ENV=production node build.js` |
| `touch-cli .env` | `touchx .env` |

`gnux --list` prints every tool, and `gnux help rmx` shows one's manual.

## Globs and other shell features

npm runs a script through `sh` on Linux and macOS but through `cmd.exe` on Windows, and `cmd.exe` does not expand `*`: `cpx src/*.css dist` copies a file literally named `src/*.css` there, or fails. When a script needs a glob, `&&` chains that differ between shells, a loop or a redirection that must mean the same everywhere, run it with `gnux shell`, which expands globs itself:

```json
{
  "scripts": {
    "copy:css": "gnux shell -c 'cpx src/*.css dist/'",
    "clean:logs": "gnux shell -c 'for f in logs/*.log; do rmx -f \"$f\"; done'",
    "release": "gnux shell -e scripts/release.gnux"
  }
}
```

The [portable scripting](portable-scripting.md) guide covers the language.

## From JavaScript

```js
const gnux = require('gnux');

// Run a tool with the parent's output streams; returns its exit status.
const status = gnux.run('rmx', ['-rf', 'dist']);

// Capture the output instead (any child_process.spawnSync option works).
const listing = gnux.run('lsx', ['-1', 'src'], { stdio: 'pipe' });

// The binary itself, to run with your own spawn, execa or zx.
const binary = gnux.binaryPath();
```

`run` with `null` as the tool runs `gnux` itself (`gnux.run(null, ['--version'])`). Type definitions ship with the package.

## When the binary is missing

- **Optional dependencies left out.** `npm install --omit=optional` (or `--no-optional`, or a `.npmrc` with `optional=false`) skips the platform package, and each tool then says `gnux: the package gnux-linux-x64 is not installed` and exits 127. Install with optional dependencies, or add the platform package yourself.
- **A platform without a package.** gnux names the platform and suggests building it with `go install github.com/braswelljr/gnux@latest`.
- **A gnux of your own.** Set `GNUX_BINARY` to its path, and every launcher runs that binary instead: a build from source, the Homebrew one, or the one a CI image already has.

## In CI

The binary comes from the npm registry with the rest of `node_modules`, so a cached install has it too, and nothing else needs installing on the runner. A matrix over operating systems runs the same scripts unchanged:

```yaml
jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v7
        with:
          node-version: 22
          cache: npm
      - run: npm ci
      - run: npm run clean && npm run build
```

Outside a Node project, the [setup-gnux action](ci-and-containers.md) installs the binary directly.

## Each package's provenance

Every package is published from the release workflow with npm's provenance statement, which ties it to the commit and the workflow run that built it. `npm audit signatures` checks it.
