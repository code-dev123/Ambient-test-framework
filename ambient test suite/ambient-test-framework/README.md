# ambient-test-framework

An open-source, self-contained test suite for validating [Istio Ambient mesh](https://istio.io/latest/docs/ambient/) behaviour on Kubernetes clusters.

**Language:** Go 1.22+ | **License:** Apache 2.0

---

## Design Principles

1. **Manifest-per-scenario** — every test scenario owns its YAML manifests in a `testdata/` subfolder, embedded at compile time via `//go:embed`.
2. **`go test -run` as primary UX** — no custom tooling required. Any Go developer can run a single scenario without reading docs.
3. **Server-side apply** — manifests are applied idempotently. Re-applying with new values (e.g., updated canary weights) updates resources in-place.
4. **`t.Cleanup()` for teardown** — resources are deleted even when `t.Fatal()` is called, in LIFO order (routes before gateways, policies before workloads).
5. **Template values** — namespaces, image names, and canary weights are injected at runtime, so the same YAML works across test runs.

---

## Repository Layout

```
ambient-test-framework/
├── cmd/ambient-test/        # Optional Cobra CLI wrapper
├── pkg/
│   ├── cluster/             # ClusterProvider interface (EKS impl)
│   ├── config/              # TestConfig loader
│   ├── id/                  # Random short ID generator
│   ├── manifest/            # Apply/Delete/Wait lifecycle engine
│   ├── namespace/           # Namespace creation + ambient labelling
│   ├── workload/            # PodInfo deployer + templates
│   ├── traffic/             # HTTP, gRPC, TCP traffic helpers
│   ├── assert/              # Test assertions (connectivity, split, headers, mTLS)
│   ├── wait/                # Polling helpers (ForDeploymentsReady, etc.)
│   └── report/              # JUnit XML + JSON report writers
└── test/
    ├── suite_test.go         # TestMain — cluster connect, namespace create
    ├── setup_test.go         # Shared environment helpers
    ├── framework.go          # Environment struct + GetEnvironment()
    ├── l4/
    │   ├── mtls_enforcement/ # PeerAuthentication STRICT tests
    │   ├── l4_authz/         # L4 AuthorizationPolicy DENY tests
    │   └── cross_namespace/  # Cross-namespace ALLOW policy tests
    ├── l7/
    │   ├── canary/           # Traffic weight splitting (N-S + E-W)
    │   ├── fault_injection/  # Delay + abort fault injection
    │   ├── header_routing/   # HTTPRoute header-match routing
    │   ├── l7_authz/         # Waypoint-enforced AuthorizationPolicy
    │   ├── grpc_routing/     # GRPCRoute + health check
    │   └── traffic_mirror/   # VirtualService mirroring
    └── baseline/
        └── no_mesh/          # Baseline HTTP/TCP without Istio policy
```

---

## Prerequisites

- Go 1.22+
- A Kubernetes cluster with Istio Ambient mode installed (`ztunnel` DaemonSet present)
- `kubectl` / `kubeconfig` configured for the target cluster

---

## Quick Start

```bash
# Clone
git clone https://github.com/yourorg/ambient-test-framework
cd ambient-test-framework

# Download dependencies
go mod tidy

# Run all tests (requires a live cluster)
go test ./test/... -v -timeout 30m

# Run only L4 tests
go test ./test/l4/... -v -layer l4

# Run only canary tests
go test ./test/l7/canary/... -v -run TestCanary

# Run a specific sub-test (50/50 promote stage)
go test ./test/l7/canary/... -v -run TestCanaryFullPromote/promote_50_50

# Keep namespaces after the run (for debugging)
go test ./test/... -v -keep-ns
```

### Using the Makefile

```bash
make test                    # All tests
make test-l4                 # L4 only
make test-l7                 # L7 only
make test-canary             # Canary scenario only
make test-unit               # pkg/ unit tests (no cluster)
make check                   # fmt + vet + tidy
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-layer` | `all` | Filter by layer: `l4`, `l7`, `all` |
| `-kubeconfig` | `""` | Path to kubeconfig; falls back to `KUBECONFIG` env or `~/.kube/config` |
| `-eks-cluster` | `""` | EKS cluster name (triggers `aws eks update-kubeconfig`) |
| `-region` | `us-west-2` | AWS region |
| `-keep-ns` | `false` | Preserve test namespaces after run |
| `-timeout` | `30m` | Overall test timeout |

---

## Adding a New Scenario

Follow the pattern in `test/l7/traffic_mirror/` as a template:

```
1. Create folder:
   test/l7/my_scenario/

2. Add manifests under testdata/:
   test/l7/my_scenario/testdata/
     ├── 01-resource.yaml
     └── 02-resource.yaml

3. Write the test file:
   test/l7/my_scenario/my_scenario_test.go

4. Follow the pattern:
   //go:embed testdata/*
   var testdataFS embed.FS

   func TestMyScenario(t *testing.T) {
       env := test.GetEnvironment(t)
       applier := env.NewApplier()
       result, err := applier.ApplyFolder(ctx, testdataFS, "testdata", values)
       t.Cleanup(func() { applier.DeleteResult(cleanupCtx, result) })
       // assertions...
   }

5. Run:
   go test ./test/l7/my_scenario/... -v
```

---

## Architecture

```
go test -run TestCanaryNorthSouth ./test/l7/canary/...
│
├─ TestMain (test/suite_test.go)
│   ├─ Connect to cluster
│   ├─ Pre-flight: verify ambient (ztunnel DaemonSet)
│   ├─ Create namespaces: http-<runID>, grpc-<runID>, non-mesh-<runID>
│   ├─ Deploy client pods (traffic sources)
│   └─ SetEnvironment(env)
│
├─ TestCanaryNorthSouth (test/l7/canary/canary_test.go)
│   ├─ SETUP: ApplyFolder("testdata/workloads") → podinfo v1+v2
│   ├─ SETUP: ApplyFolder("testdata/ns-canary") → Gateway + HTTPRoute + DR + VS
│   ├─ TEST:  traffic_split_80_20 → 200 requests → verify ~80%/~20%
│   ├─ TEST:  canary_version_header → direct GET → 200 OK
│   ├─ TEST:  shift_to_50_50 → re-apply with new weights → verify split
│   └─ TEARDOWN: t.Cleanup() → DeleteResult (LIFO order)
│
└─ TestMain cleanup → DeleteAll namespaces
```
