package e2emode

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	// e2eBeaconPath is the canonical path to the beacon file in e2e mode.
	e2eBeaconPath = "/var/run/kubesnake/beacon.json"

	// e2ePostTimeout bounds the total time spent beaconing back in e2e mode
	// (DNS + connect + request + response headers/body).
	e2ePostTimeout = 10 * time.Second
)

// RunE2EMode reads the beacon file and POSTs it to the given URL.
func RunE2EMode(beaconURL string) error {
	// Read the beacon file.
	beacon, err := os.ReadFile(e2eBeaconPath)
	if err != nil {
		return fmt.Errorf("read beacon file: %w", err)
	}

	// POST the beacon to the URL (bounded by a timeout to avoid hanging).
	client := &http.Client{Timeout: e2ePostTimeout}
	req, err := http.NewRequest(http.MethodPost, beaconURL, bytes.NewReader(beacon))
	if err != nil {
		return fmt.Errorf("create beacon request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post beacon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("beacon POST failed: %s", resp.Status)
	}

	return nil
}
