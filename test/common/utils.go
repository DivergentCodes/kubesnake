// Package common provides shared utilities for e2e test scenarios.
package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
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

	"github.com/DivergentCodes/kubesnake/internal/config"
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

// CopyKubesnakeBinary copies the kubesnake binary from localPath to a container and makes it executable.
// localPath can point to a plain built binary (e.g. from dist/) or a scratch copy with an embedded config.
func CopyKubesnakeBinary(ctx context.Context, cfg *envconf.Config, localPath, namespace, pod, container, remotePath string) error {
	if _, err := os.Stat(localPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("binary not found: %s", localPath)
		}
		return fmt.Errorf("stat binary: %w", err)
	}

	if err := KubectlCp(ctx, cfg, localPath, namespace, pod, container, remotePath); err != nil {
		return fmt.Errorf("copy binary to %s/%s:%s: %w", namespace, pod, remotePath, err)
	}

	// Make executable (kubectl cp can strip mode bits depending on transport).
	output, err := KubectlExec(ctx, cfg, namespace, pod, container, "chmod", "+x", remotePath)
	if err != nil {
		return fmt.Errorf("chmod binary: %w: %s", err, string(output))
	}

	return nil
}

// CopyKubesnakeBinaryFunc returns an env.Func that copies the kubesnake binary at localPath to a container.
func CopyKubesnakeBinaryFunc(localPath, namespace, pod, container, remotePath string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		if err := CopyKubesnakeBinary(ctx, cfg, localPath, namespace, pod, container, remotePath); err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

type preparedKubesnakeBinaryPathKey struct{}

// PreparedKubesnakeBinaryPath returns the path to a previously prepared kubesnake binary stored in the
// e2e context (see PrepareKubesnakeBinaryWithEmbeddedConfigFunc).
func PreparedKubesnakeBinaryPath(ctx context.Context) (string, bool) {
	p, ok := ctx.Value(preparedKubesnakeBinaryPathKey{}).(string)
	if !ok || strings.TrimSpace(p) == "" {
		return "", false
	}
	return p, true
}

// CopyPreparedKubesnakeBinaryToContainersFunc copies a previously prepared kubesnake binary (stored in
// the e2e context) into the specified pod containers.
func CopyPreparedKubesnakeBinaryToContainersFunc(namespace, pod, remotePath string, containers ...string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		localPath, ok := PreparedKubesnakeBinaryPath(ctx)
		if !ok {
			return ctx, fmt.Errorf("prepared kubesnake binary missing from context (did you run PrepareKubesnakeBinaryWithEmbeddedConfigFunc?)")
		}
		for _, c := range containers {
			if strings.TrimSpace(c) == "" {
				continue
			}
			if err := CopyKubesnakeBinary(ctx, cfg, localPath, namespace, pod, c, remotePath); err != nil {
				return ctx, err
			}
		}
		return ctx, nil
	}
}

// PrepareKubesnakeBinaryWithEmbeddedConfig creates a temporary kubesnake binary on the host with the
// provided embedded JSON config applied for the cluster's architecture.
//
// The returned path should be cleaned up by the caller (os.Remove) when done.
func PrepareKubesnakeBinaryWithEmbeddedConfig(ctx context.Context, cfg *envconf.Config, embeddedConfigJSON []byte) (string, error) {
	// Validate config is JSON up-front so failures are obvious.
	var anyJSON any
	if err := json.Unmarshal(embeddedConfigJSON, &anyJSON); err != nil {
		return "", fmt.Errorf("invalid embedded config JSON: %w", err)
	}

	arch, err := GetNodeArchitecture(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("detect architecture: %w", err)
	}

	baseBinaryPath := ResolveBinaryPath("linux", arch)
	src, err := os.Open(baseBinaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("binary not found: %s (run 'task build' first)", baseBinaryPath)
		}
		return "", fmt.Errorf("open base binary: %w", err)
	}
	defer src.Close()

	st, err := src.Stat()
	if err != nil {
		return "", fmt.Errorf("stat base binary: %w", err)
	}

	tmp, err := os.CreateTemp("", fmt.Sprintf("kubesnake-%s-embedded-*", arch))
	if err != nil {
		return "", fmt.Errorf("create temp binary: %w", err)
	}
	tmpPath := tmp.Name()
	defer tmp.Close()

	if err := tmp.Chmod(st.Mode()); err != nil {
		return "", fmt.Errorf("chmod temp binary: %w", err)
	}
	if _, err := io.Copy(tmp, src); err != nil {
		return "", fmt.Errorf("copy temp binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close temp binary: %w", err)
	}

	if err := config.EmbedConfigDataIntoExecutable(tmpPath, embeddedConfigJSON); err != nil {
		return "", fmt.Errorf("embed config into temp binary: %w", err)
	}

	return tmpPath, nil
}

// PrepareKubesnakeBinaryWithEmbeddedConfigFunc prepares an embedded-config kubesnake binary on the host
// and stores its path in the e2e context for later steps.
func PrepareKubesnakeBinaryWithEmbeddedConfigFunc(embeddedConfigJSON []byte) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		p, err := PrepareKubesnakeBinaryWithEmbeddedConfig(ctx, cfg, embeddedConfigJSON)
		if err != nil {
			return ctx, err
		}
		return context.WithValue(ctx, preparedKubesnakeBinaryPathKey{}, p), nil
	}
}

// CleanupPreparedKubesnakeBinaryFunc removes a previously prepared embedded-config kubesnake binary,
// if present in the context.
func CleanupPreparedKubesnakeBinaryFunc() env.Func {
	return func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
		if p, ok := ctx.Value(preparedKubesnakeBinaryPathKey{}).(string); ok && strings.TrimSpace(p) != "" {
			_ = os.Remove(p)
		}
		return ctx, nil
	}
}

// k3dNetworkGatewayIP returns the Docker network gateway IP for the k3d cluster network.
func k3dNetworkGatewayIP(clusterName string) (string, error) {
	networkName := fmt.Sprintf("k3d-%s", clusterName)
	cmd := exec.Command("docker", "network", "inspect", networkName, "-f", "{{(index .IPAM.Config 0).Gateway}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect k3d network: %w: %s", err, strings.TrimSpace(string(output)))
	}

	gateway := strings.TrimSpace(string(output))
	if gateway == "" {
		return "", fmt.Errorf("inspect k3d network: empty gateway for %s", networkName)
	}

	return gateway, nil
}

// detectBeaconHostIP returns the preferred host IP to advertise to pods.
// Priority: explicit override (KUBESNAKE_E2E_BEACON_IP) > primary host interface IP > k3d gateway.
func detectBeaconHostIP(ctx context.Context, clusterName string) (string, error) {
	if override := strings.TrimSpace(os.Getenv("KUBESNAKE_E2E_BEACON_IP")); override != "" {
		return override, nil
	}

	if primary := primaryHostIP(); primary != "" {
		return primary, nil
	}

	gateway, err := k3dNetworkGatewayIP(clusterName)
	if err != nil {
		return "", fmt.Errorf("detect k3d gateway: %w", err)
	}

	return gateway, nil
}

// primaryHostIP attempts to discover the primary host IP by opening a UDP connection to a well-known
// address and inspecting the local socket. This avoids platform-specific parsing of routing tables.
func primaryHostIP() string {
	// "Connect" a UDP socket to a non-routable documentation address (RFC 5737).
	// This does not need to send any traffic. It just asks the OS which source IP it
	// would use for an outbound route, and is read from the local socket address.
	conn, err := net.Dial("udp", "192.0.2.1:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr()
	udpAddr, ok := localAddr.(*net.UDPAddr)
	if !ok {
		return ""
	}

	return udpAddr.IP.String()
}

// ensureBeaconHostResolves updates CoreDNS to resolve host.k3d.internal to an IP reachable on the host.
// On macOS, the docker bridge gateway often belongs to the VM hosting Docker rather than the macOS host
// itself, so dialing the gateway can fail. Prefer a host interface IP (or override) and fall back to the
// k3d gateway when necessary to keep beacons portable across Linux runners and Apple Silicon Macs.
func ensureBeaconHostResolves(ctx context.Context, cfg *envconf.Config, clusterName string) error {
	beaconHostIP, err := detectBeaconHostIP(ctx, clusterName)
	if err != nil {
		return err
	}

	args := KubectlArgs(cfg, "-n", "kube-system", "get", "configmap", "coredns", "-o", "jsonpath={.data.Corefile}")
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	corefileRaw, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("read coredns corefile: %w: %s", err, string(corefileRaw))
	}

	newBlock := fmt.Sprintf(`host.k3d.internal:53 {
hosts {
%s host.k3d.internal
fallthrough
}
forward . /etc/resolv.conf
}
`, beaconHostIP)

	// Ensure there are blank lines between the existing Corefile and the new block
	// while avoiding malformed YAML by passing the content via JSON merge patch.
	updated := strings.TrimSpace(string(corefileRaw)) + "\n\n" + newBlock

	patch := fmt.Sprintf(`{"data":{"Corefile":%q}}`, updated)
	patchArgs := KubectlArgs(cfg, "-n", "kube-system", "patch", "configmap", "coredns", "--type", "merge", "-p", patch)
	patchCmd := exec.CommandContext(ctx, "kubectl", patchArgs...)
	patchOutput, err := patchCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update coredns corefile: %w: %s", err, string(patchOutput))
	}

	// Restart CoreDNS to pick up the new configuration.
	restartArgs := KubectlArgs(cfg, "-n", "kube-system", "rollout", "restart", "deployment/coredns")
	restartCmd := exec.CommandContext(ctx, "kubectl", restartArgs...)
	restartOutput, err := restartCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart coredns: %w: %s", err, string(restartOutput))
	}

	return nil
}

// ConfigureBeaconDNSFunc ensures pods can resolve host.k3d.internal to the Docker host.
func ConfigureBeaconDNSFunc(clusterName string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		if err := ensureBeaconHostResolves(ctx, cfg, clusterName); err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

// indentLines indents each line in the provided string with the given number of spaces.
func indentLines(s string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}

	return strings.Join(lines, "\n")
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
		ConfigureBeaconDNSFunc(clusterName),
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
