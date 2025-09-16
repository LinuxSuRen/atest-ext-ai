#!/bin/bash
# Installation script for atest-ext-ai plugin
set -euo pipefail

# Configuration
PLUGIN_NAME="atest-store-ai"
GITHUB_REPO="linuxsuren/atest-ext-ai"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/atest-ai"
LOG_DIR="/var/log/atest-ai"
USER="atest"
GROUP="atest"

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

# Detect OS and architecture
detect_system() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        armv7l) ARCH="arm" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    case $OS in
        linux) ;;
        darwin) ;;
        *) error "Unsupported operating system: $OS" ;;
    esac

    log "Detected system: $OS-$ARCH"
}

# Get latest release version
get_latest_version() {
    log "Fetching latest release information..."

    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    if [[ -z "$VERSION" ]]; then
        error "Failed to fetch latest version"
    fi

    log "Latest version: $VERSION"
}

# Download and install binary
install_binary() {
    local binary_name="${PLUGIN_NAME}-${OS}-${ARCH}"
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$VERSION/${binary_name}.tar.gz"
    local temp_dir=$(mktemp -d)

    log "Downloading $PLUGIN_NAME $VERSION..."

    if command -v curl >/dev/null 2>&1; then
        curl -L "$download_url" -o "$temp_dir/${binary_name}.tar.gz"
    elif command -v wget >/dev/null 2>&1; then
        wget "$download_url" -O "$temp_dir/${binary_name}.tar.gz"
    fi

    # Extract and install
    log "Installing binary to $INSTALL_DIR..."
    tar -xzf "$temp_dir/${binary_name}.tar.gz" -C "$temp_dir"
    chmod +x "$temp_dir/$binary_name"
    mv "$temp_dir/$binary_name" "$INSTALL_DIR/$PLUGIN_NAME"

    # Cleanup
    rm -rf "$temp_dir"

    log "Binary installed successfully"
}

# Create user and group
create_user() {
    if ! id "$USER" >/dev/null 2>&1; then
        log "Creating user $USER..."
        useradd -r -s /bin/false -M "$USER"
    else
        log "User $USER already exists"
    fi
}

# Create directories
create_directories() {
    log "Creating directories..."

    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"

    # Set permissions
    chown root:$GROUP "$CONFIG_DIR"
    chmod 755 "$CONFIG_DIR"

    chown $USER:$GROUP "$LOG_DIR"
    chmod 755 "$LOG_DIR"

    log "Directories created"
}

# Install configuration files
install_config() {
    log "Installing configuration files..."

    if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
        cat > "$CONFIG_DIR/config.yaml" << 'EOF'
# atest-ext-ai Plugin Configuration
ai:
  provider: local
  model: codellama
  confidence_threshold: 0.7
  ollama_endpoint: http://localhost:11434

plugin:
  socket_path: /tmp/atest-store-ai.sock
  log_level: info
  metrics_enabled: true
  metrics_port: 9090

logging:
  level: info
  format: json
  output: stdout

security:
  rate_limit:
    enabled: true
    requests_per_minute: 60
EOF

        chown root:$GROUP "$CONFIG_DIR/config.yaml"
        chmod 640 "$CONFIG_DIR/config.yaml"

        log "Default configuration installed"
    else
        log "Configuration file already exists, skipping"
    fi
}

# Install systemd service
install_systemd_service() {
    if [[ ! -d "/etc/systemd/system" ]]; then
        warn "Systemd not detected, skipping service installation"
        return
    fi

    log "Installing systemd service..."

    cat > "/etc/systemd/system/${PLUGIN_NAME}.service" << EOF
[Unit]
Description=atest-ext-ai Plugin
Documentation=https://github.com/$GITHUB_REPO
After=network.target
Wants=network.target

[Service]
Type=simple
User=$USER
Group=$GROUP
ExecStart=$INSTALL_DIR/$PLUGIN_NAME --config $CONFIG_DIR/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$PLUGIN_NAME

# Security
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/tmp $LOG_DIR
PrivateTmp=yes

# Resource limits
LimitNOFILE=65536
MemoryMax=2G
CPUQuota=200%

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload

    log "Systemd service installed"
}

# Install dependencies
install_dependencies() {
    log "Checking dependencies..."

    # Detect package manager
    if command -v apt-get >/dev/null 2>&1; then
        PKG_MANAGER="apt-get"
    elif command -v yum >/dev/null 2>&1; then
        PKG_MANAGER="yum"
    elif command -v dnf >/dev/null 2>&1; then
        PKG_MANAGER="dnf"
    elif command -v pacman >/dev/null 2>&1; then
        PKG_MANAGER="pacman"
    else
        warn "Package manager not detected, skipping dependency installation"
        return
    fi

    # Install curl if not present
    if ! command -v curl >/dev/null 2>&1; then
        log "Installing curl..."
        case $PKG_MANAGER in
            "apt-get")
                apt-get update && apt-get install -y curl
                ;;
            "yum"|"dnf")
                $PKG_MANAGER install -y curl
                ;;
            "pacman")
                pacman -S --noconfirm curl
                ;;
        esac
    fi
}

# Verify installation
verify_installation() {
    log "Verifying installation..."

    # Check binary
    if [[ ! -x "$INSTALL_DIR/$PLUGIN_NAME" ]]; then
        error "Binary not found or not executable"
    fi

    # Check version
    local version_output
    version_output=$("$INSTALL_DIR/$PLUGIN_NAME" --version 2>/dev/null || echo "version command not available")
    info "Installed version: $version_output"

    # Check configuration
    if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
        error "Configuration file not found"
    fi

    # Check systemd service (if available)
    if [[ -d "/etc/systemd/system" ]]; then
        systemctl is-enabled "${PLUGIN_NAME}.service" >/dev/null 2>&1 || warn "Service not enabled"
    fi

    log "Installation verified successfully"
}

# Show post-installation instructions
show_instructions() {
    info ""
    info "=========================================="
    info "  atest-ext-ai Plugin Installation Complete"
    info "=========================================="
    info ""
    info "Next steps:"
    info ""
    info "1. Configure the plugin:"
    info "   sudo nano $CONFIG_DIR/config.yaml"
    info ""
    info "2. Start the service:"
    info "   sudo systemctl enable $PLUGIN_NAME"
    info "   sudo systemctl start $PLUGIN_NAME"
    info ""
    info "3. Check service status:"
    info "   sudo systemctl status $PLUGIN_NAME"
    info ""
    info "4. View logs:"
    info "   sudo journalctl -u $PLUGIN_NAME -f"
    info ""
    info "5. Configure your atest stores.yaml:"
    info "   Add the following to your stores configuration:"
    info ""
    info "   stores:"
    info "     - name: \"ai-assistant\""
    info "     type: \"ai\""
    info "     url: \"unix:///tmp/atest-store-ai.sock\""
    info ""
    info "For more information, see:"
    info "https://github.com/$GITHUB_REPO"
    info ""
    info "Socket path: /tmp/atest-store-ai.sock"
    info "Config path: $CONFIG_DIR/config.yaml"
    info "Log path: $LOG_DIR"
    info ""
}

# Main installation function
main() {
    log "Starting atest-ext-ai plugin installation..."

    check_root
    detect_system
    install_dependencies
    get_latest_version
    install_binary
    create_user
    create_directories
    install_config
    install_systemd_service
    verify_installation
    show_instructions

    log "Installation completed successfully!"
}

# Handle command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [--version VERSION] [--help]"
            echo ""
            echo "Options:"
            echo "  --version VERSION    Install specific version"
            echo "  --help               Show this help message"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Run main function
main "$@"