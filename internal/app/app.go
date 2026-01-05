// Package app contains the top level application logic for kubesnake.
package app

import (
	"github.com/DivergentCodes/kubesnake/internal/config"
	"github.com/DivergentCodes/kubesnake/internal/e2emode"
)

// Run is the interface-agnostic entrypoint for running kubesnake.
// CLI/GUI/etc should call this without the app needing to know about any interface details.
func Run() error {
	// Load embedded user configuration (if present).
	cfg, err := config.LoadEmbeddedConfigFromSelf()
	if err != nil {
		return err
	}

	// Check for e2e mode via embedded config.
	if beaconURL := cfg.E2EBeaconURL(); beaconURL != "" {
		return e2emode.RunE2EMode(beaconURL)
	}

	return nil
}
