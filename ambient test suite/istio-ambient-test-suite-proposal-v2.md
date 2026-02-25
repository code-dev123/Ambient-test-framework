# Istio Ambient Test Suite — Updated Architecture & Framework Proposal (v2)

**Project:** `ambient-test-framework` (Open Source)
**Language:** Go 1.22+
**License:** Apache 2.0
**Date:** February 2026
**Status:** PROPOSAL v2 — Self-Contained Manifest-Driven Scenarios

---

## 1. Core Design Change: Manifest-per-Scenario

Every test scenario is a **self-contained folder** that owns:

1. Its Istio manifests (YAML files) — Gateway API resources, VirtualServices, DestinationRules, AuthorizationPolicies, etc.
2. A Go test file that applies those manifests in setup, runs assertions, and deletes them in teardown.
3. A `testdata/` subfolder (Go convention) containing all YAML manifests.

This means `go test -run Canary` works out of the box — Go discovers the test function, the test function knows where its manifests live, and everything is hermetic.

---

## 2. Updated Repository Layout

```
ambient-test-framework/
├── cmd/
│   └── ambient-test/
│       └── main.go                    # Cobra CLI (optional runner)
│
├── pkg/                               # ★ Importable libraries
│   ├── cluster/
│   │   ├── provider.go                # ClusterProvider interface
│   │   ├── eks.go
│   │   └── kubeconfig.go
│   ├── namespace/
│   │   ├── manager.go
│   │   └── ambient.go
│   ├── workload/
│   │   ├── deployer.go
│   │   ├── podinfo.go
│   │   └── templates/
│   │       ├── podinfo-http.yaml
│   │       ├── podinfo-grpc.yaml
│   │       └── podinfo-canary.yaml    # v1 + v2 deployments for canary
│   ├── manifest/                      # ★ NEW: Manifest lifecycle engine
│   │   ├── applier.go                 # Apply/Delete YAML from folder
│   │   ├── template.go               # Go template rendering for manifests
│   │   ├── wait.go                    # Wait for CRD readiness after apply
│   │   └── loader.go                 # Load YAMLs from embed.FS or disk
│   ├── traffic/
│   │   ├── http.go
│   │   ├── grpc.go
│   │   └── tcp.go
│   ├── assert/
│   │   ├── connectivity.go
│   │   ├── traffic_split.go           # ★ NEW: Canary weight assertions
│   │   ├── headers.go
│   │   ├── mtls.go
│   │   └── retry.go
│   ├── wait/
│   │   ├── conditions.go
│   │   └── poll.go
│   └── report/
│       ├── junit.go
│       └── json.go
│
├── test/                              # ★ Test suites
│   ├── suite_test.go                  # TestMain — shared infra setup
│   ├── setup_test.go                  # Shared env + helpers
│   ├── framework.go                   # BaseScenario helpers
│   │
│   ├── l4/                            # Layer 4 scenarios
│   │   ├── mtls_enforcement/
│   │   │   ├── testdata/
│   │   │   │   └── peer-auth-strict.yaml
│   │   │   └── mtls_enforcement_test.go
│   │   ├── l4_authz/
│   │   │   ├── testdata/
│   │   │   │   └── l4-authz-deny.yaml
│   │   │   └── l4_authz_test.go
│   │   └── cross_namespace/
│   │       ├── testdata/
│   │       │   └── cross-ns-authz.yaml
│   │       └── cross_namespace_test.go
│   │
│   ├── l7/                            # Layer 7 scenarios
│   │   ├── canary/                    # ★ CANARY SCENARIO
│   │   │   ├── testdata/
│   │   │   │   ├── ns-canary/         # North-South canary manifests
│   │   │   │   │   ├── gateway.yaml
│   │   │   │   │   ├── httproute.yaml
│   │   │   │   │   ├── virtualservice.yaml
│   │   │   │   │   └── destination-rule.yaml
│   │   │   │   └── ew-canary/         # East-West canary manifests
│   │   │   │       ├── virtualservice.yaml
│   │   │   │       └── destination-rule.yaml
│   │   │   └── canary_test.go         # TestCanaryNorthSouth, TestCanaryEastWest
│   │   │
│   │   ├── fault_injection/
│   │   │   ├── testdata/
│   │   │   │   ├── fault-delay.yaml
│   │   │   │   └── fault-abort.yaml
│   │   │   └── fault_injection_test.go
│   │   │
│   │   ├── header_routing/
│   │   │   ├── testdata/
│   │   │   │   ├── waypoint.yaml
│   │   │   │   └── httproute-header-match.yaml
│   │   │   └── header_routing_test.go
│   │   │
│   │   ├── l7_authz/
│   │   │   ├── testdata/
│   │   │   │   ├── waypoint.yaml
│   │   │   │   └── authz-deny-path.yaml
│   │   │   └── l7_authz_test.go
│   │   │
│   │   ├── grpc_routing/
│   │   │   ├── testdata/
│   │   │   │   ├── waypoint.yaml
│   │   │   │   └── grpc-route.yaml
│   │   │   └── grpc_routing_test.go
│   │   │
│   │   └── traffic_mirror/
│   │       ├── testdata/
│   │       │   └── mirror-policy.yaml
│   │       └── traffic_mirror_test.go
│   │
│   └── baseline/
│       └── no_mesh/
│           └── no_mesh_test.go
│
├── config/
│   └── default.yaml
├── hack/
├── docs/
├── Makefile
├── go.mod
└── README.md
```

---

## 3. The Manifest Lifecycle Engine (`pkg/manifest/`)

This is the central library that every test scenario uses to apply and teardown its own manifests.

### 3.1 `applier.go` — Apply & Delete

```go
// pkg/manifest/applier.go
package manifest

import (
    "context"
    "fmt"
    "io/fs"
    "path/filepath"
    "strings"
    "time"

    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/restmapper"
)

// Applier handles applying and deleting Kubernetes/Istio YAML manifests.
type Applier struct {
    dynClient dynamic.Interface
    mapper    *restmapper.DeferredDiscoveryRESTMapper
    logger    *slog.Logger
}

func NewApplier(dynClient dynamic.Interface,
    mapper *restmapper.DeferredDiscoveryRESTMapper,
    logger *slog.Logger) *Applier {
    return &Applier{
        dynClient: dynClient,
        mapper:    mapper,
        logger:    logger,
    }
}

// ApplyResult tracks what was applied so we can delete in reverse order.
type ApplyResult struct {
    Resources []AppliedResource
}

type AppliedResource struct {
    GVR       schema.GroupVersionResource
    Namespace string
    Name      string
}

// ApplyFolder reads all .yaml/.yml files from an fs.FS path,
// templates them with the provided values, and applies them
// in filesystem order. Returns an ApplyResult for teardown.
func (a *Applier) ApplyFolder(ctx context.Context, fsys fs.FS,
    dir string, values TemplateValues) (*ApplyResult, error) {

    result := &ApplyResult{}

    files, err := fs.Glob(fsys, filepath.Join(dir, "*.yaml"))
    if err != nil {
        return nil, fmt.Errorf("glob manifests: %w", err)
    }

    // Also match .yml
    ymlFiles, _ := fs.Glob(fsys, filepath.Join(dir, "*.yml"))
    files = append(files, ymlFiles...)
    sort.Strings(files) // deterministic ordering

    for _, f := range files {
        raw, err := fs.ReadFile(fsys, f)
        if err != nil {
            return nil, fmt.Errorf("read %s: %w", f, err)
        }

        // Template the manifest (inject namespace, names, etc.)
        rendered, err := RenderTemplate(string(raw), values)
        if err != nil {
            return nil, fmt.Errorf("template %s: %w", f, err)
        }

        // A single file can contain multiple YAML docs (--- separated)
        docs := splitYAMLDocs(rendered)
        for _, doc := range docs {
            applied, err := a.applyOne(ctx, doc)
            if err != nil {
                return result, fmt.Errorf("apply %s: %w", f, err)
            }
            result.Resources = append(result.Resources, *applied)
        }

        a.logger.Info("applied manifest",
            "file", f,
            "resources", len(docs))
    }

    return result, nil
}

// ApplyFile applies a single manifest file.
func (a *Applier) ApplyFile(ctx context.Context, fsys fs.FS,
    path string, values TemplateValues) (*ApplyResult, error) {

    raw, err := fs.ReadFile(fsys, path)
    if err != nil {
        return nil, fmt.Errorf("read %s: %w", path, err)
    }

    rendered, err := RenderTemplate(string(raw), values)
    if err != nil {
        return nil, fmt.Errorf("template %s: %w", path, err)
    }

    result := &ApplyResult{}
    docs := splitYAMLDocs(rendered)
    for _, doc := range docs {
        applied, err := a.applyOne(ctx, doc)
        if err != nil {
            return result, err
        }
        result.Resources = append(result.Resources, *applied)
    }
    return result, nil
}

// DeleteResult tears down everything from an ApplyResult in
// reverse order (LIFO) — routes before gateways, policies before workloads.
func (a *Applier) DeleteResult(ctx context.Context,
    result *ApplyResult) error {

    var errs []error
    // Delete in reverse order of creation
    for i := len(result.Resources) - 1; i >= 0; i-- {
        r := result.Resources[i]
        err := a.dynClient.Resource(r.GVR).
            Namespace(r.Namespace).
            Delete(ctx, r.Name, metav1.DeleteOptions{})
        if err != nil && !errors.IsNotFound(err) {
            errs = append(errs, fmt.Errorf("delete %s/%s: %w",
                r.Namespace, r.Name, err))
        } else {
            a.logger.Info("deleted resource",
                "kind", r.GVR.Resource,
                "namespace", r.Namespace,
                "name", r.Name)
        }
    }
    return multierr.Combine(errs...)
}

// applyOne decodes a single YAML doc and applies it via server-side apply.
func (a *Applier) applyOne(ctx context.Context,
    yamlDoc string) (*AppliedResource, error) {

    obj := &unstructured.Unstructured{}
    dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
    _, gvk, err := dec.Decode([]byte(yamlDoc), nil, obj)
    if err != nil {
        return nil, fmt.Errorf("decode yaml: %w", err)
    }

    mapping, err := a.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
    if err != nil {
        return nil, fmt.Errorf("rest mapping for %s: %w", gvk, err)
    }

    // Server-side apply — idempotent, handles conflicts
    _, err = a.dynClient.Resource(mapping.Resource).
        Namespace(obj.GetNamespace()).
        Apply(ctx, obj.GetName(),
            obj, metav1.ApplyOptions{
                FieldManager: "ambient-test-framework",
                Force:        true,
            })
    if err != nil {
        return nil, fmt.Errorf("apply %s/%s: %w",
            obj.GetNamespace(), obj.GetName(), err)
    }

    return &AppliedResource{
        GVR:       mapping.Resource,
        Namespace: obj.GetNamespace(),
        Name:      obj.GetName(),
    }, nil
}

func splitYAMLDocs(raw string) []string {
    docs := strings.Split(raw, "\n---")
    var out []string
    for _, d := range docs {
        trimmed := strings.TrimSpace(d)
        if trimmed != "" && trimmed != "---" {
            out = append(out, trimmed)
        }
    }
    return out
}
```

### 3.2 `template.go` — Manifest Templating

```go
// pkg/manifest/template.go
package manifest

import (
    "bytes"
    "text/template"
)

// TemplateValues are injected into manifest YAML at apply-time.
// This lets the same manifest work across test runs with
// different namespaces, versions, and labels.
type TemplateValues struct {
    Namespace        string
    TestID           string // unique per test run, e.g., "a1b2c3"
    PodInfoImageV1   string
    PodInfoImageV2   string
    CanaryWeight     int
    StableWeight     int
    ServiceName      string
    GatewayClassName string // e.g., "istio" for ambient waypoint
    // Add more as needed — users can extend with custom fields
    Extra            map[string]string
}

func RenderTemplate(raw string, values TemplateValues) (string, error) {
    tmpl, err := template.New("manifest").
        Option("missingkey=error"). // Fail fast on typos
        Parse(raw)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, values); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```

### 3.3 `wait.go` — Wait for CRD Readiness After Apply

```go
// pkg/manifest/wait.go
package manifest

// WaitForReady polls until all resources in an ApplyResult
// reach their expected ready state.
func (a *Applier) WaitForReady(ctx context.Context,
    result *ApplyResult, timeout time.Duration) error {

    return wait.PollUntilContextTimeout(ctx,
        2*time.Second, timeout, true,
        func(ctx context.Context) (bool, error) {
            for _, r := range result.Resources {
                ready, err := a.isResourceReady(ctx, r)
                if err != nil || !ready {
                    return false, nil
                }
            }
            return true, nil
        })
}

func (a *Applier) isResourceReady(ctx context.Context,
    r AppliedResource) (bool, error) {

    obj, err := a.dynClient.Resource(r.GVR).
        Namespace(r.Namespace).
        Get(ctx, r.Name, metav1.GetOptions{})
    if err != nil {
        return false, err
    }

    // Check common readiness patterns
    switch r.GVR.Resource {
    case "gateways":
        return gatewayReady(obj), nil
    case "httproutes", "grpcroutes":
        return routeAccepted(obj), nil
    case "deployments":
        return deploymentReady(obj), nil
    default:
        // For CRDs without status (like AuthzPolicy),
        // existence is sufficient
        return true, nil
    }
}
```

---

## 4. Canary Test Scenario — Complete Example

### 4.1 Folder Structure

```
test/l7/canary/
├── testdata/
│   ├── workloads/
│   │   ├── podinfo-v1.yaml            # Deployment + Service for v1
│   │   └── podinfo-v2.yaml            # Deployment for v2 (canary)
│   ├── ns-canary/                     # North-South canary manifests
│   │   ├── 01-gateway.yaml            # Gateway API Gateway resource
│   │   ├── 02-httproute.yaml          # HTTPRoute with weight split
│   │   ├── 03-destination-rule.yaml   # Subsets: v1, v2
│   │   └── 04-virtualservice.yaml     # (optional, if not using Gateway API)
│   └── ew-canary/                     # East-West canary manifests
│       ├── 01-destination-rule.yaml   # Subsets: v1, v2
│       └── 02-virtualservice.yaml     # Internal traffic split
├── canary_test.go                     # Test functions
└── README.md                          # Scenario documentation
```

### 4.2 Manifests (with template variables)

**`testdata/workloads/podinfo-v1.yaml`**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: podinfo-v1
  namespace: {{ .Namespace }}
  labels:
    app: podinfo
    version: v1
    test-id: {{ .TestID }}
spec:
  replicas: 2
  selector:
    matchLabels:
      app: podinfo
      version: v1
  template:
    metadata:
      labels:
        app: podinfo
        version: v1
        test-id: {{ .TestID }}
    spec:
      containers:
        - name: podinfo
          image: {{ .PodInfoImageV1 }}
          ports:
            - containerPort: 9898
              name: http
          env:
            - name: PODINFO_UI_MESSAGE
              value: "v1-stable"
---
apiVersion: v1
kind: Service
metadata:
  name: podinfo
  namespace: {{ .Namespace }}
  labels:
    test-id: {{ .TestID }}
spec:
  selector:
    app: podinfo
  ports:
    - port: 9898
      targetPort: 9898
      name: http
```

**`testdata/workloads/podinfo-v2.yaml`**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: podinfo-v2
  namespace: {{ .Namespace }}
  labels:
    app: podinfo
    version: v2
    test-id: {{ .TestID }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: podinfo
      version: v2
  template:
    metadata:
      labels:
        app: podinfo
        version: v2
        test-id: {{ .TestID }}
    spec:
      containers:
        - name: podinfo
          image: {{ .PodInfoImageV2 }}
          ports:
            - containerPort: 9898
              name: http
          env:
            - name: PODINFO_UI_MESSAGE
              value: "v2-canary"
```

**`testdata/ns-canary/01-gateway.yaml`**
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: podinfo-gateway
  namespace: {{ .Namespace }}
  labels:
    test-id: {{ .TestID }}
  annotations:
    # In ambient mode, the gateway is handled by waypoint proxy
    networking.istio.io/service-type: ClusterIP
spec:
  gatewayClassName: {{ .GatewayClassName }}
  listeners:
    - name: http
      port: 80
      protocol: HTTP
      allowedRoutes:
        namespaces:
          from: Same
```

**`testdata/ns-canary/02-httproute.yaml`**
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: podinfo-canary-route
  namespace: {{ .Namespace }}
  labels:
    test-id: {{ .TestID }}
spec:
  parentRefs:
    - name: podinfo-gateway
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /
      backendRefs:
        - name: podinfo
          port: 9898
          weight: {{ .StableWeight }}
        - name: podinfo-canary
          port: 9898
          weight: {{ .CanaryWeight }}
```

**`testdata/ns-canary/03-destination-rule.yaml`**
```yaml
apiVersion: networking.istio.io/v1
kind: DestinationRule
metadata:
  name: podinfo-subsets
  namespace: {{ .Namespace }}
  labels:
    test-id: {{ .TestID }}
spec:
  host: podinfo.{{ .Namespace }}.svc.cluster.local
  trafficPolicy:
    connectionPool:
      http:
        h2UpgradePolicy: UPGRADE
  subsets:
    - name: stable
      labels:
        version: v1
    - name: canary
      labels:
        version: v2
```

**`testdata/ns-canary/04-virtualservice.yaml`**
```yaml
# Alternative to HTTPRoute — use if Gateway API is not available
apiVersion: networking.istio.io/v1
kind: VirtualService
metadata:
  name: podinfo-canary-vs
  namespace: {{ .Namespace }}
  labels:
    test-id: {{ .TestID }}
spec:
  hosts:
    - podinfo.{{ .Namespace }}.svc.cluster.local
  gateways:
    - mesh
    - podinfo-gateway
  http:
    - route:
        - destination:
            host: podinfo.{{ .Namespace }}.svc.cluster.local
            subset: stable
          weight: {{ .StableWeight }}
        - destination:
            host: podinfo.{{ .Namespace }}.svc.cluster.local
            subset: canary
          weight: {{ .CanaryWeight }}
```

**`testdata/ew-canary/01-destination-rule.yaml`**
```yaml
apiVersion: networking.istio.io/v1
kind: DestinationRule
metadata:
  name: podinfo-ew-subsets
  namespace: {{ .Namespace }}
  labels:
    test-id: {{ .TestID }}
spec:
  host: podinfo.{{ .Namespace }}.svc.cluster.local
  subsets:
    - name: stable
      labels:
        version: v1
    - name: canary
      labels:
        version: v2
```

**`testdata/ew-canary/02-virtualservice.yaml`**
```yaml
apiVersion: networking.istio.io/v1
kind: VirtualService
metadata:
  name: podinfo-ew-canary
  namespace: {{ .Namespace }}
  labels:
    test-id: {{ .TestID }}
spec:
  hosts:
    - podinfo.{{ .Namespace }}.svc.cluster.local
  gateways:
    - mesh    # East-West only — internal mesh traffic
  http:
    - route:
        - destination:
            host: podinfo.{{ .Namespace }}.svc.cluster.local
            subset: stable
          weight: {{ .StableWeight }}
        - destination:
            host: podinfo.{{ .Namespace }}.svc.cluster.local
            subset: canary
          weight: {{ .CanaryWeight }}
```

### 4.3 Test Code (`canary_test.go`)

```go
// test/l7/canary/canary_test.go
package canary

import (
    "context"
    "embed"
    "fmt"
    "math"
    "sync"
    "sync/atomic"
    "testing"
    "time"

    "github.com/yourorg/ambient-test-framework/pkg/assert"
    "github.com/yourorg/ambient-test-framework/pkg/manifest"
    "github.com/yourorg/ambient-test-framework/pkg/traffic"
    "github.com/yourorg/ambient-test-framework/pkg/wait"
    "github.com/yourorg/ambient-test-framework/test"
)

// Embed all testdata so the binary is self-contained.
// No external file paths needed at runtime.
//
//go:embed testdata/*
var testdataFS embed.FS

// templateValues builds the TemplateValues for this scenario,
// injecting the shared test environment's namespace and
// scenario-specific canary weights.
func templateValues(env *test.Environment,
    stableWeight, canaryWeight int) manifest.TemplateValues {
    return manifest.TemplateValues{
        Namespace:        env.Namespaces.HTTP,
        TestID:           env.RunID,
        PodInfoImageV1:   env.Config.Workloads.PodInfoImage,
        PodInfoImageV2:   env.Config.Workloads.PodInfoImage, // same image, diff env
        StableWeight:     stableWeight,
        CanaryWeight:     canaryWeight,
        GatewayClassName: "istio",
    }
}

// ============================================================
// go test -run TestCanaryNorthSouth
// ============================================================

func TestCanaryNorthSouth(t *testing.T) {
    t.Parallel()

    env := test.GetEnvironment(t) // fetch shared test environment
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    applier := env.NewApplier()
    values := templateValues(env, 80, 20) // 80/20 split

    // ── SETUP: Deploy workloads ────────────────────────────
    workloadResult, err := applier.ApplyFolder(ctx, testdataFS,
        "testdata/workloads", values)
    if err != nil {
        t.Fatalf("deploy workloads: %v", err)
    }
    // Register cleanup IMMEDIATELY — runs even on t.Fatal
    t.Cleanup(func() {
        cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
        defer c()
        // Delete manifests in reverse: routes → gateway → workloads
        if nsResult != nil {
            applier.DeleteResult(cleanupCtx, nsResult)
        }
        applier.DeleteResult(cleanupCtx, workloadResult)
    })

    // Wait for workloads to be ready
    err = wait.ForDeploymentsReady(ctx, env.Clientset,
        env.Namespaces.HTTP, "test-id="+env.RunID, 2*time.Minute)
    if err != nil {
        t.Fatalf("workloads not ready: %v", err)
    }

    // ── SETUP: Apply N-S canary manifests ──────────────────
    nsResult, err := applier.ApplyFolder(ctx, testdataFS,
        "testdata/ns-canary", values)
    if err != nil {
        t.Fatalf("apply ns-canary manifests: %v", err)
    }

    // Wait for Gateway and HTTPRoute to be accepted
    err = applier.WaitForReady(ctx, nsResult, 90*time.Second)
    if err != nil {
        t.Fatalf("ns-canary resources not ready: %v", err)
    }

    // ── TEST: Verify traffic split (80/20) ─────────────────
    t.Run("traffic_split_80_20", func(t *testing.T) {
        verifyTrafficSplit(ctx, t, env, 80, 20, 200)
    })

    // ── TEST: Canary responds correctly ────────────────────
    t.Run("canary_version_header", func(t *testing.T) {
        // Hit the canary subset directly to verify it serves v2
        resp, err := traffic.HTTPGet(ctx,
            env.Workloads.HTTPClient,
            traffic.Endpoint{
                Host: fmt.Sprintf("podinfo.%s.svc.cluster.local",
                    env.Namespaces.HTTP),
                Port: 9898,
                Path: "/api/info",
            })
        assert.NoError(t, err)
        assert.StatusOK(t, resp)
    })

    // ── TEST: Shift to 50/50, verify convergence ───────────
    t.Run("shift_to_50_50", func(t *testing.T) {
        newValues := templateValues(env, 50, 50)

        // Re-apply the same folder with updated weights
        // Server-side apply handles the update idempotently
        _, err := applier.ApplyFolder(ctx, testdataFS,
            "testdata/ns-canary", newValues)
        if err != nil {
            t.Fatalf("re-apply with 50/50: %v", err)
        }

        // Wait for route to converge
        time.Sleep(5 * time.Second) // brief settle

        verifyTrafficSplit(ctx, t, env, 50, 50, 200)
    })
}

// ============================================================
// go test -run TestCanaryEastWest
// ============================================================

func TestCanaryEastWest(t *testing.T) {
    t.Parallel()

    env := test.GetEnvironment(t)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    applier := env.NewApplier()
    values := templateValues(env, 90, 10) // 90/10 split

    // ── SETUP: Deploy workloads ────────────────────────────
    workloadResult, err := applier.ApplyFolder(ctx, testdataFS,
        "testdata/workloads", values)
    if err != nil {
        t.Fatalf("deploy workloads: %v", err)
    }
    t.Cleanup(func() {
        cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
        defer c()
        if ewResult != nil {
            applier.DeleteResult(cleanupCtx, ewResult)
        }
        applier.DeleteResult(cleanupCtx, workloadResult)
    })

    err = wait.ForDeploymentsReady(ctx, env.Clientset,
        env.Namespaces.HTTP, "test-id="+env.RunID, 2*time.Minute)
    if err != nil {
        t.Fatalf("workloads not ready: %v", err)
    }

    // ── SETUP: Apply E-W canary manifests ──────────────────
    ewResult, err := applier.ApplyFolder(ctx, testdataFS,
        "testdata/ew-canary", values)
    if err != nil {
        t.Fatalf("apply ew-canary manifests: %v", err)
    }
    err = applier.WaitForReady(ctx, ewResult, 60*time.Second)
    if err != nil {
        t.Fatalf("ew-canary resources not ready: %v", err)
    }

    // ── TEST: E-W traffic split ────────────────────────────
    t.Run("ew_traffic_split_90_10", func(t *testing.T) {
        verifyTrafficSplit(ctx, t, env, 90, 10, 300)
    })

    // ── TEST: Non-mesh client bypasses split ───────────────
    t.Run("non_mesh_no_split", func(t *testing.T) {
        // Traffic from non-mesh namespace should reach
        // podinfo service normally (round-robin, no weight)
        for i := 0; i < 20; i++ {
            resp, err := traffic.HTTPGet(ctx,
                env.Workloads.NonMeshClient,
                traffic.Endpoint{
                    Host: fmt.Sprintf("podinfo.%s.svc.cluster.local",
                        env.Namespaces.HTTP),
                    Port: 9898,
                    Path: "/api/info",
                })
            assert.NoError(t, err)
            assert.StatusOK(t, resp)
        }
    })
}

// ============================================================
// go test -run TestCanaryFullPromote
// ============================================================

func TestCanaryFullPromote(t *testing.T) {
    t.Parallel()

    env := test.GetEnvironment(t)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    applier := env.NewApplier()

    // Start at 90/10
    values := templateValues(env, 90, 10)

    workloadResult, err := applier.ApplyFolder(ctx, testdataFS,
        "testdata/workloads", values)
    if err != nil {
        t.Fatalf("deploy workloads: %v", err)
    }
    var currentResult *manifest.ApplyResult
    t.Cleanup(func() {
        cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
        defer c()
        if currentResult != nil {
            applier.DeleteResult(cleanupCtx, currentResult)
        }
        applier.DeleteResult(cleanupCtx, workloadResult)
    })

    err = wait.ForDeploymentsReady(ctx, env.Clientset,
        env.Namespaces.HTTP, "test-id="+env.RunID, 2*time.Minute)
    if err != nil {
        t.Fatalf("workloads not ready: %v", err)
    }

    // Progressive promotion: 90/10 → 50/50 → 0/100
    stages := []struct {
        stable, canary int
    }{
        {90, 10},
        {50, 50},
        {0, 100},
    }

    for _, stage := range stages {
        name := fmt.Sprintf("promote_%d_%d", stage.stable, stage.canary)
        t.Run(name, func(t *testing.T) {
            stageValues := templateValues(env, stage.stable, stage.canary)

            currentResult, err = applier.ApplyFolder(ctx, testdataFS,
                "testdata/ew-canary", stageValues)
            if err != nil {
                t.Fatalf("apply stage %d/%d: %v",
                    stage.stable, stage.canary, err)
            }
            time.Sleep(5 * time.Second)

            verifyTrafficSplit(ctx, t, env,
                stage.stable, stage.canary, 200)
        })
    }
}

// ============================================================
// Shared helper: verify traffic distribution matches weights
// ============================================================

func verifyTrafficSplit(ctx context.Context, t *testing.T,
    env *test.Environment,
    expectedStable, expectedCanary, totalRequests int) {

    t.Helper()

    var v1Hits, v2Hits atomic.Int64
    var wg sync.WaitGroup

    endpoint := traffic.Endpoint{
        Host: fmt.Sprintf("podinfo.%s.svc.cluster.local",
            env.Namespaces.HTTP),
        Port: 9898,
        Path: "/api/info",
    }

    // Send traffic in parallel for speed
    concurrency := 10
    perWorker := totalRequests / concurrency

    for w := 0; w < concurrency; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for i := 0; i < perWorker; i++ {
                resp, err := traffic.HTTPGetJSON(ctx,
                    env.Workloads.HTTPClient, endpoint)
                if err != nil {
                    continue
                }
                if msg, ok := resp["message"].(string); ok {
                    switch msg {
                    case "v1-stable":
                        v1Hits.Add(1)
                    case "v2-canary":
                        v2Hits.Add(1)
                    }
                }
            }
        }()
    }
    wg.Wait()

    total := v1Hits.Load() + v2Hits.Load()
    if total == 0 {
        t.Fatal("no successful responses received")
    }

    actualV1Pct := float64(v1Hits.Load()) / float64(total) * 100
    actualV2Pct := float64(v2Hits.Load()) / float64(total) * 100

    // Allow ±15% tolerance for statistical variance
    tolerance := 15.0
    assert.InDelta(t, float64(expectedStable), actualV1Pct, tolerance,
        "v1 (stable) traffic: expected ~%d%%, got %.1f%%",
        expectedStable, actualV1Pct)
    assert.InDelta(t, float64(expectedCanary), actualV2Pct, tolerance,
        "v2 (canary) traffic: expected ~%d%%, got %.1f%%",
        expectedCanary, actualV2Pct)

    t.Logf("Traffic split: v1=%.1f%% (%d), v2=%.1f%% (%d), total=%d",
        actualV1Pct, v1Hits.Load(),
        actualV2Pct, v2Hits.Load(), total)
}
```

---

## 5. How `go test -run` Works With This Layout

```bash
# Run ALL canary tests
go test ./test/l7/canary/... -v

# Run only North-South canary
go test ./test/l7/canary/... -v -run TestCanaryNorthSouth

# Run only East-West canary
go test ./test/l7/canary/... -v -run TestCanaryEastWest

# Run the full promotion lifecycle test
go test ./test/l7/canary/... -v -run TestCanaryFullPromote

# Run a specific sub-test (the 50/50 stage)
go test ./test/l7/canary/... -v -run TestCanaryFullPromote/promote_50_50

# Run ALL L7 tests
go test ./test/l7/... -v

# Run ALL L4 tests
go test ./test/l4/... -v

# Run everything
go test ./test/... -v -timeout 30m

# Using the flag-based layer filter (via TestMain)
go test ./test/... -v -layer l4
go test ./test/... -v -layer l7
go test ./test/... -v -layer all
```

---

## 6. Shared Test Environment (`test/framework.go`)

```go
// test/framework.go
package test

import (
    "sync"
    "testing"

    "github.com/yourorg/ambient-test-framework/pkg/manifest"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/kubernetes"
)

// Environment is the shared state created once in TestMain.
type Environment struct {
    Clientset   kubernetes.Interface
    DynClient   dynamic.Interface
    Namespaces  NamespaceSet
    Workloads   WorkloadSet
    Config      *TestConfig
    RunID       string         // unique per test run
    Logger      *slog.Logger
}

var (
    globalEnv  *Environment
    envOnce    sync.Once
)

// SetEnvironment is called by TestMain after setup.
func SetEnvironment(env *Environment) {
    globalEnv = env
}

// GetEnvironment retrieves the shared environment.
// Fails the test if TestMain hasn't set it up.
func GetEnvironment(t *testing.T) *Environment {
    t.Helper()
    if globalEnv == nil {
        t.Fatal("test environment not initialized — " +
            "is TestMain running?")
    }
    return globalEnv
}

// NewApplier creates a manifest.Applier tied to this environment.
func (e *Environment) NewApplier() *manifest.Applier {
    mapper := restmapper.NewDeferredDiscoveryRESTMapper(
        memory.NewMemCacheClient(e.DiscoveryClient))
    return manifest.NewApplier(e.DynClient, mapper, e.Logger)
}
```

---

## 7. Updated TestMain (`test/suite_test.go`)

```go
// test/suite_test.go
package test

import (
    "context"
    "flag"
    "os"
    "testing"

    "github.com/yourorg/ambient-test-framework/pkg/cluster"
    "github.com/yourorg/ambient-test-framework/pkg/namespace"
    "github.com/yourorg/ambient-test-framework/pkg/workload"
    "github.com/yourorg/ambient-test-framework/pkg/id"
)

var (
    flagLayer      = flag.String("layer", "all", "l4, l7, all")
    flagKeepNS     = flag.Bool("keep-ns", false, "preserve namespaces")
    flagKubeconfig = flag.String("kubeconfig", "", "kubeconfig path")
    flagEKSCluster = flag.String("eks-cluster", "", "EKS cluster name")
    flagRegion     = flag.String("region", "us-west-2", "AWS region")
)

func TestMain(m *testing.M) {
    flag.Parse()
    ctx := context.Background()
    runID := id.Short() // e.g., "a1b2c3"

    // 1. Connect
    provider := cluster.NewEKSProvider()
    clientset, dynClient, err := provider.Connect(ctx, cluster.ConnectOpts{
        Kubeconfig:  *flagKubeconfig,
        ClusterName: *flagEKSCluster,
        Region:      *flagRegion,
    })
    if err != nil {
        slog.Error("cluster connect failed", "error", err)
        os.Exit(1)
    }

    // 2. Pre-flight
    status, err := provider.Healthcheck(ctx)
    if err != nil || !status.AmbientEnabled {
        slog.Error("pre-flight failed", "error", err)
        os.Exit(1)
    }

    // 3. Create namespaces (shared across all scenarios)
    nsMgr := namespace.NewManager(clientset)
    ns, err := nsMgr.CreateTestNamespaces(ctx, namespace.Config{
        HTTPPrefix:    "http",
        GRPCPrefix:    "grpc",
        NonMeshPrefix: "non-mesh",
        Suffix:        runID,
        AmbientLabel:  true,
    })
    if err != nil {
        slog.Error("namespace creation failed", "error", err)
        os.Exit(1)
    }

    // 4. Deploy baseline workloads (client pods for traffic generation)
    deployer := workload.NewPodInfoDeployer(clientset)
    wl, err := deployer.DeployClients(ctx, ns)
    if err != nil {
        slog.Error("workload deploy failed", "error", err)
        os.Exit(1)
    }

    // 5. Set shared environment
    SetEnvironment(&Environment{
        Clientset:  clientset,
        DynClient:  dynClient,
        Namespaces: *ns,
        Workloads:  *wl,
        RunID:      runID,
        Config:     LoadConfig(),
        Logger:     slog.Default(),
    })

    // 6. Run
    code := m.Run()

    // 7. Teardown
    if !*flagKeepNS {
        nsMgr.DeleteAll(context.Background(), ns)
    }

    os.Exit(code)
}
```

---

## 8. Pattern: Adding a New Scenario (e.g., Traffic Mirror)

Step-by-step for any contributor:

```
1. Create folder:
   test/l7/traffic_mirror/

2. Add manifests:
   test/l7/traffic_mirror/testdata/
     ├── mirror-destination-rule.yaml
     └── mirror-virtualservice.yaml

3. Write test:
   test/l7/traffic_mirror/traffic_mirror_test.go

4. Follow the pattern:
   - //go:embed testdata/*
   - Build TemplateValues from env
   - applier.ApplyFolder() in setup
   - t.Cleanup() calls applier.DeleteResult()
   - Assertions in t.Run() subtests

5. Run:
   go test ./test/l7/traffic_mirror/... -v
```

---

## 9. Architectural Flow Diagram (Updated)

```
go test -run TestCanaryNorthSouth ./test/l7/canary/...
│
├─ TestMain (test/suite_test.go)
│   ├─ Connect EKS cluster
│   ├─ Pre-flight healthcheck
│   ├─ Create namespaces: http-a1b2c3, grpc-a1b2c3, non-mesh-a1b2c3
│   ├─ Deploy client pods (traffic sources)
│   └─ SetEnvironment(env)
│
├─ TestCanaryNorthSouth (test/l7/canary/canary_test.go)
│   │
│   ├─ SETUP
│   │   ├─ applier.ApplyFolder("testdata/workloads", values)
│   │   │   → podinfo-v1 Deployment + Service
│   │   │   → podinfo-v2 Deployment
│   │   │
│   │   ├─ applier.ApplyFolder("testdata/ns-canary", values)
│   │   │   → Gateway (01-gateway.yaml)
│   │   │   → HTTPRoute with 80/20 weights (02-httproute.yaml)
│   │   │   → DestinationRule with subsets (03-destination-rule.yaml)
│   │   │   → VirtualService (04-virtualservice.yaml)
│   │   │
│   │   └─ applier.WaitForReady() → polls until all resources are ready
│   │
│   ├─ TEST
│   │   ├─ t.Run("traffic_split_80_20")
│   │   │   → 200 requests → verify ~80% v1, ~20% v2
│   │   ├─ t.Run("canary_version_header")
│   │   │   → direct request → verify v2 responds
│   │   └─ t.Run("shift_to_50_50")
│   │       → re-apply with new weights → verify convergence
│   │
│   └─ TEARDOWN (t.Cleanup — runs even on failure)
│       ├─ applier.DeleteResult(nsResult)   ← reverse order
│       │   → delete VirtualService
│       │   → delete DestinationRule
│       │   → delete HTTPRoute
│       │   → delete Gateway
│       └─ applier.DeleteResult(workloadResult)
│           → delete podinfo-v2 Deployment
│           → delete podinfo-v1 Deployment + Service
│
└─ TestMain cleanup
    └─ Delete namespaces (unless --keep-ns)
```

---

## 10. Key Design Decisions Summary

| Decision | Rationale |
|----------|-----------|
| **`embed.FS` for manifests** | Binary is self-contained; no file path dependencies at runtime. Works in containers, CI, and local dev. |
| **`testdata/` convention** | Go tooling ignores `testdata/` in builds. Well-known pattern. |
| **Numbered file prefixes (`01-`, `02-`)** | `sort.Strings()` gives deterministic apply order. Gateways before routes, rules before policies. |
| **`t.Cleanup()` for teardown** | Runs even on `t.Fatal()`. Registered immediately after apply, so resources are always cleaned. |
| **LIFO deletion order** | Routes deleted before gateways, policies before workloads — avoids dangling references. |
| **Server-side apply** | Idempotent. Re-applying with changed weights updates in-place. No delete-then-create needed. |
| **Template values, not `sed`** | Type-safe, testable, IDE-friendly. No shell scripting. |
| **Separate folders per variant** | `ns-canary/` vs `ew-canary/` — clear separation, independent lifecycle, easy to understand for contributors. |
| **`go test -run` as primary UX** | Zero custom tooling needed. Any Go developer knows this immediately. |
