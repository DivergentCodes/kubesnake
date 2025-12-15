//go:build e2e

package smoke

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DivergentCodes/kubesnake/test/common"
	"github.com/google/uuid"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	testenv           env.Environment
	beaconReceiverSvc *common.BeaconReceiverService
)

// TestMain is the entry point for the smoke test scenario.
func TestMain(m *testing.M) {
	// Start the beacon receiver service before cluster setup.
	// This must be running before kubesnake executes (e2e mode POSTs to this).
	var err error
	beaconReceiverSvc, err = common.StartBeaconReceiverService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start beacon receiver: %v\n", err)
		os.Exit(1)
	}
	defer beaconReceiverSvc.Shutdown(context.Background())

	// Generate a unique cluster name for the test environment.
	clusterName := fmt.Sprintf("kubesnake-smoketest-%s", uuid.NewString()[:6])

	// List local images to preload into the cluster.
	preloadImages := []string{
		common.LocalKubesnakeImage,
	}

	// Create a new test environment configuration.
	cfg := envconf.New()
	testenv = env.NewWithConfig(cfg)
	manifestsDir := common.ResolveManifestsDir()

	// Prepare the cluster for the test environment.
	common.PrepareCluster(testenv, clusterName, manifestsDir, preloadImages)
	testenv.Setup(
		common.WaitForPodReadyFunc("default", "target-pod", 2*time.Minute),
		common.CopyKubesnakeBinaryFunc("default", "target-pod", "app", "/tmp/kubesnake"),
		common.CopyKubesnakeBinaryFunc("default", "target-pod", "sidecar", "/tmp/kubesnake"),
	)

	// Run Test* functions and exit.
	os.Exit(testenv.Run(m))
}
