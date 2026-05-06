#!/usr/bin/env sh
# install.sh — installs devbox-cli from GitHub Releases
#
# Usage:
#   curl -fsSL https://github.com/wf-pro-dev/devbox/releases/latest/download/install.sh | sh

set -e

REPO="wf-pro-dev/devbox"
BINARY_NAME="devbox-cli"
INSTALL_DIR="/usr/local/bin"

# ── Helpers ───────────────────────────────────────────────────────────────────

info()  { echo "[devbox] $*"; }
fatal() { echo "[devbox] error: $*" >&2; exit 1; }
need()  { command -v "$1" >/dev/null 2>&1 || fatal "'$1' is required but not found"; }

# ── Platform detection ────────────────────────────────────────────────────────

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux|darwin) ;;
  *) fatal "unsupported OS: $OS" ;;
esac

case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) fatal "unsupported architecture: $ARCH" ;;
esac

# ── Resolve latest version ────────────────────────────────────────────────────

need curl

info "Fetching latest release..."
VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed -E 's/.*"([^"]+)".*/\1/')"
[ -n "$VERSION" ] || fatal "could not determine latest release version"
info "Version: $VERSION"

# ── Download ──────────────────────────────────────────────────────────────────

ASSET="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading $ASSET..."
curl -fsSL -o "${TMPDIR}/${ASSET}" "${BASE_URL}/${ASSET}" \
  || fatal "download failed: ${BASE_URL}/${ASSET}"

# ── Verify checksum ───────────────────────────────────────────────────────────

info "Verifying checksum..."
curl -fsSL -o "${TMPDIR}/checksums.txt" "${BASE_URL}/checksums.txt" \
  || fatal "could not download checksums.txt"

EXPECTED="$(grep "$ASSET" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
[ -n "$EXPECTED" ] || fatal "no checksum entry for $ASSET"

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "${TMPDIR}/${ASSET}" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "${TMPDIR}/${ASSET}" | awk '{print $1}')"
fi

[ "$ACTUAL" = "$EXPECTED" ] || fatal "checksum mismatch — download may be corrupted"
info "Checksum OK"

# ── Install binary ────────────────────────────────────────────────────────────

tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR"
chmod +x "${TMPDIR}/${BINARY_NAME}"

info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
sudo mv "${TMPDIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"

# ── Docker group — ensure tailkitd can talk to the Docker socket ──────────────
# Docker does not require sudo. Access is controlled by the docker group.
# If tailkitd is not in that group it cannot run docker commands.

if getent group docker >/dev/null 2>&1; then
  if ! id -nG tailkitd 2>/dev/null | grep -qw docker; then
    info "Adding tailkitd to the docker group..."
    sudo usermod -aG docker tailkitd
    info "Done. The change takes effect on tailkitd's next restart."
  else
    info "tailkitd is already in the docker group."
  fi
else
  info "Warning: docker group not found — is Docker installed on this node?"
fi

# ── Shell completions ─────────────────────────────────────────────────────────

setup_completions() {
  local bin="${INSTALL_DIR}/${BINARY_NAME}"

  # Verify the completion subcommand exists
  if ! "$bin" completion --help >/dev/null 2>&1; then
    info "Shell completion not available for this version — skipping."
    return 0
  fi

  case "$OS" in
    linux)
      local bash_rc="$HOME/.bashrc"
      if [ -f "$bash_rc" ]; then
        if ! grep -qF "source <(${bin} completion bash)" "$bash_rc"; then
          info "Setting up bash completion..."
          printf '\n# Added by devbox-cli installer\nsource <(%s completion bash)\n' "$bin" >> "$bash_rc"
          info "Bash completion configured in ${bash_rc}."
        else
          info "Bash completion already configured in ${bash_rc}."
        fi
      else
        info "Warning: ${bash_rc} not found — skipping bash completion setup."
      fi
      ;;
    darwin)
      local zsh_rc="$HOME/.zshrc"
      if [ -f "$zsh_rc" ]; then
        if ! grep -qF "source <(${bin} completion zsh)" "$zsh_rc"; then
          info "Setting up zsh completion..."
          printf '\n# Added by devbox-cli installer\nsource <(%s completion zsh)\n' "$bin" >> "$zsh_rc"
          info "Zsh completion configured in ${zsh_rc}."
        else
          info "Zsh completion already configured in ${zsh_rc}."
        fi
      else
        info "Warning: ${zsh_rc} not found — skipping zsh completion setup."
      fi
      ;;
  esac
}

setup_completions

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "  devbox-cli ${VERSION} installed successfully."
echo ""
echo "  Next step — register devbox-cli in tailkitd:"
echo ""
echo "    devbox-cli setup"
echo ""
