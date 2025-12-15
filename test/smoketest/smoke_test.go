//go:build e2e

package smoke

import (
	"fmt"
	"testing"
	"time"

	"github.com/DivergentCodes/kubesnake/test/common"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// TestSmoke executes the high-level smoke validation against the provisioned cluster.
func TestSmoke(t *testing.T) {

	// Pre-launch
	testKubesnakeBinaryAvailable(t)
	testKubesnakeBinaryExecutes(t)
	testBeaconFilesExist(t)

	// Launch kubesnake and verify beacon callback.
	testKubesnakeBeaconsBack(t)
}

// testBeaconFilesExist verifies that the beacon files exist in each of the target pod's containers.
func testKubesnakeBinaryAvailable(t *testing.T) {
	builder := features.New("kubesnake binary exists")

	checks := []struct {
		namespace, pod, container, file string
	}{
		{"default", "target-pod", "app", "/tmp/kubesnake"},
		{"default", "target-pod", "sidecar", "/tmp/kubesnake"},
	}

	for _, c := range checks {
		builder.Assess(
			fmt.Sprintf("%s:%s:%s", c.namespace, c.pod, c.container),
			common.VerifyContainerHasFile(c.namespace, c.pod, c.container, c.file),
		)
	}

	testenv.Test(t, builder.Feature())
}

// testKubesnakeBinaryExecutes verifies that the kubesnake binary executes in each of the target pod's containers.
func testKubesnakeBinaryExecutes(t *testing.T) {
	builder := features.New("kubesnake binary executes")
	checks := []struct {
		namespace, pod, container, binaryPath string
	}{
		{"default", "target-pod", "app", "/tmp/kubesnake"},
		{"default", "target-pod", "sidecar", "/tmp/kubesnake"},
	}

	for _, c := range checks {
		builder.Assess(
			fmt.Sprintf("%s:%s:%s", c.namespace, c.pod, c.container),
			common.VerifyKubesnakeRuns(c.namespace, c.pod, c.container, c.binaryPath),
		)
	}

	testenv.Test(t, builder.Feature())
}

// testBeaconFilesExist verifies that the beacon files exist in each of the target pod's containers.
func testBeaconFilesExist(t *testing.T) {
	checks := []struct {
		namespace, pod, container, file string
	}{
		{"default", "target-pod", "app", "/var/run/kubesnake/beacon.json"},
		{"default", "target-pod", "sidecar", "/var/run/kubesnake/beacon.json"},
	}

	builder := features.New("beacon file exists")
	for _, c := range checks {
		builder.Assess(
			fmt.Sprintf("%s:%s:%s", c.namespace, c.pod, c.container),
			common.VerifyContainerHasFile(c.namespace, c.pod, c.container, c.file),
		)
	}

	testenv.Test(t, builder.Feature())
}

// testKubesnakeBeaconsBack verifies that kubesnake sends beacon callbacks to the test receiver.
func testKubesnakeBeaconsBack(t *testing.T) {
	checks := []struct {
		namespace, pod, container, binaryPath string
	}{
		{"default", "target-pod", "app", "/tmp/kubesnake"},
		{"default", "target-pod", "sidecar", "/tmp/kubesnake"},
	}

	builder := features.New("kubesnake beacons back")
	for _, c := range checks {
		builder.Assess(
			fmt.Sprintf("%s:%s:%s", c.namespace, c.pod, c.container),
			common.VerifyKubesnakeBeaconsBack(beaconReceiverSvc, c.namespace, c.pod, c.container, c.binaryPath, 30*time.Second),
		)
	}

	testenv.Test(t, builder.Feature())
}
