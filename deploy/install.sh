#!/bin/bash

# OneDock Installation Script
# This script installs OneDock as a systemd service

set -e

# Configuration
INSTALL_DIR="/opt/onedock"
SERVICE_FILE="/etc/systemd/system/onedock.service"
USER="onedock"
GROUP="onedock"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
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

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

check_dependencies() {
    log_info "Checking dependencies..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running. Please start Docker first."
        exit 1
    fi
    
    # Check if systemd is available
    if ! command -v systemctl &> /dev/null; then
        log_error "systemctl is not available. This script requires systemd."
        exit 1
    fi
    
    log_success "All dependencies are satisfied"
}

create_user() {
    log_info "Creating user and group..."
    
    if ! getent group "$GROUP" &> /dev/null; then
        groupadd -r "$GROUP"
        log_success "Created group: $GROUP"
    else
        log_info "Group $GROUP already exists"
    fi
    
    if ! getent passwd "$USER" &> /dev/null; then
        useradd -r -g "$GROUP" -d "$INSTALL_DIR" -s /bin/false -c "OneDock service user" "$USER"
        log_success "Created user: $USER"
    else
        log_info "User $USER already exists"
    fi
    
    # Add user to docker group for Docker access
    usermod -aG docker "$USER"
    log_success "Added $USER to docker group"
}

create_directories() {
    log_info "Creating installation directory..."
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$INSTALL_DIR/logs"
    
    chown -R "$USER:$GROUP" "$INSTALL_DIR"
    chmod 755 "$INSTALL_DIR"
    
    log_success "Created directory: $INSTALL_DIR"
}

install_binary() {
    log_info "Installing OneDock binary..."
    
    # Check if binary exists in current directory
    if [[ ! -f "onedock" ]]; then
        log_error "OneDock binary not found. Please build it first with: go build -o onedock"
        exit 1
    fi
    
    # Copy binary
    cp onedock "$INSTALL_DIR/"
    chown "$USER:$GROUP" "$INSTALL_DIR/onedock"
    chmod 755 "$INSTALL_DIR/onedock"
    
    # Copy config if exists
    if [[ -f "config.toml" ]]; then
        cp config.toml "$INSTALL_DIR/"
        chown "$USER:$GROUP" "$INSTALL_DIR/config.toml"
        chmod 644 "$INSTALL_DIR/config.toml"
        log_success "Copied config.toml"
    elif [[ -f "config.toml.example" ]]; then
        cp config.toml.example "$INSTALL_DIR/config.toml"
        chown "$USER:$GROUP" "$INSTALL_DIR/config.toml"
        chmod 644 "$INSTALL_DIR/config.toml"
        log_success "Copied config.toml.example as config.toml"
        log_warning "Please review and update $INSTALL_DIR/config.toml"
    else
        log_warning "No config file found. OneDock will use default settings."
    fi
    
    log_success "Binary installed to: $INSTALL_DIR/onedock"
}

install_service() {
    log_info "Installing systemd service..."
    
    # Check if service file exists in deploy directory
    if [[ -f "deploy/onedock.service" ]]; then
        cp deploy/onedock.service "$SERVICE_FILE"
    else
        log_error "Service file not found at deploy/onedock.service"
        exit 1
    fi
    
    # Reload systemd
    systemctl daemon-reload
    
    # Enable service
    systemctl enable onedock.service
    
    log_success "Service installed and enabled"
}

start_service() {
    log_info "Starting OneDock service..."
    
    systemctl start onedock.service
    
    # Check if service started successfully
    if systemctl is-active --quiet onedock.service; then
        log_success "OneDock service is running"
    else
        log_error "Failed to start OneDock service"
        log_info "Check logs with: journalctl -u onedock.service -f"
        exit 1
    fi
}

show_status() {
    log_info "Service status:"
    systemctl status onedock.service --no-pager -l
    
    echo ""
    log_info "Useful commands:"
    echo "  Start service:   sudo systemctl start onedock"
    echo "  Stop service:    sudo systemctl stop onedock"
    echo "  Restart service: sudo systemctl restart onedock"
    echo "  View logs:       journalctl -u onedock -f"
    echo "  Service status:  systemctl status onedock"
    echo ""
    log_info "OneDock should be accessible at: http://localhost:8801"
    log_info "API documentation: http://localhost:8801/swagger/index.html"
}

main() {
    echo "OneDock Installation Script"
    echo "=========================="
    echo ""
    
    check_root
    check_dependencies
    create_user
    create_directories
    install_binary
    install_service
    start_service
    
    echo ""
    log_success "OneDock installation completed successfully!"
    echo ""
    show_status
}

# Run main function
main "$@"