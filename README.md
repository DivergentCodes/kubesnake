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
