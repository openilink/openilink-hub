#!/bin/sh
# OpeniLink Hub installer
# Usage: curl -fsSL https://raw.githubusercontent.com/openilink/openilink-hub/main/install.sh | sh
set -e

REPO="openilink/openilink-hub"
BINARY="oih"
INSTALL_DIR="/usr/local/bin"

# Colors (if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BOLD='\033[1m'
    RESET='\033[0m'
else
    RED='' GREEN='' YELLOW='' BOLD='' RESET=''
fi

info()  { printf "${GREEN}==>${RESET} ${BOLD}%s${RESET}\n" "$1"; }
warn()  { printf "${YELLOW}warning:${RESET} %s\n" "$1"; }
error() { printf "${RED}error:${RESET} %s\n" "$1" >&2; exit 1; }

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       error "Unsupported OS: $(uname -s). Only Linux and macOS are supported." ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             error "Unsupported architecture: $(uname -m). Only amd64 and arm64 are supported." ;;
    esac
}

# Get latest release tag from GitHub API
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//'
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Download and extract
download() {
    url="$1"
    dest="$2"
    info "Downloading ${url}"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" | tar xz -C "$dest"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$url" | tar xz -C "$dest"
    fi
}

main() {
    OS=$(detect_os)
    ARCH=$(detect_arch)

    # Check platform support
    if [ "$OS" = "linux" ] && [ "$ARCH" = "arm64" ]; then
        error "Linux ARM64 is not yet supported (silk audio codec requires CGO fix). Use Docker instead:
  docker run -d -p 9800:9800 ghcr.io/openilink/openilink-hub:latest"
    fi

    info "Detected: ${OS}/${ARCH}"

    # Get version
    VERSION="${OIH_VERSION:-}"
    if [ -z "$VERSION" ]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version)
        if [ -z "$VERSION" ]; then
            error "Could not determine latest version. Set OIH_VERSION=v0.0.1 to specify manually."
        fi
    fi
    info "Version: ${VERSION}"

    # Build download URL
    VERSION_NUM="${VERSION#v}"
    ARCHIVE="openilink-hub_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

    # Create temp dir
    TMP=$(mktemp -d)
    trap 'rm -rf "$TMP"' EXIT

    # Download
    download "$URL" "$TMP"

    if [ ! -f "${TMP}/${BINARY}" ]; then
        error "Binary '${BINARY}' not found in archive. Download may have failed."
    fi

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        info "Installing to ${INSTALL_DIR} (requires sudo)"
        sudo mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    fi
    chmod +x "${INSTALL_DIR}/${BINARY}"

    info "Installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"

    # Verify
    if command -v oih >/dev/null 2>&1; then
        printf "\n"
        oih version
        printf "\n"
    fi

    # Next steps
    printf "${BOLD}Next steps:${RESET}\n"
    printf "  ${GREEN}oih install${RESET}        Install as system service\n"
    printf "  ${GREEN}oih${RESET}                Run in foreground\n"
    printf "  ${GREEN}oih version${RESET}        Show version\n"
    printf "\n"
    printf "  Then open ${BOLD}http://localhost:9800${RESET} in your browser.\n"
}

main "$@"
