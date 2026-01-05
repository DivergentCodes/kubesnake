# E2E Tests

This directory contains the repository’s end-to-end (E2E) tests. They use the
Kubernetes [`e2e-framework`](https://github.com/kubernetes-sigs/e2e-framework)
to stand up a real Kubernetes environment (k3d), apply manifests, run
validations, and tear everything down.

## Approach

- **Real cluster, real pods**: tests create a uniquely named k3d cluster, apply
  manifests from `test/**/manifests/`, then execute checks against live
  pods/containers via `kubectl`.
- **Portable binary under test**: the suite copies the built `kubesnake` binary
  from `dist/` into target containers (auto-detecting the cluster node
  architecture and selecting the matching `dist/kubesnake-linux-*` artifact).
- **Beacon callback verification**: when the embedded config contains
  `e2e.beaconUrl`, `kubesnake` runs in e2e mode and beacons back proofs for each
  container it executed in.
  - The test harness runs a local "beacon receiver" HTTP server on `:18080` to
    capture callback proofs.
  - `kubesnake` reads a `/var/run/kubesnake/beacon.json` JSON file unique to the
    container.
  - `kubesnake` sends a `POST` request with the JSON beacon file to the URL
    defined in `e2e.beaconUrl`.
- **Deployment parity**: the e2e harness embeds the config into the correct
  `dist/kubesnake-linux-{amd64|arm64}` binary on the host _before_ uploading it
  to the target cluster (it does not embed config inside the cluster).
- **DNS bridging for host callbacks**: the harness patches CoreDNS so pods can
  resolve `host.k3d.internal` to an IP reachable on the host running the tests.
  You can override the chosen IP via `KUBESNAKE_E2E_BEACON_IP`.

## Test Suites

### Smoke Tests

The smoke test suite verifies basic functionality of KubeSnake, including:

- **Binary distribution**: `kubesnake` is copied into each target container and
  is executable.
- **Beacon file presence**: `/var/run/kubesnake/beacon.json` exists in each
  container.
- **Beacon callback**: running `kubesnake` results in a beacon callback received
  by the host test process. Tests validate the payload is valid JSON and matches
  the expected `namespace`/`pod`/`container`.

## Beacon File Format

The beacon files are meant to be unqiue proofs per container that demonstrate
KubeSnake successfully executed in a given container. It can be used to verify
propagation and sequencing.

E2E beaconing is enabled when the embedded config includes an `e2e.beaconUrl`,
for example:

```json
{ "e2e": { "beaconUrl": "http://host.k3d.internal:18080/beacon" } }
```

Beacon files are stored at `/var/run/kubesnake/beacon.json` and follow this
format:

```json
{
  "v": 1,
  "namespace": "some-ns",
  "pod": "foo",
  "container": "bar",
  "pod_uid": "",
  "beacon_id": "some-ns/foo:app"
}
```

## Running locally

Requirements:

- `docker`
- `k3d`
- `kubectl`

Execution:

- `task test`: run all unit and e2e tests.
- `task test:e2e`: run e2e tests in parallel (default).
- `task test:e2e:serial`: run e2e tests in serial.

Notes:

- The tests are built with the `e2e` build tag and live under `test/`.
