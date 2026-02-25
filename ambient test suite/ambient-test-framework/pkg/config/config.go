// pkg/config/config.go
package config

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

// TestConfig holds the full test suite configuration.
type TestConfig struct {
	Workloads WorkloadConfig `yaml:"workloads"`
	Traffic   TrafficConfig  `yaml:"traffic"`
	Timeouts  TimeoutConfig  `yaml:"timeouts"`
}

// WorkloadConfig holds workload image references.
type WorkloadConfig struct {
	PodInfoImage   string `yaml:"podInfoImage"`
	PodInfoImageV1 string `yaml:"podInfoImageV1"`
	PodInfoImageV2 string `yaml:"podInfoImageV2"`
}

// TrafficConfig holds traffic generation settings.
type TrafficConfig struct {
	DefaultRequests    int `yaml:"defaultRequests"`
	DefaultConcurrency int `yaml:"defaultConcurrency"`
	TolerancePct       int `yaml:"tolerancePct"`
}

// TimeoutConfig holds timeout settings.
type TimeoutConfig struct {
	DeploymentReadySec int `yaml:"deploymentReadySec"`
	GatewayReadySec    int `yaml:"gatewayReadySec"`
	TestOverallSec     int `yaml:"testOverallSec"`
}

// LoadFile loads a TestConfig from a YAML file.
func LoadFile(path string) (*TestConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := Defaults()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Defaults returns a TestConfig populated with sensible defaults.
func Defaults() *TestConfig {
	return &TestConfig{
		Workloads: WorkloadConfig{
			PodInfoImage:   "ghcr.io/stefanprodan/podinfo:6.5.0",
			PodInfoImageV1: "ghcr.io/stefanprodan/podinfo:6.5.0",
			PodInfoImageV2: "ghcr.io/stefanprodan/podinfo:6.5.0",
		},
		Traffic: TrafficConfig{
			DefaultRequests:    200,
			DefaultConcurrency: 10,
			TolerancePct:       15,
		},
		Timeouts: TimeoutConfig{
			DeploymentReadySec: 120,
			GatewayReadySec:    90,
			TestOverallSec:     300,
		},
	}
}
