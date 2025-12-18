#!/bin/sh
set -e

BINARY_NAME="kubesnake"
VERSION="${VERSION:-dev}"

# POSIX-safe way to get script directory
SCRIPTS_PATH=$(cd "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(dirname "$SCRIPTS_PATH")

# Embedded CA certificates
CERT_REL_PATH="internal/certs"
CERT_PATH="$PROJECT_ROOT/$CERT_REL_PATH"

# Platforms to cross-compile for (GOOS/GOARCH)
PLATFORMS="linux/amd64 linux/arm64"

# Copy embedded CA certificates to the project root
copy_certs() {
    mkdir -p "$CERT_PATH"

    if [ -f /etc/ssl/cert.pem ]; then
        cp /etc/ssl/cert.pem "$CERT_PATH/ca-certificates.crt"
    elif [ -f /etc/ssl/certs/ca-certificates.crt ]; then
        cp /etc/ssl/certs/ca-certificates.crt "$CERT_PATH/ca-certificates.crt"
    elif [ -f /etc/pki/tls/certs/ca-bundle.crt ]; then
        cp /etc/pki/tls/certs/ca-bundle.crt "$CERT_PATH/ca-certificates.crt"
    else
        echo "Could not find system CA certificates" >&2
        exit 1
    fi

    echo "Copied CA certificates to $CERT_REL_PATH/"
}

# Build a single binary for a specific platform.
# Usage: build [goos] [goarch] [outfile]
# Defaults: goos=linux, goarch=amd64, outfile=$PROJECT_ROOT/dist/$BINARY_NAME
build() {
    goos="${1:-linux}"
    goarch="${2:-amd64}"
    outfile="${3:-$PROJECT_ROOT/dist/$BINARY_NAME}"
    mkdir -p "$(dirname "$outfile")"

    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
        -ldflags "-X main.Version=$VERSION" \
        -o "$outfile" \
        ./cmd/kubesnake \
        || {
            echo "Failed to build $outfile" >&2
            exit 1
        }

    echo "Built $outfile ($goos/$goarch)"
}

# Build binaries for all platforms
build_all() {
    for platform in $PLATFORMS; do
        goos="${platform%/*}"
        goarch="${platform#*/}"
        outfile="$PROJECT_ROOT/dist/${BINARY_NAME}-${goos}-${goarch}"
        build "$goos" "$goarch" "$outfile"
    done
}

main() {
    copy_certs

    case "${1:-}" in
        all)
            build_all
            ;;
        *)
            build
            ;;
    esac
}
# Main
main "$@"
