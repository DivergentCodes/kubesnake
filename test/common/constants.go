package common

// Constants for the common package.

const (
	// BeaconReceiverPort is the port the beacon receiver listens on.
	// Pods reach the host via host.k3d.internal (k3d default host alias).
	BeaconReceiverPort = 18080

	// BeaconReceiverPath is the HTTP path for receiving beacons.
	BeaconReceiverPath = "/beacon"
)

var (
	// LocalKubesnakeImage is the local image and tag name for kubesnake.
	LocalKubesnakeImage = "kubesnake:dev"

	// RequiredCommands is the list of commands that must be available on PATH.
	RequiredCommands = []string{
		"k3d",
		"kubectl",
	}

	// defaultManifestsDir is the default relative directory for manifests.
	defaultManifestsDir = "manifests"
)
