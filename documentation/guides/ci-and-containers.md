# CI and containers

Recipes for using gnux in continuous integration and container images: installing it on GitHub Actions and GitLab CI runners, replacing per-OS `bash` and PowerShell steps with one `gnux shell` script that runs identically on Linux, macOS and Windows runners, building and using the gnux Docker image, copying the static binary into `scratch`, distroless and other minimal images through multi-stage builds, and the exit-status conventions that make gnux steps fail exactly when they should.

## Why gnux in CI

A matrix build across `ubuntu-latest`, `macos-latest` and `windows-latest` usually needs every file step written twice, or forced through Git Bash on Windows. With gnux:

- one script, run with `gnux shell -e`, does the same thing on all three runners;
- `rmx -rf`, `mkdirx -p` and `cpx -r` behave identically, including their diagnostics;
- the only dependency is one static binary, installed in seconds with `go install`;
- exit statuses follow GNU, and `-e` stops at the first failure, so a broken step fails the job.

## GitHub Actions

### Install once, use everywhere

`actions/setup-go` puts `$(go env GOPATH)/bin` on the `PATH`, so a `go install` is all it takes. Link the tool names if you want `rmx` rather than `gnux rmx`.

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.25'

- name: Install gnux
  shell: bash
  run: |
    go install github.com/braswelljr/gnux@latest
    bin="$(go env GOPATH)/bin"
    ext=""; [ "$RUNNER_OS" = "Windows" ] && ext=".exe"
    for t in $(gnux --list); do cp "$bin/gnux$ext" "$bin/$t$ext"; done
```

Copies rather than symbolic links keep the step identical on Windows runners, where creating a symlink needs privileges. To pin a version, replace `@latest` with a tag.

### One cross-OS build step

```yaml
# .github/workflows/package.yml
name: Package

on:
  push:
    branches: [main]
  pull_request:

jobs:
  package:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Install gnux
        run: go install github.com/braswelljr/gnux@latest

      # The same script on every runner: no bash on Windows, no pwsh on Linux.
      - name: Stage the release tree
        run: gnux shell -e scripts/stage.gnux

      - uses: actions/upload-artifact@v4
        with:
          name: dist-${{ matrix.os }}
          path: dist/
```

And the script it runs:

```bash
# scripts/stage.gnux - identical on Linux, macOS and Windows runners.
OUT=dist

rmx -rf $OUT
mkdirx -p $OUT/bin $OUT/docs
envx go build -o $OUT/bin/ ./...
cpx -v README.md LICENSE $OUT/docs/
echox $GITHUB_SHA > $OUT/COMMIT
lsx -lR $OUT
```

`$GITHUB_SHA` is not set in the script, so gnux reads it from the environment. `envx go build` reaches the Go toolchain on the runner's `PATH`.

### One-liners in a step

`gnux shell -c` is handy when a step is too small for a file. The step's own shell (bash on Linux and macOS, pwsh on Windows) only has to pass a single-quoted string through, which both do the same way:

```yaml
- name: Clean
  run: gnux shell -e -c 'rmx -rf dist coverage.out && mkdirx -p dist'

- name: Report sizes
  run: gnux shell -c 'lsx -lhS dist | headx -n 10'

- name: Count test files
  run: gnux shell -c 'lsx -1 *_test.go | wcx -l'
```

### Fail on a condition

```yaml
- name: Fail if the build left temporary files
  run: gnux shell -c 'lsx *.tmp > /dev/null 2>/dev/null && echox "temporary files left behind" && exit 1 || truex'
```

### Write a job summary

```yaml
- name: Summarize
  run: gnux shell -c 'echox "## Build output" >> $GITHUB_STEP_SUMMARY && lsx -lh dist >> $GITHUB_STEP_SUMMARY'
```

`$GITHUB_STEP_SUMMARY` comes from the runner's environment and the append is done by gnux, with the same bytes on every OS.

### Wait for a service

```yaml
- name: Start the server
  run: |
    ./bin/server &
    sleepx 2
```

`sleepx` takes fractions and units (`0.5`, `2m`) on every runner, including Windows where `sleep` is a PowerShell alias for `Start-Sleep`.

## GitLab CI

```yaml
# .gitlab-ci.yml
stages: [build, package]

variables:
  GOPATH: "$CI_PROJECT_DIR/.go"

.gnux:
  image: golang:1.25
  before_script:
    - go install github.com/braswelljr/gnux@latest
    - export PATH="$GOPATH/bin:$PATH"
  cache:
    paths: [.go/pkg/mod]

build:
  extends: .gnux
  stage: build
  script:
    - gnux shell -e scripts/stage.gnux
  artifacts:
    paths: [dist/]

package:
  extends: .gnux
  stage: package
  script:
    - gnux shell -e -c 'mkdirx -p public && cpx -r dist/. public/ && lsx -lR public'
  artifacts:
    paths: [public/]
```

On a Windows runner (PowerShell executor) the `script` lines stay the same; only the `PATH` line changes:

```yaml
windows-build:
  tags: [windows]
  before_script:
    - go install github.com/braswelljr/gnux@latest
    - $env:Path = "$(go env GOPATH)\bin;$env:Path"
  script:
    - gnux shell -e scripts/stage.gnux
```

## Other CI systems

Any system that can run a command works the same way: install with `go install` (or copy a prebuilt binary into the image), then call `gnux shell -e script.gnux`. A few equivalents:

```yaml
# Azure Pipelines
- script: go install github.com/braswelljr/gnux@latest
- script: gnux shell -e scripts/stage.gnux
```

```yaml
# CircleCI
- run: go install github.com/braswelljr/gnux@latest
- run: gnux shell -e scripts/stage.gnux
```

## Exit statuses in CI

| What happens | Status | Effect on the step |
|--------------|--------|--------------------|
| Everything succeeded | 0 | Passes |
| A tool failed (`rmx: cannot remove ...`) | 1 | Fails; with `-e` the script stops there |
| `lsx` could not access an operand | 2 | Fails |
| `envx` could not find the program | 127 | Fails |
| Syntax error in the script | 2 | Fails before anything after it runs |
| Unknown command in the script | 127 | Fails (with `-e`, immediately) |

Two habits keep CI steps honest:

- **Use `-e`** so the first failure stops the script, instead of the last command's status deciding the step.
- **Use `rmx -f` for cleanup** so a missing file is not a failure, and `|| truex` for any other failure you expect.

## Docker

### The gnux image

The repository's `Dockerfile` builds a static binary in `golang:1.25-alpine`, links every tool name next to it, and copies the result into `alpine:3.20`, where it runs as uid 10001 in `/work` with `gnux` as the entry point:

```bash
docker build -t gnux --build-arg VERSION=v0.1.0 .

docker run --rm gnux --list
docker run --rm -v "$(pwd):/work" gnux lsx -la
docker run --rm -v "$(pwd):/work" gnux shell -e scripts/stage.gnux
docker run --rm -v "$(pwd):/work" --entrypoint rmx gnux -rv build/
```

```powershell
docker run --rm -v "${PWD}:/work" gnux shell -e scripts/stage.gnux
```

Because the container runs as uid 10001, files it creates on a bind mount belong to that user. Add `--user "$(id -u):$(id -g)"` on Linux to create them as yourself.

### Add gnux to your own image

The binary is static and has no runtime dependencies, so it can be copied into any image, including `scratch` and distroless ones that have no shell and no coreutils at all:

```dockerfile
# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS gnux
RUN CGO_ENABLED=0 go install -trimpath -ldflags "-s -w" github.com/braswelljr/gnux@latest \
 && mkdir /out && cp /go/bin/gnux /out/ \
 && cd /out && for t in $(./gnux --list); do ln -s gnux "$t"; done

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app .

FROM gcr.io/distroless/static-debian12
COPY --from=gnux /out/ /usr/local/bin/
COPY --from=build /out/app /app
# Prepare the runtime directories, then start the app - no shell in the image.
ENTRYPOINT ["/usr/local/bin/gnux", "shell", "-e", "-c", "mkdirx -p /tmp/cache /tmp/logs && envx /app"]
```

The last line is the point: a distroless image has no `/bin/sh`, but `gnux shell -c` gives the entry point `&&`, variables and the file tools anyway, and `envx` starts the application.

### Run file steps in a `RUN` without a shell

In a `FROM scratch` stage there is no `/bin/sh`, so `RUN` must use the exec form. gnux makes that bearable:

```dockerfile
FROM scratch
COPY --from=gnux /out/gnux /gnux
COPY site/ /site/
RUN ["/gnux", "shell", "-e", "-c", "mkdirx -p /site/cache && rmx -rf /site/drafts && lsx -lR /site"]
```

### Clean a build context inside a multi-stage build

```dockerfile
FROM node:22 AS web
COPY --from=gnux /out/ /usr/local/bin/
WORKDIR /app
COPY . .
RUN gnux shell -e -c 'rmx -rf node_modules/.cache dist && mkdirx -p dist'
RUN npm ci && npm run build
```

### Docker Compose

The repository's `docker-compose.yml` builds the image and mounts the current directory at `/work`:

```bash
docker compose run --rm gnux --version
docker compose run --rm gnux lsx -la
docker compose run --rm gnux shell -e scripts/stage.gnux
```

These are real file tools working on your files through the mount.

## Testing your scripts locally

Before pushing, run the CI script exactly as the runner will:

```bash
gnux shell -e scripts/stage.gnux; echo "status $?"
docker run --rm -v "$(pwd):/work" gnux shell -e scripts/stage.gnux
```

If it works in both, it works on all three runners: the container covers Linux, your machine covers your OS, and the script contains nothing platform-specific.

## See also

- [Portable scripting](portable-scripting.md): the language and four complete scripts.
- [Windows and PowerShell](windows-and-powershell.md): Windows runners in more depth.
- [Getting started](getting-started.md): every installation route.
- The project's own CI is described in [workflows](../../docs/workflows/README.md); its requirements are NFR entries in the [non-functional requirements](../../docs/requirements/non-functional-requirements.md).
