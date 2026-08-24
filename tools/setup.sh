#!/usr/bin/env bash
#
# Barbara dev environment bootstrap.
#
# Idempotent: run it as many times as you like. Each step checks what is
# already installed at the pinned version and skips if it is satisfied.
#
# A fresh container has no Go and no make, so bootstrap directly the first time:
#
#     bash tools/setup.sh
#
# After that, `make setup` runs this same script.
#
# Version pins below are the source of truth for local dev. Keep them in sync
# with:
#   - Go            -> go.mod `toolchain`
#   - golangci-lint -> Makefile `install-tools` / .github/workflows/ci.yml
#   - gosec         -> .github/workflows/ci.yml
#   - air           -> Makefile `install-tools`
#   - pnpm          -> web/package.json `packageManager` (once web/ lands)
set -euo pipefail

GO_VERSION="1.25.3"
GOLANGCI_LINT_VERSION="v2.7.2"
GOSEC_VERSION="v2.22.11"
AIR_VERSION="latest"
PNPM_VERSION="10.27.0"
NODE_MIN_MAJOR="22"

GO_ROOT="/usr/local/go"
PROFILE_SNIPPET="/etc/profile.d/barbara-dev.sh"

# ---------------------------------------------------------------------------
# output helpers
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
	C_BLUE=$'\033[36m'; C_GREEN=$'\033[32m'; C_YELLOW=$'\033[33m'; C_RED=$'\033[31m'; C_OFF=$'\033[0m'
else
	C_BLUE=""; C_GREEN=""; C_YELLOW=""; C_RED=""; C_OFF=""
fi
step() { printf '%s==>%s %s\n' "$C_BLUE" "$C_OFF" "$*"; }
ok()   { printf '%s  ok%s %s\n' "$C_GREEN" "$C_OFF" "$*"; }
warn() { printf '%swarn%s %s\n' "$C_YELLOW" "$C_OFF" "$*"; }
die()  { printf '%s err%s %s\n' "$C_RED" "$C_OFF" "$*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# platform detection
# ---------------------------------------------------------------------------
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
[ "$OS" = "linux" ] || die "this bootstrap targets Linux; found '$OS'. Install the pinned tools manually."

case "$(uname -m)" in
	x86_64|amd64) ARCH="amd64" ;;
	aarch64|arm64) ARCH="arm64" ;;
	*) die "unsupported arch '$(uname -m)'" ;;
esac

# Everything here writes to /usr/local, /etc, and go install dirs. On a fresh
# container we are usually root; otherwise sudo is needed for those writes.
SUDO=""
if [ "$(id -u)" -ne 0 ]; then
	if command -v sudo >/dev/null 2>&1; then
		SUDO="sudo"
	else
		die "not root and no sudo; cannot write to $GO_ROOT. Re-run as root."
	fi
fi

# ---------------------------------------------------------------------------
# make — needed for the Makefile targets that wrap this script
# ---------------------------------------------------------------------------
step "make"
if command -v make >/dev/null 2>&1; then
	ok "make present ($(make --version | head -1))"
elif command -v apt-get >/dev/null 2>&1; then
	$SUDO apt-get update -qq
	$SUDO apt-get install -y -qq make
	ok "installed make via apt"
else
	warn "make not found and no apt-get; install make yourself or the Makefile targets won't run"
fi

# ---------------------------------------------------------------------------
# Go toolchain
# ---------------------------------------------------------------------------
step "Go ${GO_VERSION}"
need_go=1
if [ -x "${GO_ROOT}/bin/go" ]; then
	current="$("${GO_ROOT}/bin/go" version 2>/dev/null | awk '{print $3}')"
	if [ "$current" = "go${GO_VERSION}" ]; then
		need_go=0
		ok "go${GO_VERSION} already installed at ${GO_ROOT}"
	else
		warn "found ${current} at ${GO_ROOT}, replacing with go${GO_VERSION}"
	fi
fi
if [ "$need_go" -eq 1 ]; then
	tarball="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
	url="https://go.dev/dl/${tarball}"
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	step "downloading ${url}"
	curl -fsSL "$url" -o "${tmp}/${tarball}" || die "failed to download ${url}"
	$SUDO rm -rf "$GO_ROOT"
	$SUDO tar -C /usr/local -xzf "${tmp}/${tarball}"
	installed="$("${GO_ROOT}/bin/go" version 2>/dev/null | awk '{print $3}')"
	[ "$installed" = "go${GO_VERSION}" ] || die "go install verification failed (got '${installed}')"
	rm -rf "$tmp"; trap - EXIT
	ok "installed ${installed}"
fi

# Make the freshly installed Go usable for the rest of this script.
export PATH="${GO_ROOT}/bin:${PATH}"
GOBIN_DIR="$(go env GOPATH)/bin"
export PATH="${GOBIN_DIR}:${PATH}"
mkdir -p "$GOBIN_DIR"

# ---------------------------------------------------------------------------
# Go-based tools (installed into $(go env GOPATH)/bin)
# ---------------------------------------------------------------------------
step "golangci-lint ${GOLANGCI_LINT_VERSION}"
want="${GOLANGCI_LINT_VERSION#v}"
if command -v golangci-lint >/dev/null 2>&1 && golangci-lint version 2>/dev/null | grep -q "$want"; then
	ok "golangci-lint ${GOLANGCI_LINT_VERSION} already installed"
else
	go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"
	ok "installed golangci-lint ${GOLANGCI_LINT_VERSION}"
fi

step "gosec ${GOSEC_VERSION}"
# `gosec --version` reports "dev" when installed via `go install` (no ldflags
# stamp), so read the real module version out of the binary's build info.
if command -v gosec >/dev/null 2>&1 && \
	go version -m "$(command -v gosec)" 2>/dev/null | grep -q "securego/gosec/v2[[:space:]]*${GOSEC_VERSION}\b"; then
	ok "gosec ${GOSEC_VERSION} already installed"
else
	go install "github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION}"
	ok "installed gosec ${GOSEC_VERSION}"
fi

step "air (${AIR_VERSION})"
if command -v air >/dev/null 2>&1; then
	ok "air already installed ($(air -v 2>/dev/null | head -1))"
else
	go install "github.com/air-verse/air@${AIR_VERSION}"
	ok "installed air"
fi

# ---------------------------------------------------------------------------
# Node / pnpm (node is expected to be provided by the base image).
# Barbara has no web/ workspace yet; these steps warn-and-continue until it
# lands, then become required for the web build.
# ---------------------------------------------------------------------------
step "node >= ${NODE_MIN_MAJOR}"
if command -v node >/dev/null 2>&1; then
	node_major="$(node -v | sed 's/^v//; s/\..*//')"
	if [ "$node_major" -ge "$NODE_MIN_MAJOR" ]; then
		ok "node $(node -v) satisfies >= ${NODE_MIN_MAJOR}"
	else
		warn "node $(node -v) is below required >= ${NODE_MIN_MAJOR}; upgrade node for the web workspace"
	fi
else
	warn "node not found; install node >= ${NODE_MIN_MAJOR} for the web workspace (needed once web/ lands)"
fi

step "pnpm ${PNPM_VERSION}"
if command -v pnpm >/dev/null 2>&1 && [ "$(pnpm -v 2>/dev/null)" = "$PNPM_VERSION" ]; then
	ok "pnpm ${PNPM_VERSION} already installed"
elif command -v corepack >/dev/null 2>&1; then
	corepack enable
	corepack prepare "pnpm@${PNPM_VERSION}" --activate
	ok "activated pnpm ${PNPM_VERSION} via corepack"
elif command -v npm >/dev/null 2>&1; then
	$SUDO npm install -g "pnpm@${PNPM_VERSION}"
	ok "installed pnpm ${PNPM_VERSION} via npm"
else
	warn "no corepack or npm; install pnpm ${PNPM_VERSION} yourself for the web workspace"
fi

# ---------------------------------------------------------------------------
# Docker — only needed for the compose-based dev stack / integration tests.
# We don't install a daemon here (it's a host/container concern), just report.
# ---------------------------------------------------------------------------
step "docker (optional)"
if command -v docker >/dev/null 2>&1; then
	ok "docker present ($(docker --version 2>/dev/null))"
else
	warn "docker not found — needed for 'make dev', 'make test-integration', and the compose stack (docker-compose.yml). Install it separately if you need the DB stack."
fi

# ---------------------------------------------------------------------------
# Persist PATH so future shells find go and the go-installed tools
# ---------------------------------------------------------------------------
step "PATH"
path_line="export PATH=\"${GO_ROOT}/bin:${GOBIN_DIR}:\$PATH\""
persisted=0
if [ -d "$(dirname "$PROFILE_SNIPPET")" ]; then
	if printf '#!/bin/sh\n%s\n' "$path_line" | $SUDO tee "$PROFILE_SNIPPET" >/dev/null 2>&1; then
		$SUDO chmod +x "$PROFILE_SNIPPET" 2>/dev/null || true
		ok "wrote ${PROFILE_SNIPPET}"
		persisted=1
	fi
fi
if [ "$persisted" -eq 0 ]; then
	# Fall back to the user's shell rc files, guarded against duplicates.
	for rc in "$HOME/.bashrc" "$HOME/.profile"; do
		[ -e "$rc" ] || touch "$rc"
		if ! grep -qF "$path_line" "$rc" 2>/dev/null; then
			printf '\n# barbara dev toolchain\n%s\n' "$path_line" >>"$rc"
			ok "appended PATH to ${rc}"
		fi
	done
fi

printf '\n%sBarbara dev environment ready.%s\n' "$C_GREEN" "$C_OFF"
printf 'Open a new shell, or run this now to use the tools in the current one:\n\n'
printf '    %s\n\n' "$path_line"
printf 'Then verify with: make check\n'
