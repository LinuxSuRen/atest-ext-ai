#!/bin/bash
# Deployment script for atest-ext-ai plugin
set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PLUGIN_NAME="atest-ext-ai"

# Default values
ENVIRONMENT=""
VERSION=""
DOCKER_REGISTRY="ghcr.io"
DOCKER_IMAGE="$DOCKER_REGISTRY/linuxsuren/atest-ext-ai"
NAMESPACE="atest-system"
DRY_RUN=false
VERBOSE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

debug() {
    [[ "$VERBOSE" == "true" ]] && echo -e "${CYAN}[DEBUG]${NC} $*"
}

# Print usage information
usage() {
    cat << EOF
Usage: $0 -e ENVIRONMENT [OPTIONS]

Deploy atest-ext-ai plugin to different environments.

Required:
  -e, --environment ENV    Target environment (local, docker, k8s-dev, k8s-prod)

Options:
  -v, --version VERSION    Version to deploy (default: latest git tag)
  -n, --namespace NS       Kubernetes namespace (default: atest-system)
  -r, --registry URL       Docker registry URL (default: ghcr.io)
  -d, --dry-run            Show what would be done without executing
  -V, --verbose            Enable verbose output
  -h, --help               Show this help message

Environments:
  local        Deploy locally using systemd service
  docker       Deploy using Docker Compose
  k8s-dev      Deploy to Kubernetes development environment
  k8s-prod     Deploy to Kubernetes production environment

Examples:
  $0 -e local                    # Deploy latest version locally
  $0 -e docker -v v1.2.3         # Deploy specific version with Docker
  $0 -e k8s-prod -v v1.2.3 -d    # Dry run production Kubernetes deployment
  $0 -e k8s-dev --verbose        # Deploy to dev with verbose output

Prerequisites:
  - For local: systemd, sudo privileges
  - For docker: Docker, Docker Compose
  - For k8s-*: kubectl, proper cluster access

EOF
}

# Check prerequisites for deployment environment
check_prerequisites() {
    local env="$1"

    debug "Checking prerequisites for environment: $env"

    case "$env" in
        "local")
            command -v systemctl >/dev/null 2>&1 || error "systemctl not found (systemd required)"
            [[ $EUID -eq 0 ]] || error "Root privileges required for local deployment"
            ;;
        "docker")
            command -v docker >/dev/null 2>&1 || error "Docker not found"
            command -v docker-compose >/dev/null 2>&1 || error "Docker Compose not found"
            ;;
        "k8s-dev"|"k8s-prod")
            command -v kubectl >/dev/null 2>&1 || error "kubectl not found"
            kubectl cluster-info >/dev/null 2>&1 || error "No Kubernetes cluster access"
            ;;
        *)
            error "Unknown environment: $env"
            ;;
    esac

    log "Prerequisites check passed for $env environment"
}

# Get version to deploy
get_version() {
    if [[ -n "$VERSION" ]]; then
        debug "Using specified version: $VERSION"
        return
    fi

    # Try to get version from git
    if command -v git >/dev/null 2>&1 && [[ -d "$PROJECT_DIR/.git" ]]; then
        VERSION=$(cd "$PROJECT_DIR" && git describe --tags --exact-match 2>/dev/null || git describe --tags --always --dirty)
        debug "Version from git: $VERSION"
    else
        VERSION="latest"
        warn "Could not determine version, using: $VERSION"
    fi
}

# Deploy locally using systemd
deploy_local() {
    log "Deploying to local environment..."

    local binary_path="/usr/local/bin/$PLUGIN_NAME"
    local config_path="/etc/atest-ai/config.yaml"
    local service_name="${PLUGIN_NAME}.service"

    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would deploy locally with systemd"
        info "  - Binary: $binary_path"
        info "  - Config: $config_path"
        info "  - Service: $service_name"
        return
    fi

    # Stop service if running
    if systemctl is-active --quiet "$service_name"; then
        log "Stopping existing service..."
        systemctl stop "$service_name"
    fi

    # Build and install binary
    log "Building and installing binary..."
    cd "$PROJECT_DIR"
    make build
    cp "bin/$PLUGIN_NAME" "$binary_path"
    chmod +x "$binary_path"

    # Update configuration if needed
    if [[ ! -f "$config_path" ]]; then
        log "Installing default configuration..."
        mkdir -p "$(dirname "$config_path")"
        cp "config/production.yaml" "$config_path"
    fi

    # Start service
    log "Starting service..."
    systemctl daemon-reload
    systemctl enable "$service_name"
    systemctl start "$service_name"

    # Wait for service to be ready
    log "Waiting for service to be ready..."
    sleep 5

    if systemctl is-active --quiet "$service_name"; then
        log "Local deployment completed successfully"
        systemctl status "$service_name" --no-pager
    else
        error "Service failed to start"
    fi
}

# Deploy using Docker Compose
deploy_docker() {
    log "Deploying to Docker environment..."

    local compose_file="$PROJECT_DIR/docker-compose.yml"
    local env_file="$PROJECT_DIR/.env"

    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would deploy using Docker Compose"
        info "  - Compose file: $compose_file"
        info "  - Environment file: $env_file"
        info "  - Image: $DOCKER_IMAGE:$VERSION"
        return
    fi

    cd "$PROJECT_DIR"

    # Create .env file with deployment settings
    log "Creating environment configuration..."
    cat > "$env_file" << EOF
# Auto-generated deployment configuration
AI_PROVIDER=local
AI_MODEL=codellama
OLLAMA_ENDPOINT=http://ollama:11434
LOG_LEVEL=info
VERSION=$VERSION
DOCKER_IMAGE=$DOCKER_IMAGE
EOF

    # Build image if VERSION is 'latest' or contains 'dev'
    if [[ "$VERSION" == "latest" || "$VERSION" == *"dev"* || "$VERSION" == *"dirty"* ]]; then
        log "Building Docker image..."
        docker-compose build
    fi

    # Deploy services
    log "Starting services..."
    docker-compose up -d

    # Wait for services to be ready
    log "Waiting for services to be ready..."
    sleep 30

    # Check service health
    if docker-compose ps | grep -q "Up"; then
        log "Docker deployment completed successfully"
        docker-compose ps
    else
        error "One or more services failed to start"
    fi
}

# Deploy to Kubernetes
deploy_kubernetes() {
    local env_type="$1"
    log "Deploying to Kubernetes ($env_type environment)..."

    local k8s_dir="$PROJECT_DIR/k8s"

    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would deploy to Kubernetes"
        info "  - Namespace: $NAMESPACE"
        info "  - Image: $DOCKER_IMAGE:$VERSION"
        info "  - Environment: $env_type"
        return
    fi

    # Create namespace
    log "Creating namespace $NAMESPACE..."
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    # Apply base resources
    log "Applying Kubernetes manifests..."
    kubectl apply -f "$k8s_dir/configmap.yaml" -n "$NAMESPACE"
    kubectl apply -f "$k8s_dir/secret.yaml" -n "$NAMESPACE"
    kubectl apply -f "$k8s_dir/rbac.yaml" -n "$NAMESPACE"
    kubectl apply -f "$k8s_dir/pvc.yaml" -n "$NAMESPACE"

    # Update image version in deployment
    log "Updating deployment image to $DOCKER_IMAGE:$VERSION..."
    kubectl patch deployment atest-ai-plugin \
        -p '{"spec":{"template":{"spec":{"containers":[{"name":"atest-ai-plugin","image":"'$DOCKER_IMAGE:$VERSION'"}]}}}}' \
        -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    # Apply deployment
    kubectl apply -f "$k8s_dir/deployment.yaml" -n "$NAMESPACE"
    kubectl apply -f "$k8s_dir/service.yaml" -n "$NAMESPACE"

    # Apply environment-specific resources
    if [[ "$env_type" == "k8s-prod" ]]; then
        log "Applying production-specific resources..."
        kubectl apply -f "$k8s_dir/hpa.yaml" -n "$NAMESPACE"
        kubectl apply -f "$k8s_dir/ingress.yaml" -n "$NAMESPACE"
    fi

    # Wait for deployment to be ready
    log "Waiting for deployment to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/atest-ai-plugin -n "$NAMESPACE"

    # Verify deployment
    log "Verifying deployment..."
    kubectl get pods -l app.kubernetes.io/name=atest-ext-ai -n "$NAMESPACE"

    # Show service information
    kubectl get services -n "$NAMESPACE"

    log "Kubernetes deployment completed successfully"
}

# Verify deployment
verify_deployment() {
    local env="$1"
    log "Verifying deployment for $env environment..."

    case "$env" in
        "local")
            # Check systemd service
            if systemctl is-active --quiet "${PLUGIN_NAME}.service"; then
                log "✓ Service is running"
            else
                error "✗ Service is not running"
            fi

            # Check socket
            if [[ -S "/tmp/atest-ext-ai.sock" ]]; then
                log "✓ Unix socket is available"
            else
                warn "✗ Unix socket not found"
            fi
            ;;
        "docker")
            # Check Docker containers
            if docker-compose ps | grep -q "Up"; then
                log "✓ Docker containers are running"
            else
                error "✗ Docker containers are not running properly"
            fi
            ;;
        "k8s-dev"|"k8s-prod")
            # Check Kubernetes pods
            local pod_count
            pod_count=$(kubectl get pods -l app.kubernetes.io/name=atest-ext-ai -n "$NAMESPACE" -o jsonpath='{.items[*].status.phase}' | grep -c "Running" || echo "0")

            if [[ "$pod_count" -gt 0 ]]; then
                log "✓ $pod_count pod(s) are running"
            else
                error "✗ No pods are running"
            fi
            ;;
    esac

    # Test health endpoint if available
    if command -v curl >/dev/null 2>&1; then
        case "$env" in
            "local"|"docker")
                if curl -f http://localhost:9090/health >/dev/null 2>&1; then
                    log "✓ Health check passed"
                else
                    warn "✗ Health check failed (this may be expected if not exposed)"
                fi
                ;;
        esac
    fi

    log "Deployment verification completed"
}

# Rollback deployment
rollback() {
    local env="$1"
    warn "Rolling back deployment for $env environment..."

    case "$env" in
        "local")
            systemctl stop "${PLUGIN_NAME}.service" || warn "Failed to stop service"
            ;;
        "docker")
            docker-compose down || warn "Failed to stop Docker services"
            ;;
        "k8s-dev"|"k8s-prod")
            kubectl rollout undo deployment/atest-ai-plugin -n "$NAMESPACE" || warn "Failed to rollback Kubernetes deployment"
            ;;
    esac

    warn "Rollback initiated"
}

# Handle cleanup on exit
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]] && [[ "$DRY_RUN" != "true" ]]; then
        error "Deployment failed, consider running rollback"
    fi
}

trap cleanup EXIT

# Main deployment function
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -r|--registry)
                DOCKER_REGISTRY="$2"
                DOCKER_IMAGE="$DOCKER_REGISTRY/linuxsuren/atest-ext-ai"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -V|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done

    # Validate required parameters
    if [[ -z "$ENVIRONMENT" ]]; then
        error "Environment is required. Use -e|--environment to specify."
    fi

    log "Starting deployment to $ENVIRONMENT environment..."
    [[ "$DRY_RUN" == "true" ]] && warn "DRY RUN MODE - No actual changes will be made"

    get_version
    log "Deploying version: $VERSION"

    check_prerequisites "$ENVIRONMENT"

    # Deploy based on environment
    case "$ENVIRONMENT" in
        "local")
            deploy_local
            ;;
        "docker")
            deploy_docker
            ;;
        "k8s-dev")
            deploy_kubernetes "k8s-dev"
            ;;
        "k8s-prod")
            deploy_kubernetes "k8s-prod"
            ;;
        *)
            error "Unsupported environment: $ENVIRONMENT"
            ;;
    esac

    # Verify deployment (skip in dry run mode)
    if [[ "$DRY_RUN" != "true" ]]; then
        sleep 5  # Give services time to settle
        verify_deployment "$ENVIRONMENT"
    fi

    log "Deployment completed successfully!"

    # Show next steps
    case "$ENVIRONMENT" in
        "local")
            info "Next steps:"
            info "  - Configure plugin: sudo nano /etc/atest-ai/config.yaml"
            info "  - View logs: sudo journalctl -u ${PLUGIN_NAME}.service -f"
            info "  - Check status: sudo systemctl status ${PLUGIN_NAME}.service"
            ;;
        "docker")
            info "Next steps:"
            info "  - View logs: docker-compose logs -f atest-ai-plugin"
            info "  - Check services: docker-compose ps"
            info "  - Access Ollama: http://localhost:11434"
            ;;
        "k8s-dev"|"k8s-prod")
            info "Next steps:"
            info "  - View pods: kubectl get pods -n $NAMESPACE"
            info "  - View logs: kubectl logs -l app.kubernetes.io/name=atest-ext-ai -n $NAMESPACE -f"
            info "  - Check services: kubectl get services -n $NAMESPACE"
            ;;
    esac
}

# Run main function
main "$@"