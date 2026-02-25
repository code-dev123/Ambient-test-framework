# Canary Scenario

Tests traffic weight splitting between a stable (v1) and canary (v2) version
of podinfo, covering both North-South (ingress gateway) and East-West
(service mesh internal) traffic paths.

## Tests

| Test | Description |
|------|-------------|
| `TestCanaryNorthSouth` | 80/20 weight split via Gateway API HTTPRoute; shifts to 50/50 |
| `TestCanaryEastWest` | 90/10 weight split via VirtualService for internal mesh traffic |
| `TestCanaryFullPromote` | Progressive promotion: 90/10 → 50/50 → 0/100 |

## Manifest Layout

```
testdata/
├── workloads/
│   ├── podinfo-v1.yaml   # Deployment (v1) + Service
│   └── podinfo-v2.yaml   # Deployment (v2)
├── ns-canary/            # North-South via Gateway API
│   ├── 01-gateway.yaml
│   ├── 02-httproute.yaml
│   ├── 03-destination-rule.yaml
│   └── 04-virtualservice.yaml
└── ew-canary/            # East-West via VirtualService
    ├── 01-destination-rule.yaml
    └── 02-virtualservice.yaml
```

## Running

```bash
# All canary tests
go test ./test/l7/canary/... -v

# Only North-South
go test ./test/l7/canary/... -v -run TestCanaryNorthSouth

# Only the 50/50 promote stage
go test ./test/l7/canary/... -v -run TestCanaryFullPromote/promote_50_50
```
