# Multi-Architecture Docker Build Justfile
# Recipes for building, testing, and pushing modware-stock images
# Automatically uses apple/container on macOS, docker buildx on Linux

# Variables
name        := "modware-stock"
namespace   := "dictybase"
dockerfile  := "build/package/Dockerfile.multiarch"
platforms   := "linux/amd64,linux/arm64"
github_user := "sba964"

image      := namespace + "/" + name
ghcr_image := "ghcr.io/" + image

# OS detection — routes to apple/container on macOS, docker on Linux
on_macos := if os() == "macos" { "true" } else { "false" }

# Default recipe - show help
default:
    @just --list

# Build multi-architecture image (no push)
build-multiarch:
    @echo "Building {{image}} for {{platforms}}..."
    docker buildx build \
        --platform {{platforms}} \
        -t {{image}}:multiarch \
        -f {{dockerfile}} \
        .
    @echo "✓ Multi-architecture build completed"

# Build and load AMD64 image locally for testing
build-amd64:
    @echo "Building {{image}} for linux/amd64 (local)..."
    docker buildx build \
        --platform linux/amd64 \
        --load \
        -t {{image}}:amd64 \
        -f {{dockerfile}} \
        .
    @echo "✓ AMD64 image built and loaded"

# Build and load ARM64 image locally for testing (requires emulation)
build-arm64:
    @echo "Building {{image}} for linux/arm64 (emulated)..."
    docker buildx build \
        --platform linux/arm64 \
        --load \
        -t {{image}}:arm64 \
        -f {{dockerfile}} \
        .
    @echo "✓ ARM64 image built and loaded"

# Test the locally loaded AMD64 image
test-amd64: build-amd64
    @echo "Testing {{image}}:amd64..."
    docker run --rm {{image}}:amd64 --help
    @echo "✓ AMD64 image test passed"

# Test the ARM64 image (via emulation)
test-arm64: build-arm64
    @echo "Testing {{image}}:arm64..."
    docker run --rm --platform linux/arm64 {{image}}:arm64 --help
    @echo "✓ ARM64 image test passed"

# Setup Docker buildx if not available
setup-buildx:
    @echo "Setting up Docker buildx..."
    @docker buildx create --use || docker buildx use default
    @docker buildx ls

# Show available platforms
show-platforms:
    @docker buildx ls

# Clean up images
clean:
    docker rmi {{image}}:multiarch {{image}}:amd64 {{image}}:arm64 2>/dev/null || true
    @echo "✓ Cleanup completed"

# Build for GitHub Container Registry
build-ghcr tag="latest":
    @echo "Building {{ghcr_image}}:{{tag}} for {{platforms}}..."
    docker buildx build \
        --platform {{platforms}} \
        -t {{ghcr_image}}:{{tag}} \
        -f {{dockerfile}} \
        .
    @echo "✓ GitHub Container Registry build completed"

# Push to GitHub Container Registry
push-ghcr tag="latest":
    @echo "Logging into GitHub Container Registry..."
    echo $GITHUB_REGISTRY_TOKEN | docker login ghcr.io -u {{github_user}} --password-stdin
    @echo "Building and pushing {{ghcr_image}}:{{tag}} for {{platforms}}..."
    docker buildx build \
        --platform {{platforms}} \
        -t {{ghcr_image}}:{{tag}} \
        -f {{dockerfile}} \
        --push \
        .
    @echo "✓ Successfully pushed to GitHub Container Registry"
