// Package http provides HTTP client utilities with embedded TLS certificates.
package http

import (
	"net/http"

	"github.com/DivergentCodes/kubesnake/internal/certs"
)

// Client wraps http.Client with embedded CA certificates for TLS.
type Client struct {
	*http.Client
}

// NewClient creates an HTTP client using embedded CA certificates.
func NewClient() (*Client, error) {
	tlsConfig, err := certs.TLSConfig()
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
	}, nil
}
