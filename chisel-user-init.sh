#!/bin/bash
#
# Chisel User-Level Setup Script
# Initializes chisel for user-level package management
#
# This script sets up:
# - User-level chisel directories (~/.local/share/chisel/)
# - User configuration file (~/.config/chisel/config.json)
# - User symlink directory (~/.local/bin/)
# - Shell configuration for PATH and LD_LIBRARY_PATH
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHISEL_BIN="${SCRIPT_DIR}/chisel"

# Determine user directories following XDG spec
XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
XDG_DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
CHISEL_CONFIG_DIR="${XDG_CONFIG_HOME}/chisel"
CHISEL_DATA_DIR="${XDG_DATA_HOME}/chisel"
CHISEL_BIN_DIR="${HOME}/.local/bin"
CHISEL_CONFIG_FILE="${CHISEL_CONFIG_DIR}/config.json"

# Parse arguments
FORCE=false
SHELL_CONFIG_ONLY=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE=true
            shift
            ;;
        --shell-config-only)
            SHELL_CONFIG_ONLY=true
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --force                 Overwrite existing configuration"
            echo "  --shell-config-only     Only update shell configuration (don't create dirs)"
            echo "  --help                  Show this help message"
            echo ""
            echo "This script initializes chisel for user-level package management."
            echo "It creates:"
            echo "  - ${CHISEL_DATA_DIR}"
            echo "  - ${CHISEL_CONFIG_FILE}"
            echo "  - ${CHISEL_BIN_DIR}"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Helper functions
log_step() {
    echo -e "${BLUE}==>${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_info() {
    echo -e "   $1"
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites..."
    
    if [[ ! -x "$CHISEL_BIN" ]]; then
        log_error "Chisel binary not found at $CHISEL_BIN"
        log_info "Run: go build -o chisel ./cmd/chisel"
        exit 1
    fi
    
    log_success "Chisel binary found"
    
    # Check if chisel supports user-level config
    if ! "$CHISEL_BIN" --help 2>&1 | grep -q "Phase 5"; then
        log_warning "Chisel might not have latest user-level support"
    fi
    
    echo ""
}

# Create directories
create_directories() {
    log_step "Creating user-level directories..."
    
    # Create config directory
    if [[ ! -d "$CHISEL_CONFIG_DIR" ]]; then
        mkdir -p "$CHISEL_CONFIG_DIR"
        log_info "Created $CHISEL_CONFIG_DIR"
    else
        log_info "$CHISEL_CONFIG_DIR already exists"
    fi
    
    # Create data directory
    if [[ ! -d "$CHISEL_DATA_DIR" ]]; then
        mkdir -p "$CHISEL_DATA_DIR"
        log_info "Created $CHISEL_DATA_DIR"
    else
        log_info "$CHISEL_DATA_DIR already exists"
    fi
    
    # Create bin directory
    if [[ ! -d "$CHISEL_BIN_DIR" ]]; then
        mkdir -p "$CHISEL_BIN_DIR"
        log_info "Created $CHISEL_BIN_DIR"
    else
        log_info "$CHISEL_BIN_DIR already exists"
    fi
    
    # Create subdirectories in CHISEL_DATA_DIR
    for dir in store db wrappers cache; do
        if [[ ! -d "$CHISEL_DATA_DIR/$dir" ]]; then
            mkdir -p "$CHISEL_DATA_DIR/$dir"
            log_info "Created $CHISEL_DATA_DIR/$dir"
        fi
    done
    
    echo ""
}

# Create configuration file
create_config() {
    log_step "Creating user-level configuration..."
    
    if [[ -f "$CHISEL_CONFIG_FILE" ]] && [[ "$FORCE" != true ]]; then
        log_warning "Configuration already exists at $CHISEL_CONFIG_FILE"
        log_info "Use --force to overwrite"
        echo ""
        return
    fi
    
    # Create config JSON
    cat > "$CHISEL_CONFIG_FILE" << 'EOF'
{
  "base_dir": "will_be_set_by_user_config",
  "symlink_root": "will_be_set_by_user_config",
  "mirror_url": "https://mirror.rackspace.com/archlinux",
  "architecture": "x86_64",
  "repositories": ["core", "extra"],
  "verify_signatures": false,
  "max_concurrent_downloads": 5,
  "download_timeout": 300,
  "keep_versions": 3
}
EOF
    
    log_success "Created configuration file"
    log_info "Location: $CHISEL_CONFIG_FILE"
    log_info "Note: base_dir and symlink_root are automatically configured"
    echo ""
}

# Add to shell configuration
update_shell_config() {
    log_step "Updating shell configuration..."
    
    # Detect shell
    SHELL_NAME=$(basename "$SHELL")
    case "$SHELL_NAME" in
        bash)
            SHELL_RC="$HOME/.bashrc"
            ;;
        zsh)
            SHELL_RC="$HOME/.zshrc"
            ;;
        fish)
            SHELL_RC="$HOME/.config/fish/config.fish"
            ;;
        *)
            log_warning "Unsupported shell: $SHELL_NAME"
            log_info "Manually add the following to your shell configuration:"
            print_shell_config
            echo ""
            return
            ;;
    esac
    
    # Check if already configured
    if grep -q "CHISEL_USER_BASE_DIR" "$SHELL_RC" 2>/dev/null; then
        log_warning "Shell already configured for chisel"
        log_info "Skipping shell configuration update"
        echo ""
        return
    fi
    
    # Add configuration to shell rc
    cat >> "$SHELL_RC" << EOF

# Chisel user-level configuration
export CHISEL_USER_BASE_DIR="$CHISEL_DATA_DIR"
export PATH="\$PATH:\$HOME/.local/bin"
export LD_LIBRARY_PATH="\$HOME/.local/lib:\$LD_LIBRARY_PATH"
EOF
    
    log_success "Updated $SHELL_RC"
    log_info "Added CHISEL_USER_BASE_DIR, PATH, and LD_LIBRARY_PATH"
    echo ""
}

# Print shell configuration snippet
print_shell_config() {
    echo ""
    echo "Add the following to your shell configuration file (~/.bashrc, ~/.zshrc, etc.):"
    echo ""
    echo "# Chisel user-level configuration"
    echo "export CHISEL_USER_BASE_DIR=\"$CHISEL_DATA_DIR\""
    echo "export PATH=\"\$PATH:\$HOME/.local/bin\""
    echo "export LD_LIBRARY_PATH=\"\$HOME/.local/lib:\$LD_LIBRARY_PATH\""
    echo ""
}

# Main execution
main() {
    echo "================================================================================"
    echo "Chisel User-Level Setup"
    echo "================================================================================"
    echo ""
    
    if [[ "$SHELL_CONFIG_ONLY" == true ]]; then
        update_shell_config
        return
    fi
    
    check_prerequisites
    create_directories
    create_config
    update_shell_config
    
    # Print summary
    echo "================================================================================"
    echo "Setup Complete!"
    echo "================================================================================"
    echo ""
    log_info "User-level configuration:"
    echo "  Config:   $CHISEL_CONFIG_FILE"
    echo "  Data:     $CHISEL_DATA_DIR"
    echo "  Symlinks: $CHISEL_BIN_DIR"
    echo ""
    log_info "Next steps:"
    echo "  1. Reload your shell: source $SHELL_RC"
    echo "  2. Install packages: chisel install <package>"
    echo "  3. List packages:    chisel list"
    echo "  4. Remove packages:  chisel remove <package>"
    echo ""
    log_info "To use chisel:"
    echo "  chisel --help"
    echo ""
}

main "$@"
