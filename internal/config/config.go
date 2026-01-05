// Package config provides configuration schema, loading, and embedding for kubesnake.
//
// kubesnake is designed to remain a single portable binary that can carry its configuration
// as it propagates through a cluster. This package defines the config schema and provides
// helpers for loading/embedding config into the current executable.
package config

import "strings"

// Config is the top-level configuration schema.
// Keep this minimal and additive (backwards compatible) over time.
type Config struct {
	E2E *E2EConfig `json:"e2e"`
}

type E2EConfig struct {
	BeaconURL string `json:"beaconUrl"`
}

// E2EBeaconURL returns the configured e2e beacon URL, if present.
func (c *Config) E2EBeaconURL() string {
	if c == nil || c.E2E == nil {
		return ""
	}
	return strings.TrimSpace(c.E2E.BeaconURL)
}
