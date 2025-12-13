#!/bin/sh
set -e

BINARY_NAME="kubesnake"

# POSIX-safe way to get script directory
SCRIPTS_PATH=$(cd "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(dirname "$SCRIPTS_PATH")

CERT_REL_PATH="internal/certs"
CERT_PATH="$PROJECT_ROOT/$CERT_REL_PATH"

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

build() {
    outfile="$PROJECT_ROOT/bin/$BINARY_NAME"
    mkdir -p "$(dirname "$outfile")"

    CGO_ENABLED=0 GOOS=linux go build \
        -o "$outfile" \
        ./cmd/kubesnake \
        || {
            echo "Failed to build $outfile" >&2
            exit 1
        }

    echo "Built $outfile"
}

copy_certs
build
