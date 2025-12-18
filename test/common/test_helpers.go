package common

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// VerifyContainerHasFile returns a features.Func that checks if a container has a given file.
func VerifyContainerHasFile(namespace, pod, container, filename string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		output, err := KubectlExec(ctx, cfg, namespace, pod, container, "test", "-f", filename)
		if err != nil {
			t.Fatalf("%s missing in container %s/%s/%s: %v: %s", filename, namespace, pod, container, err, string(output))
		}

		return ctx
	}
}

// VerifyKubesnakeRuns returns a features.Func that verifies the kubesnake binary executes successfully.
func VerifyKubesnakeRuns(namespace, pod, container, binaryPath string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		output, err := KubectlExec(ctx, cfg, namespace, pod, container, binaryPath)
		if err != nil {
			t.Fatalf("kubesnake failed in %s/%s/%s: %v: %s", namespace, pod, container, err, string(output))
		}

		cliOutputSubstring := "KubeSnake"
		if !strings.Contains(string(output), cliOutputSubstring) {
			t.Fatalf("unexpected output: %q", string(output))
		}

		t.Logf("kubesnake output: %s", strings.TrimSpace(string(output)))
		return ctx
	}
}

// VerifyKubesnakeBeaconsBack returns a features.Func that verifies kubesnake sends a beacon callback.
// It expects a BeaconReceiverService to be running and waits for a beacon from the specified container.
func VerifyKubesnakeBeaconsBack(svc *BeaconReceiverService, namespace, pod, container, binaryPath string, timeout time.Duration) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		// Run kubesnake in the container (it will POST to the beacon URL due to env var).
		output, err := KubectlExec(ctx, cfg, namespace, pod, container, binaryPath)
		if err != nil {
			t.Fatalf("kubesnake failed in %s/%s/%s: %v: %s", namespace, pod, container, err, string(output))
		}
		t.Logf("kubesnake output: %s", strings.TrimSpace(string(output)))

		// Wait for the beacon callback.
		select {
		case beacon := <-svc.Beacons():
			// Validate beacon structure.
			var beaconData map[string]interface{}
			if err := json.Unmarshal(beacon, &beaconData); err != nil {
				t.Fatalf("invalid beacon JSON: %v", err)
			}

			// Verify the beacon is from the expected container.
			if beaconData["container"] != container {
				t.Fatalf("beacon container mismatch: got %v, want %s", beaconData["container"], container)
			}
			if beaconData["pod"] != pod {
				t.Fatalf("beacon pod mismatch: got %v, want %s", beaconData["pod"], pod)
			}
			if beaconData["namespace"] != namespace {
				t.Fatalf("beacon namespace mismatch: got %v, want %s", beaconData["namespace"], namespace)
			}

			t.Logf("received beacon: %s", string(beacon))

		case <-time.After(timeout):
			t.Fatalf("timeout waiting for beacon from %s/%s/%s", namespace, pod, container)
		}

		return ctx
	}
}
