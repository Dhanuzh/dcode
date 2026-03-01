#!/usr/bin/env bash
# DCode installation script
# Usage: curl -fsSL https://raw.githubusercontent.com/Dhanuzh/dcode/main/install.sh | bash

set -euo pipefail

REPO="Dhanuzh/dcode"
BINARY="dcode"
VERSION="${DCODE_VERSION:-latest}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { echo -e "${CYAN}${BOLD}[dcode]${RESET} $*"; }
success() { echo -e "${GREEN}${BOLD}[dcode]${RESET} $*"; }
warn()    { echo -e "${YELLOW}${BOLD}[dcode]${RESET} $*"; }
error()   { echo -e "${RED}${BOLD}[dcode]${RESET} $*" >&2; }
die()     { error "$*"; exit 1; }

# Determine OS and architecture
detect_platform() {
    local os arch

    os="$(uname -s)"
    arch="$(uname -m)"

    case "$os" in
        Darwin)  os="darwin" ;;
        Linux)   os="linux" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *) die "Unsupported OS: $os" ;;
    esac

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) die "Unsupported architecture: $arch" ;;
    esac

    echo "${os}_${arch}"
}

# Get the latest release version from GitHub
get_latest_version() {
    if command -v curl &>/dev/null; then
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep '"tag_name"' \
            | sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
    elif command -v wget &>/dev/null; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep '"tag_name"' \
            | sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
    else
        die "Neither curl nor wget found. Please install one of them."
    fi
}

# Determine installation directory
get_install_dir() {
    if [[ -n "${DCODE_INSTALL_DIR:-}" ]]; then
        echo "$DCODE_INSTALL_DIR"
    elif [[ -n "${XDG_BIN_DIR:-}" ]]; then
        echo "$XDG_BIN_DIR"
    elif [[ -d "$HOME/bin" ]] || mkdir -p "$HOME/bin" 2>/dev/null; then
        echo "$HOME/bin"
    else
        echo "$HOME/.dcode/bin"
    fi
}

# Download and install the binary
install() {
    local platform install_dir tmp_dir download_url filename

    platform="$(detect_platform)"
    install_dir="$(get_install_dir)"

    info "Installing dcode for ${platform}..."
    info "Installation directory: ${install_dir}"

    if [[ "$VERSION" == "latest" ]]; then
        VERSION="$(get_latest_version)"
        if [[ -z "$VERSION" ]]; then
            warn "Could not determine latest version. Trying go install..."
            install_via_go
            return
        fi
        info "Latest version: ${VERSION}"
    fi

    # Construct download URL
    local ext="tar.gz"
    if [[ "$platform" == windows* ]]; then
        ext="zip"
    fi
    filename="${BINARY}_${platform}.${ext}"
    download_url="https://github.com/${REPO}/releases/download/${VERSION}/${filename}"

    # Create temp directory
    tmp_dir="$(mktemp -d)"
    trap 'rm -rf "$tmp_dir"' EXIT

    info "Downloading ${download_url}..."

    # Download
    if command -v curl &>/dev/null; then
        curl -fsSL "$download_url" -o "${tmp_dir}/${filename}" || {
            warn "Download failed. Trying go install..."
            install_via_go
            return
        }
    elif command -v wget &>/dev/null; then
        wget -q "$download_url" -O "${tmp_dir}/${filename}" || {
            warn "Download failed. Trying go install..."
            install_via_go
            return
        }
    else
        die "Neither curl nor wget found."
    fi

    # Extract
    info "Extracting..."
    if [[ "$ext" == "tar.gz" ]]; then
        tar -xzf "${tmp_dir}/${filename}" -C "$tmp_dir"
    else
        unzip -q "${tmp_dir}/${filename}" -d "$tmp_dir"
    fi

    # Find the binary
    local binary_path
    binary_path="$(find "$tmp_dir" -name "$BINARY" -type f | head -1)"
    if [[ -z "$binary_path" ]]; then
        binary_path="$(find "$tmp_dir" -name "${BINARY}.exe" -type f | head -1)"
    fi

    if [[ -z "$binary_path" ]]; then
        die "Binary not found in archive"
    fi

    # Install
    mkdir -p "$install_dir"
    cp "$binary_path" "${install_dir}/${BINARY}"
    chmod +x "${install_dir}/${BINARY}"

    success "dcode ${VERSION} installed to ${install_dir}/${BINARY}"
    check_path "$install_dir"
}

# Install via go install as fallback
install_via_go() {
    if ! command -v go &>/dev/null; then
        die "Go is not installed. Please install Go from https://go.dev/dl/ or download a binary release."
    fi

    info "Installing via go install..."
    go install "github.com/${REPO}/cmd/dcode@latest"
    success "dcode installed via go install"
    info "Binary at: $(go env GOPATH)/bin/dcode"
}

# Check if the install directory is in PATH
check_path() {
    local dir="$1"
    if [[ ":$PATH:" != *":${dir}:"* ]]; then
        warn ""
        warn "Add ${dir} to your PATH to use dcode:"
        warn ""
        if [[ -f "$HOME/.bashrc" ]]; then
            warn "  echo 'export PATH=\"\$PATH:${dir}\"' >> ~/.bashrc"
            warn "  source ~/.bashrc"
        elif [[ -f "$HOME/.zshrc" ]]; then
            warn "  echo 'export PATH=\"\$PATH:${dir}\"' >> ~/.zshrc"
            warn "  source ~/.zshrc"
        else
            warn "  export PATH=\"\$PATH:${dir}\""
        fi
        warn ""
    fi
}

# Verify installation
verify() {
    if command -v dcode &>/dev/null; then
        success "dcode is ready to use!"
        info "Run 'dcode --help' to get started."
        info "Run 'dcode auth login' to configure your AI provider."
    else
        local install_dir
        install_dir="$(get_install_dir)"
        if [[ -f "${install_dir}/dcode" ]]; then
            success "dcode installed at ${install_dir}/dcode"
            warn "Add ${install_dir} to your PATH to use 'dcode' directly."
        fi
    fi
}

# Main
main() {
    echo ""
    echo -e "${CYAN}${BOLD}"
    echo "  ____  ____  ___  ____  ___"
    echo " |  _ \/ ___|/ _ \|  _ \| __|"
    echo " | | | \___ \ | | | | | | _| "
    echo " | |_| |___) | |_| | |_| | |__"
    echo " |____/|____/ \___/|____/|____|"
    echo ""
    echo -e "${RESET}"
    echo -e "${BOLD}The AI Coding Agent${RESET}"
    echo ""

    install
    verify

    echo ""
    success "Installation complete!"
}

main "$@"
