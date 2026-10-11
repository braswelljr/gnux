#!/usr/bin/env bash
#
# Install gnux, the GNU core utilities in one cross-platform binary.
#
# Usage:
#   ./install.sh                # install the latest version with `go install`
#   VERSION=v1.2.3 ./install.sh # install a specific tag
#   ./install.sh --from-source  # build from the current checkout instead
#   ./install.sh --no-links     # install gnux only; skip the per-tool links
#
# By default every tool name (rmx, cpx, lsx, ...) is linked to the gnux binary
# so each can be run as a command of its own.
#
set -euo pipefail

MODULE="github.com/braswelljr/gnux"
VERSION="${VERSION:-latest}"
FROM_SOURCE=0
LINKS=1
for arg in "$@"; do
  case "${arg}" in
    --from-source) FROM_SOURCE=1 ;;
    --no-links) LINKS=0 ;;
    *)
      echo "error: unknown option '${arg}'" >&2
      exit 2
      ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  echo "error: the Go toolchain is required. Install it from https://go.dev/dl/ and re-run." >&2
  exit 1
fi

if [ "${FROM_SOURCE}" -eq 1 ]; then
  version="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
  echo "Building gnux ${version} from source..."
  go install -trimpath \
    -ldflags "-s -w -X ${MODULE}/internal/common.Version=${version}" \
    .
else
  echo "Installing ${MODULE}@${VERSION}..."
  go install "${MODULE}@${VERSION}"
fi

bindir="$(go env GOBIN)"
[ -z "${bindir}" ] && bindir="$(go env GOPATH)/bin"
echo "Installed gnux to ${bindir}/gnux"

if [ "${LINKS}" -eq 1 ]; then
  count=0
  for tool in $("${bindir}/gnux" --list); do
    ln -sf gnux "${bindir}/${tool}"
    count=$((count + 1))
  done
  echo "Linked ${count} tools (rmx, cpx, lsx, ...) in ${bindir}"
fi

echo "Ensure ${bindir} is on your PATH, then run: gnux --help"
