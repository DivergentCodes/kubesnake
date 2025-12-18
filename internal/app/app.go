// Package app contains the top level application logic for kubesnake.
package app

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

const (
	// E2EBeaconURLEnv is the environment variable that enables e2e mode.
	// When set, kubesnake reads the beacon file and POSTs it to this URL.
	E2EBeaconURLEnv = "KUBESNAKE_E2E_BEACON_URL"

	// E2EBeaconPath is the canonical path to the beacon file in e2e mode.
	E2EBeaconPath = "/var/run/kubesnake/beacon.json"
)

// Run is the main entry point for the kubesnake application.
func Run() error {
	fmt.Println("KubeSnake ( https://github.com/DivergentCodes/kubesnake )")

	// Check for e2e mode.
	if beaconURL := os.Getenv(E2EBeaconURLEnv); beaconURL != "" {
		return runE2EMode(beaconURL)
	}

	return nil
}

// runE2EMode reads the beacon file and POSTs it to the given URL.
func runE2EMode(beaconURL string) error {
	fmt.Printf("e2e mode: sending beacon to %s\n", beaconURL)

	// Read the beacon file.
	beacon, err := os.ReadFile(E2EBeaconPath)
	if err != nil {
		return fmt.Errorf("read beacon file: %w", err)
	}

	// POST the beacon to the URL.
	resp, err := http.Post(beaconURL, "application/json", bytes.NewReader(beacon))
	if err != nil {
		return fmt.Errorf("post beacon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("beacon POST failed: %s", resp.Status)
	}

	fmt.Println("e2e mode: beacon sent successfully")
	return nil
}
