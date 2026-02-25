// pkg/workload/podinfo.go
package workload

const (
	// DefaultPodInfoImage is the default image used for podinfo workloads.
	DefaultPodInfoImage = "ghcr.io/stefanprodan/podinfo:6.5.0"
	// DefaultPodInfoPort is the HTTP port podinfo listens on.
	DefaultPodInfoPort = 9898
	// DefaultPodInfoGRPCPort is the gRPC port podinfo listens on.
	DefaultPodInfoGRPCPort = 9999
)

// PodInfoConfig configures a podinfo workload deployment.
type PodInfoConfig struct {
	Name      string
	Namespace string
	Image     string
	Replicas  int32
	Version   string // injected as PODINFO_UI_MESSAGE env var
	Labels    map[string]string
}

// DefaultPodInfoConfig returns a PodInfoConfig with sensible defaults.
func DefaultPodInfoConfig(name, namespace, version string) PodInfoConfig {
	return PodInfoConfig{
		Name:      name,
		Namespace: namespace,
		Image:     DefaultPodInfoImage,
		Replicas:  1,
		Version:   version,
		Labels: map[string]string{
			"app": "podinfo",
		},
	}
}
