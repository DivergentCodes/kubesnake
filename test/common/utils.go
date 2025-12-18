// Package common provides shared utilities for e2e test scenarios.
package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/k3d"
)

// CommandAvailable reports whether the binary is on PATH.
func CommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// CheckMissingCommands returns the subset of binaries that are not present on PATH.
func CheckMissingCommands(names ...string) []string {
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if !CommandAvailable(name) {
			missing = append(missing, name)
		}
	}

	return missing
}

// ContainerNames extracts the names from the provided container definitions.
func ContainerNames(containers []corev1.Container) []string {
	names := make([]string, 0, len(containers))
	for _, c := range containers {
		names = append(names, c.Name)
	}

	return names
}

// ConfigureKubectlContext returns an env.Func that sets the kube context for the given cluster.
func ConfigureKubectlContext(clusterName string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		provider, ok := envfuncs.GetClusterFromContext(ctx, clusterName)
		if !ok {
			return ctx, fmt.Errorf("configure kubectl: cluster %q missing from context", clusterName)
		}
		cfg.WithKubeContext(provider.GetKubectlContext())
		return ctx, nil
	}
}

// KubectlArgs builds a kubectl command argument list using the test environment configuration.
func KubectlArgs(cfg *envconf.Config, args ...string) []string {
	baseArgs := []string{"--kubeconfig", cfg.KubeconfigFile()}
	if ctxName := cfg.KubeContext(); ctxName != "" {
		baseArgs = append(baseArgs, "--context", ctxName)
	}

	return append(baseArgs, args...)
}

// KubectlExec runs a command in a container and returns the output.
func KubectlExec(ctx context.Context, cfg *envconf.Config, namespace, pod, container string, command ...string) ([]byte, error) {
	args := KubectlArgs(cfg, "exec", "-n", namespace, pod, "-c", container, "--")
	args = append(args, command...)
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	return cmd.CombinedOutput()
}

// KubectlApply applies manifests from the given directory.
func KubectlApply(ctx context.Context, cfg *envconf.Config, manifestsDir string) ([]byte, error) {
	args := KubectlArgs(cfg, "apply", "-f", manifestsDir)
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	return cmd.CombinedOutput()
}

// ApplyClusterManifestsFunc returns an env.Func that applies manifests from the given directory.
func ApplyClusterManifestsFunc(manifestsDir string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		output, err := KubectlApply(ctx, cfg, manifestsDir)
		if err != nil {
			return ctx, fmt.Errorf("apply manifests: %w: %s", err, string(output))
		}
		return ctx, nil
	}
}

// ResolveManifestsDir returns the absolute path to a manifests directory relative to the caller's source file.
func ResolveManifestsDir() string {
	_, thisFile, _, ok := runtime.Caller(1)
	if !ok {
		return defaultManifestsDir
	}

	return filepath.Join(filepath.Dir(thisFile), defaultManifestsDir)
}

// WaitForPodReady waits for a pod to become ready within the given timeout.
func WaitForPodReady(ctx context.Context, cfg *envconf.Config, namespace, podName string, timeout time.Duration) error {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: podName}}
	return wait.For(conditions.New(cfg.Client().Resources()).PodReady(pod), wait.WithTimeout(timeout))
}

// WaitForPodReadyFunc returns an env.Func that waits for a pod to become ready.
func WaitForPodReadyFunc(namespace, podName string, timeout time.Duration) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		if err := WaitForPodReady(ctx, cfg, namespace, podName, timeout); err != nil {
			return ctx, fmt.Errorf("wait for pod %s/%s: %w", namespace, podName, err)
		}
		return ctx, nil
	}
}

// loadImagesToCluster returns an env.Func that loads multiple images into the cluster in a single command.
func loadImagesToCluster(clusterName string, images []string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		args := append([]string{"image", "import", "-c", clusterName}, images...)
		cmd := exec.CommandContext(ctx, "k3d", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return ctx, fmt.Errorf("load images to cluster: %w: %s", err, string(output))
		}
		return ctx, nil
	}
}

// GetNodeArchitecture returns the architecture of the first node in the cluster.
func GetNodeArchitecture(ctx context.Context, cfg *envconf.Config) (string, error) {
	args := KubectlArgs(cfg, "get", "nodes", "-o", "jsonpath={.items[0].status.nodeInfo.architecture}")
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("get node architecture: %w: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// ResolveBinaryPath returns the path to the kubesnake binary for the given OS/arch.
func ResolveBinaryPath(goos, goarch string) string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	return filepath.Join(projectRoot, "dist", fmt.Sprintf("kubesnake-%s-%s", goos, goarch))
}

// KubectlCp copies a local file to a container. The target container must have 'tar' available.
func KubectlCp(ctx context.Context, cfg *envconf.Config, localPath, namespace, pod, container, remotePath string) error {
	// kubectl cp <local> <namespace>/<pod>:<remote> -c <container>
	dest := fmt.Sprintf("%s/%s:%s", namespace, pod, remotePath)
	args := KubectlArgs(cfg, "cp", localPath, dest, "-c", container)
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl cp: %w: %s", err, string(output))
	}
	return nil
}

// CopyKubesnakeBinary copies the appropriate kubesnake binary to a container.
// It auto-detects the cluster architecture and copies the matching binary.
func CopyKubesnakeBinary(ctx context.Context, cfg *envconf.Config, namespace, pod, container, remotePath string) error {
	arch, err := GetNodeArchitecture(ctx, cfg)
	if err != nil {
		return fmt.Errorf("detect architecture: %w", err)
	}

	binaryPath := ResolveBinaryPath("linux", arch)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s (run 'task build' first)", binaryPath)
	}

	if err := KubectlCp(ctx, cfg, binaryPath, namespace, pod, container, remotePath); err != nil {
		return fmt.Errorf("copy binary to %s/%s:%s: %w", namespace, pod, remotePath, err)
	}

	// Make executable
	output, err := KubectlExec(ctx, cfg, namespace, pod, container, "chmod", "+x", remotePath)
	if err != nil {
		return fmt.Errorf("chmod binary: %w: %s", err, string(output))
	}

	return nil
}

// CopyKubesnakeBinaryFunc returns an env.Func that copies the kubesnake binary to a container.
func CopyKubesnakeBinaryFunc(namespace, pod, container, remotePath string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		if err := CopyKubesnakeBinary(ctx, cfg, namespace, pod, container, remotePath); err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

// PrepareCluster prepares a cluster for the test environment with setup, teardown, and manifests.
func PrepareCluster(testenv env.Environment, clusterName string, manifestsDir string, preloadImages []string) {
	// Check for missing required commands.
	missing := CheckMissingCommands(RequiredCommands...)
	if len(missing) > 0 {
		fmt.Printf("e2e tests require tools: %s\n", strings.Join(missing, ", "))
		os.Exit(1)
	}

	// Queue cluster initialization functions.
	testenv.Setup(
		envfuncs.CreateClusterWithOpts(k3d.NewProvider(), clusterName),
		ConfigureKubectlContext(clusterName),
	)

	// Queue image preloading (all images in a single k3d command for speed).
	if len(preloadImages) > 0 {
		testenv.Setup(loadImagesToCluster(clusterName, preloadImages))
	}

	// Queue cluster configuration functions.
	testenv.Setup(
		ApplyClusterManifestsFunc(manifestsDir),
	)

	// Queue teardown functions.
	testenv.Finish(
		envfuncs.DestroyCluster(clusterName),
	)
}

// BeaconReceiverService is an HTTP server that receives e2e beacon POSTs from kubesnake.
// It collects received beacons for test verification.
type BeaconReceiverService struct {
	server  *http.Server
	beacons chan []byte
	mu      sync.Mutex
	started bool
}

// StartBeaconReceiverService creates and starts a beacon receiver HTTP server.
// The server listens on BeaconReceiverPort and accepts POST requests to BeaconReceiverPath.
// Received beacons are sent to the returned channel.
func StartBeaconReceiverService() (*BeaconReceiverService, error) {
	beacons := make(chan []byte, 100)

	mux := http.NewServeMux()
	mux.HandleFunc(BeaconReceiverPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		beacons <- body
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", BeaconReceiverPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	svc := &BeaconReceiverService{
		server:  server,
		beacons: beacons,
	}

	go func() {
		svc.mu.Lock()
		svc.started = true
		svc.mu.Unlock()

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("beacon receiver error: %v\n", err)
		}
	}()

	// Give the server a moment to start.
	time.Sleep(50 * time.Millisecond)

	return svc, nil
}

// Beacons returns the channel that receives beacon payloads.
func (s *BeaconReceiverService) Beacons() <-chan []byte {
	return s.beacons
}

// Shutdown gracefully stops the beacon receiver server.
func (s *BeaconReceiverService) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
