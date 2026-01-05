package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// selfExecutablePath returns the path to the current executable.
func selfExecutablePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	// Reduce surprise: operate on the real file, not a symlink.
	if p, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = p
	}
	return exePath, nil
}

// parseConfigJSON parses the embedded config from the given raw bytes.
func parseConfigJSON(raw []byte) (*Config, error) {
	// Strict parsing: reject unknown fields and trailing tokens.
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()

	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config is not valid JSON: %w", err)
	}

	// Ensure there is no trailing data after the first JSON value.
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("config is not valid JSON: trailing data")
		}
		return nil, fmt.Errorf("config is not valid JSON: %w", err)
	}

	return &cfg, nil
}
