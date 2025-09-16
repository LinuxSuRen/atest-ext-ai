#!/bin/bash
# Uninstallation script for atest-ext-ai plugin
set -euo pipefail

# Configuration
PLUGIN_NAME="atest-store-ai"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/atest-ai"
LOG_DIR="/var/log/atest-ai"
USER="atest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*"
    exit 1
}

info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root (use sudo)"
    fi
}

# Confirmation prompt
confirm_uninstall() {
    local force=${1:-false}

    if [[ "$force" != "true" ]]; then
        echo ""
        warn "This will remove the atest-ext-ai plugin and all its configuration files."
        warn "This action cannot be undone."
        echo ""
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log "Uninstallation cancelled."
            exit 0
        fi
    fi
}

# Stop and disable systemd service
stop_service() {
    if [[ -f "/etc/systemd/system/${PLUGIN_NAME}.service" ]]; then
        log "Stopping systemd service..."

        # Stop the service
        if systemctl is-active --quiet "${PLUGIN_NAME}.service"; then
            systemctl stop "${PLUGIN_NAME}.service"
            log "Service stopped"
        fi

        # Disable the service
        if systemctl is-enabled --quiet "${PLUGIN_NAME}.service"; then
            systemctl disable "${PLUGIN_NAME}.service"
            log "Service disabled"
        fi

        # Remove service file
        rm -f "/etc/systemd/system/${PLUGIN_NAME}.service"
        systemctl daemon-reload
        log "Service file removed"
    else
        log "No systemd service found"
    fi
}

# Remove binary
remove_binary() {
    log "Removing binary..."

    if [[ -f "$INSTALL_DIR/$PLUGIN_NAME" ]]; then
        rm -f "$INSTALL_DIR/$PLUGIN_NAME"
        log "Binary removed from $INSTALL_DIR"
    else
        log "Binary not found in $INSTALL_DIR"
    fi
}

# Remove configuration
remove_config() {
    local keep_config=${1:-false}

    if [[ "$keep_config" == "true" ]]; then
        log "Keeping configuration files as requested"
        return
    fi

    log "Removing configuration..."

    if [[ -d "$CONFIG_DIR" ]]; then
        rm -rf "$CONFIG_DIR"
        log "Configuration directory removed: $CONFIG_DIR"
    else
        log "Configuration directory not found"
    fi
}

# Remove logs
remove_logs() {
    local keep_logs=${1:-false}

    if [[ "$keep_logs" == "true" ]]; then
        log "Keeping log files as requested"
        return
    fi

    log "Removing logs..."

    if [[ -d "$LOG_DIR" ]]; then
        rm -rf "$LOG_DIR"
        log "Log directory removed: $LOG_DIR"
    else
        log "Log directory not found"
    fi
}

# Remove user and group
remove_user() {
    local keep_user=${1:-false}

    if [[ "$keep_user" == "true" ]]; then
        log "Keeping user account as requested"
        return
    fi

    log "Removing user and group..."

    # Check if user exists and remove
    if id "$USER" >/dev/null 2>&1; then
        userdel "$USER" 2>/dev/null || warn "Failed to remove user $USER (may have active processes)"
        log "User $USER removed"
    else
        log "User $USER not found"
    fi
}

# Clean up runtime files
cleanup_runtime() {
    log "Cleaning up runtime files..."

    # Remove socket files
    local socket_patterns=(
        "/tmp/atest-store-ai*.sock"
        "/var/run/atest-ai*.sock"
    )

    for pattern in "${socket_patterns[@]}"; do
        for file in $pattern; do
            if [[ -S "$file" ]]; then
                rm -f "$file"
                log "Removed socket: $file"
            fi
        done 2>/dev/null
    done

    # Remove PID files
    local pid_files=(
        "/var/run/atest-ai.pid"
        "/tmp/atest-store-ai.pid"
    )

    for pid_file in "${pid_files[@]}"; do
        if [[ -f "$pid_file" ]]; then
            rm -f "$pid_file"
            log "Removed PID file: $pid_file"
        fi
    done
}

# Show post-uninstall information
show_completion() {
    info ""
    info "=========================================="
    info "  atest-ext-ai Plugin Uninstallation Complete"
    info "=========================================="
    info ""
    info "The following items have been removed:"
    info "  ✓ Plugin binary ($INSTALL_DIR/$PLUGIN_NAME)"
    info "  ✓ Systemd service"
    info "  ✓ Configuration files (unless --keep-config used)"
    info "  ✓ Log files (unless --keep-logs used)"
    info "  ✓ User account (unless --keep-user used)"
    info "  ✓ Runtime files"
    info ""

    if [[ "$KEEP_CONFIG" == "true" || "$KEEP_LOGS" == "true" ]]; then
        warn "Some files were preserved:"
        [[ "$KEEP_CONFIG" == "true" ]] && info "  • Configuration: $CONFIG_DIR"
        [[ "$KEEP_LOGS" == "true" ]] && info "  • Logs: $LOG_DIR"
        info ""
        info "To remove these manually:"
        [[ "$KEEP_CONFIG" == "true" ]] && info "  sudo rm -rf $CONFIG_DIR"
        [[ "$KEEP_LOGS" == "true" ]] && info "  sudo rm -rf $LOG_DIR"
        info ""
    fi

    info "Don't forget to update your atest stores.yaml configuration"
    info "to remove the ai-assistant store entry."
    info ""
}

# Print usage information
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --force           Skip confirmation prompt"
    echo "  --keep-config     Keep configuration files"
    echo "  --keep-logs       Keep log files"
    echo "  --keep-user       Keep user account"
    echo "  --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                           # Interactive uninstall"
    echo "  $0 --force                   # Force uninstall without confirmation"
    echo "  $0 --keep-config --keep-logs # Uninstall but keep config and logs"
    echo ""
}

# Main uninstallation function
main() {
    # Parse command line arguments
    local force=false
    local keep_config=false
    local keep_logs=false
    local keep_user=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                force=true
                shift
                ;;
            --keep-config)
                keep_config=true
                shift
                ;;
            --keep-logs)
                keep_logs=true
                shift
                ;;
            --keep-user)
                keep_user=true
                shift
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done

    # Export variables for use in functions
    export KEEP_CONFIG=$keep_config
    export KEEP_LOGS=$keep_logs

    log "Starting atest-ext-ai plugin uninstallation..."

    check_root
    confirm_uninstall "$force"

    stop_service
    remove_binary
    cleanup_runtime
    remove_config "$keep_config"
    remove_logs "$keep_logs"
    remove_user "$keep_user"

    show_completion

    log "Uninstallation completed successfully!"
}

# Run main function
main "$@"