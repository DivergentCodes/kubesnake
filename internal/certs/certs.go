// Package certs provides embedded CA certificates for TLS connections.
// This allows the binary to be fully portable without relying on
// the host filesystem for certificate bundles.
package certs

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
)

//go:embed ca-certificates.crt
var caCertsPEM []byte

// RootCAs returns a certificate pool containing the embedded CA certificates.
func RootCAs() (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertsPEM) {
		return nil, fmt.Errorf("failed to parse embedded CA certificates")
	}
	return pool, nil
}

// TLSConfig returns a TLS configuration using the embedded CA certificates.
func TLSConfig() (*tls.Config, error) {
	pool, err := RootCAs()
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		RootCAs: pool,
	}, nil
}
