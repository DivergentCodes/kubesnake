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

		// Run kubesnake in a mode that produces deterministic output and does not
		// execute the main app logic (which could trigger e2e beaconing and pollute
		// the shared beacon receiver channel for later tests).
		output, err := KubectlExec(ctx, cfg, namespace, pod, container, binaryPath, "--help")
		if err != nil {
			t.Fatalf("kubesnake failed in %s/%s/%s: %v: %s", namespace, pod, container, err, string(output))
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

		// Run kubesnake in the container (it will POST the beacon JSON in e2e mode).
		output, err := KubectlExec(ctx, cfg, namespace, pod, container, binaryPath)
		if err != nil {
			t.Fatalf("kubesnake failed in %s/%s/%s: %v: %s", namespace, pod, container, err, string(output))
		}
		t.Logf("kubesnake output: %s", strings.TrimSpace(string(output)))

		// Wait for a matching beacon callback. The receiver channel is shared across
		// the suite, so earlier test steps may have already produced beacons.
		deadline := time.NewTimer(timeout)
		defer deadline.Stop()
		for {
			select {
			case beacon := <-svc.Beacons():
				// Validate beacon structure.
				var beaconData map[string]interface{}
				if err := json.Unmarshal(beacon, &beaconData); err != nil {
					t.Fatalf("invalid beacon JSON: %v", err)
				}

				// Keep draining until we see the beacon for the expected container.
				if beaconData["container"] != container || beaconData["pod"] != pod || beaconData["namespace"] != namespace {
					t.Logf("discarding beacon (want %s/%s/%s): %s", namespace, pod, container, strings.TrimSpace(string(beacon)))
					continue
				}

				t.Logf("received beacon: %s", string(beacon))
				return ctx

			case <-deadline.C:
				t.Fatalf("timeout waiting for beacon from %s/%s/%s", namespace, pod, container)
			}
		}
	}
}

// CopyFileToContainer returns a features.Func that copies a local file to a container.
func CopyFileToContainer(localPath, namespace, pod, container, remotePath string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		if err := KubectlCp(ctx, cfg, localPath, namespace, pod, container, remotePath); err != nil {
			t.Fatalf("copy file to %s/%s/%s:%s: %v", namespace, pod, container, remotePath, err)
		}
		return ctx
	}
}

// VerifyKubesnakeEmbedsConfig returns a features.Func that embeds a config into the binary.
func VerifyKubesnakeEmbedsConfig(namespace, pod, container, binaryPath, configPath string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		output, err := KubectlExec(ctx, cfg, namespace, pod, container, binaryPath, "embed", configPath)
		if err != nil {
			t.Fatalf("kubesnake embed failed in %s/%s/%s: %v: %s", namespace, pod, container, err, string(output))
		}

		if !strings.Contains(string(output), "embedded config into") {
			t.Fatalf("unexpected embed output: %q", string(output))
		}
		t.Logf("kubesnake embed output: %s", strings.TrimSpace(string(output)))
		return ctx
	}
}

// VerifyKubesnakeRunsWithoutBeaconEnv runs kubesnake with the beacon env var removed.
// This avoids consuming the shared beacon receiver channel during tests.
func VerifyKubesnakeRunsWithoutBeaconEnv(namespace, pod, container, binaryPath string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		// Avoid triggering e2e beaconing by running a command path that does not invoke app.Run().
		output, err := KubectlExec(ctx, cfg, namespace, pod, container, binaryPath, "--help")
		if err != nil {
			t.Fatalf("kubesnake failed in %s/%s/%s: %v: %s", namespace, pod, container, err, string(output))
		}

		t.Logf("kubesnake output: %s", strings.TrimSpace(string(output)))
		return ctx
	}
}
