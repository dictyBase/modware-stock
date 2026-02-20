# Multi-Architecture Docker Build Justfile
# Recipes for building, testing, and pushing modware-stock images
# Uses docker buildx for multi-platform builds on all platforms

# Variables
name        := "modware-stock"
namespace   := "dictybase"
dockerfile  := "build/package/Dockerfile.multiarch"
platforms   := "linux/amd64,linux/arm64"
github_user := "cybersiddhu"

image      := namespace + "/" + name
ghcr_image := "ghcr.io/" + image

# Default recipe - show help
default:
    @just --list

# Build multi-architecture image (no push)
build-multiarch:
    @echo "Building {{image}} for amd64 and arm64..."
    docker buildx build --platform {{platforms}} -t {{image}}:multiarch -f {{dockerfile}} .
    @echo "✓ Multi-architecture build completed"

# Build and load AMD64 image locally for testing
build-amd64:
    @echo "Building {{image}} for linux/amd64 (local)..."
    docker buildx build --platform linux/amd64 --load -t {{image}}:amd64 -f {{dockerfile}} .
    @echo "✓ AMD64 image built and loaded"

# Build and load ARM64 image locally for testing
build-arm64:
    @echo "Building {{image}} for linux/arm64..."
    docker buildx build --platform linux/arm64 --load -t {{image}}:arm64 -f {{dockerfile}} .
    @echo "✓ ARM64 image built and loaded"

# Test the locally loaded AMD64 image
test-amd64: build-amd64
    @echo "Testing {{image}}:amd64..."
    docker run --rm --platform linux/amd64 {{image}}:amd64 --help
    @echo "✓ AMD64 image test passed"

# Test the ARM64 image
test-arm64: build-arm64
    @echo "Testing {{image}}:arm64..."
    docker run --rm --platform linux/arm64 {{image}}:arm64 --help
    @echo "✓ ARM64 image test passed"

# Setup buildx builder
setup-buildx:
    @echo "Setting up docker buildx..."
    docker buildx create --use || docker buildx use default
    docker buildx ls

# Show available platforms
show-platforms:
    docker buildx ls

# Clean up images
clean:
    docker rmi {{image}}:multiarch {{image}}:amd64 {{image}}:arm64 2>/dev/null || true
    @echo "✓ Cleanup completed"

# Build for GitHub Container Registry
build-ghcr tag="latest":
    @echo "Building {{ghcr_image}}:{{tag}} for amd64 and arm64..."
    docker buildx build --platform {{platforms}} -t {{ghcr_image}}:{{tag}} -f {{dockerfile}} .
    @echo "✓ GitHub Container Registry build completed"

# Push to GitHub Container Registry
push-ghcr tag="latest":
    @echo "Logging into GitHub Container Registry..."
    echo $GITHUB_REGISTRY_TOKEN | docker login ghcr.io -u {{github_user}} --password-stdin
    @echo "Building and pushing {{ghcr_image}}:{{tag}}..."
    docker buildx build --platform {{platforms}} -t {{ghcr_image}}:{{tag}} -f {{dockerfile}} . --push
    @echo "✓ Successfully pushed to GitHub Container Registry"
