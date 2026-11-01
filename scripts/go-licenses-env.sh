#!/usr/bin/env bash
# Shared setup for github.com/google/go-licenses (pinned) on Go 1.21+ toolchains.
set -euo pipefail

GO_LICENSES_VERSION="${GO_LICENSES_VERSION:-v1.6.0}"

unset GOROOT
go install "github.com/google/go-licenses@${GO_LICENSES_VERSION}"
export PATH="$(go env GOPATH)/bin:${PATH}"

_kanboard_licenses_goroot() {
  local toolchain="${GOMODCACHE:-$(go env GOMODCACHE)}/golang.org/toolchain@v0.0.1-$(go env GOVERSION).$(go env GOOS)-$(go env GOARCH)"
  if [ -d "${toolchain}/src" ]; then
    printf '%s' "${toolchain}"
  else
    go env GOROOT
  fi
}

kanboard_go_licenses() {
  GOROOT="$(_kanboard_licenses_goroot)" go-licenses "$@"
}

KANBOARD_LICENSE_PACKAGE="${KANBOARD_LICENSE_PACKAGE:-github.com/swtormy-agents/kanboard-cli/cmd/kanboard}"
