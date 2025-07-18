#!/bin/bash
# Orchestrator installation script
# This script installs orchestrator in an XDG-compliant manner

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="orc"
REPO_URL="https://github.com/dotcommander/orc"

# XDG directories
XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
XDG_DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
XDG_BIN_HOME="${XDG_BIN_HOME:-$HOME/.local/bin}"

INSTALL_BIN_DIR="$XDG_BIN_HOME"
INSTALL_CONFIG_DIR="$XDG_CONFIG_HOME/orchestrator"
INSTALL_DATA_DIR="$XDG_DATA_HOME/orchestrator"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed. Please install Go 1.21 or later."
        log_info "Visit: https://golang.org/doc/install"
        exit 1
    fi
    
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    min_version="1.21"
    if [ "$(printf '%s\n' "$min_version" "$go_version" | sort -V | head -n1)" != "$min_version" ]; then
        log_error "Go version $go_version is too old. Minimum required: $min_version"
        exit 1
    fi
    
    log_success "Go $go_version is installed"
}

create_directories() {
    log_info "Creating XDG-compliant directories..."
    
    mkdir -p "$INSTALL_BIN_DIR"
    mkdir -p "$INSTALL_CONFIG_DIR"
    mkdir -p "$INSTALL_DATA_DIR/prompts"
    
    log_success "Directories created"
}

install_from_source() {
    log_info "Installing from source..."
    
    # Create temporary directory
    temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    # Clone or copy source
    if [ -d "$1" ]; then
        log_info "Installing from local source: $1"
        cp -r "$1"/* .
    else
        log_info "Cloning from repository..."
        git clone "$REPO_URL" .
    fi
    
    # Build and install
    log_info "Building orchestrator..."
    make build
    
    log_info "Installing binary..."
    cp bin/orc "$INSTALL_BIN_DIR/"
    chmod +x "$INSTALL_BIN_DIR/orc"
    
    # Install configuration files
    if [ ! -f "$INSTALL_CONFIG_DIR/config.yaml" ]; then
        log_info "Installing default configuration..."
        cp config.yaml.example "$INSTALL_CONFIG_DIR/config.yaml"
    else
        log_warning "Configuration file already exists, skipping"
    fi
    
    if [ ! -f "$INSTALL_CONFIG_DIR/.env" ]; then
        log_info "Installing example environment file..."
        cp .env.example "$INSTALL_CONFIG_DIR/.env"
    else
        log_warning "Environment file already exists, skipping"
    fi
    
    # Install prompt templates
    if [ -d "prompts" ]; then
        log_info "Installing prompt templates..."
        cp prompts/*.txt "$INSTALL_DATA_DIR/prompts/" 2>/dev/null || true
    fi
    
    # Create symlink in go/bin if it exists (user preference)
    if [ -d "$HOME/go/bin" ]; then
        log_info "Creating symlink in ~/go/bin..."
        ln -sf "$INSTALL_BIN_DIR/orc" "$HOME/go/bin/orc"
    fi
    
    # Cleanup
    cd - >/dev/null
    rm -rf "$temp_dir"
    
    log_success "Installation complete!"
}

update_path() {
    # Check if binary directory is in PATH
    if ! echo "$PATH" | grep -q "$INSTALL_BIN_DIR"; then
        log_warning "Binary directory not in PATH: $INSTALL_BIN_DIR"
        log_info "Add to your shell profile:"
        echo "    export PATH=\"$INSTALL_BIN_DIR:\$PATH\""
        echo ""
        
        # Detect shell and suggest specific file
        if [ -n "$ZSH_VERSION" ]; then
            log_info "For Zsh, add to ~/.zshrc"
        elif [ -n "$BASH_VERSION" ]; then
            log_info "For Bash, add to ~/.bashrc or ~/.bash_profile"
        else
            log_info "Add to your shell's configuration file"
        fi
    else
        log_success "Binary directory is already in PATH"
    fi
}

show_next_steps() {
    echo ""
    log_success "Orchestrator has been installed successfully!"
    echo ""
    echo "Installation locations:"
    echo "  Binary:     $INSTALL_BIN_DIR/orc"
    echo "  Config:     $INSTALL_CONFIG_DIR/config.yaml"
    echo "  Data:       $INSTALL_DATA_DIR/"
    echo ""
    echo "Next steps:"
    echo "  1. Add your Anthropic API key to $INSTALL_CONFIG_DIR/.env"
    echo "  2. Customize prompts in $INSTALL_DATA_DIR/prompts/"
    echo "  3. Run: orc \"Write a science fiction novel about time travel\""
    echo ""
    echo "For help: orc -help"
    echo "Version:  orc -version"
}

uninstall() {
    log_info "Uninstalling orchestrator..."
    
    rm -f "$INSTALL_BIN_DIR/orc"
    rm -f "$HOME/go/bin/orc"
    
    log_success "Binary removed"
    log_info "Configuration files preserved in $INSTALL_CONFIG_DIR"
    log_info "To remove all data: rm -rf \"$INSTALL_CONFIG_DIR\" \"$INSTALL_DATA_DIR\""
}

# Main script
main() {
    case "${1:-install}" in
        install)
            check_dependencies
            create_directories
            install_from_source "${2:-.}"
            update_path
            show_next_steps
            ;;
        uninstall)
            uninstall
            ;;
        *)
            echo "Usage: $0 [install|uninstall] [source_dir]"
            echo ""
            echo "install    Install orchestrator (default)"
            echo "uninstall  Remove orchestrator binary"
            echo ""
            echo "Examples:"
            echo "  $0                    # Install from current directory"
            echo "  $0 install .          # Install from current directory"
            echo "  $0 uninstall          # Remove installation"
            exit 1
            ;;
    esac
}

main "$@"