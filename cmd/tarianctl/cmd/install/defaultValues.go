package install

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// NatsHelmDefaultValues represents default Helm values for NATS configuration.
type NatsHelmDefaultValues struct {
	Nats NatsValues `yaml:"nats"`
}

// NatsValues holds configuration options related to NATS.
type NatsValues struct {
	Image     string        `yaml:"image"`
	Jetstream JetstreamOpts `yaml:"jetstream"`
}

// JetstreamOpts contains configuration options for NATS Jetstream.
type JetstreamOpts struct {
	Enabled     bool        `yaml:"enabled"`
	MemStorage  StorageOpts `yaml:"memStorage"`
	FileStorage StorageOpts `yaml:"fileStorage"`
}

// StorageOpts holds options for storage in NATS Jetstream.
type StorageOpts struct {
	Enabled bool   `yaml:"enabled"`
	Size    string `yaml:"size"`
}

func natsHelmDefaultValues(natsValuesFile string) error {
	natsValues := &NatsHelmDefaultValues{
		Nats: NatsValues{
			Image: "nats:alpine",
			Jetstream: JetstreamOpts{
				Enabled: true,
				MemStorage: StorageOpts{
					Enabled: true,
					Size:    "128Mi",
				},
				FileStorage: StorageOpts{
					Enabled: true,
					Size:    "200Mi",
				},
			},
		},
	}

	valuesYAML, err := yaml.Marshal(natsValues)
	if err != nil {
		return fmt.Errorf("failed to marshal values to YAML: %v", err)
	}
	if err := os.WriteFile(natsValuesFile, valuesYAML, 0644); err != nil {
		return fmt.Errorf("failed to write values to temporary file: %v", err)
	}

	return nil
}

// AlphaConfig represents default Helm values for Dgraph Alpha.
type AlphaConfig struct {
	Alpha Config `yaml:"alpha"`
}

// Config holds configuration options for Dgraph.
type Config struct {
	ExtraEnvs []Env `yaml:"extraEnvs"`
}

// Env represents an environment variable with a name and value.
type Env struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func dgraphHelmDefaultValues(dgraphValuesFile string) error {
	dgraphValues := &AlphaConfig{
		Alpha: Config{
			ExtraEnvs: []Env{
				{
					Name:  "DGRAPH_ALPHA_SECURITY",
					Value: "whitelist=0.0.0.0/0",
				},
			},
		},
	}

	valuesYAML, err := yaml.Marshal(dgraphValues)
	if err != nil {
		return fmt.Errorf("failed to marshal values to YAML: %v", err)
	}
	if err := os.WriteFile(dgraphValuesFile, valuesYAML, 0644); err != nil {
		return fmt.Errorf("failed to write values to temporary file: %v", err)
	}

	return nil
}
