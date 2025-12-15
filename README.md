# KubeSnake

KubeSnake is an offensive Kubernetes attack-path engine that models how an
attacker _propagates_ through a cluster.

KubeSnake is built on top of libpwn, a library of reusable attack testing
primitives.

## Local Development

### Setup

One time installations.

1. Install Docker
2. Install `kubectl`
3. Install `task`
4. Install `k3d` with `task k3d:install`

### Development Lifecycle

1. Start the cluster with `task up`
2. Iterate
3. Destroy the cluster with `task down`

### End-to-end smoke tests

The project includes a small [e2e-framework](https://github.com/kubernetes-sigs/e2e-framework) based smoke suite that provisions a k3d cluster, applies test manifests, and validates container state.

* Run the suite with `task e2e`.
* Ensure `k3d` and `kubectl` are installed and available on your `PATH`.
* Optionally pre-load container images into the cluster by setting `KUBESNAKE_E2E_IMAGES` to a comma-separated list of image names (for example, `KUBESNAKE_E2E_IMAGES=kubesnake:dev`).
