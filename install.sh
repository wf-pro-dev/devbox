#!/usr/bin/env sh
# install.sh — installs devbox-cli from GitHub Releases
#
# Usage:
#   curl -fsSL https://github.com/wf-pro-dev/devbox/releases/latest/download/install.sh | sudo sh

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

# Global variable to store the user's RC file path for the final output message
COMPLETION_RC_FILE=""

setup_completions() {
  local bin="${INSTALL_DIR}/${BINARY_NAME}"

  # Verify the completion subcommand exists
  if ! "$bin" completion --help >/dev/null 2>&1; then
    info "Shell completion not available for this version — skipping."
    return 0
  fi

  # Bypass sudo context: Identify the actual user and their home directory
  local actual_user="${SUDO_USER:-$USER}"
  local actual_home
  actual_home=$(eval echo "~${actual_user}")

  if [ -z "$actual_home" ] || [ "$actual_home" = "/root" ]; then
    info "Warning: Could not determine non-root user home directory — skipping completion setup."
    return 0
  fi

  # Unique setup per OS: Linux -> Bash, Darwin -> Zsh
  case "$OS" in
    linux)
      local comp_dir=""
      if [ -d /etc/bash_completion.d ]; then
        comp_dir="/etc/bash_completion.d"
      elif [ -d /usr/local/etc/bash_completion.d ]; then
        comp_dir="/usr/local/etc/bash_completion.d"
      fi

      if [ -n "$comp_dir" ]; then
        info "Generating bash completion file in ${comp_dir}..."
        "$bin" completion bash | sudo tee "${comp_dir}/${BINARY_NAME}" >/dev/null
        # Set the target RC file for the user output message
        COMPLETION_RC_FILE="${actual_home}/.bashrc"
      else
        info "No supported bash completion directory found — skipping."
      fi
      ;;
    darwin)
      local zsh_dir=""
      for candidate in /usr/local/share/zsh/site-functions /usr/share/zsh/vendor-completions; do
        if [ -d "$candidate" ]; then
          zsh_dir="$candidate"
          break
        fi
      done

      if [ -n "$zsh_dir" ]; then
        info "Generating zsh completion file in ${zsh_dir}..."
        "$bin" completion zsh | sudo tee "${zsh_dir}/_${BINARY_NAME}" >/dev/null
        sudo chmod 644 "${zsh_dir}/_${BINARY_NAME}"
        # Set the target RC file for the user output message
        COMPLETION_RC_FILE="${actual_home}/.zshrc"
      else
        info "No supported zsh completion directory found — skipping."
      fi
      ;;
  esac
}

setup_completions

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "  devbox-cli ${VERSION} installed successfully."

if [ -n "$COMPLETION_RC_FILE" ]; then
  echo ""
  echo "  To activate shell completions in your current terminal, execute:"
  echo ""
  echo "    source $COMPLETION_RC_FILE"
fi
