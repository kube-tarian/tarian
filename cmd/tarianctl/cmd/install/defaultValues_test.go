package install

import (
	"os"
	"strings"
	"testing"
)

func TestNatsHelmDefaultValues(t *testing.T) {
	natsValuesFile := "nats_values_test.yaml"

	defer func() {
		if err := os.Remove(natsValuesFile); err != nil {
			t.Errorf("Failed to remove temporary file: %v", err)
		}
	}()

	err := natsHelmDefaultValues(natsValuesFile)
	if err != nil {
		t.Errorf("natsHelmDefaultValues failed: %v", err)
	}

	yamlContent, err := os.ReadFile(natsValuesFile)
	if err != nil {
		t.Errorf("Failed to read the temporary file: %v", err)
	}

	if !containsSubstring(string(yamlContent), "nats:alpine") {
		t.Errorf("natsValuesFile does not contain 'nats:alpine'")
	}

	if !containsSubstring(string(yamlContent), "enabled: true") {
		t.Errorf("natsValuesFile does not contain 'enabled: true' in Jetstream")
	}

	if !containsSubstring(string(yamlContent), "size: 128Mi") {
		t.Errorf("natsValuesFile does not contain 'size: 128Mi' in MemStorage")
	}

	if !containsSubstring(string(yamlContent), "size: 200Mi") {
		t.Errorf("natsValuesFile does not contain 'size: 200Mi' in FileStorage")
	}
}

func TestDgraphHelmDefaultValues(t *testing.T) {
	dgraphValuesFile := "dgraph_values_test.yaml"

	defer func() {
		if err := os.Remove(dgraphValuesFile); err != nil {
			t.Errorf("Failed to remove temporary file: %v", err)
		}
	}()

	err := dgraphHelmDefaultValues(dgraphValuesFile)
	if err != nil {
		t.Errorf("dgraphHelmDefaultValues failed: %v", err)
	}

	yamlContent, err := os.ReadFile(dgraphValuesFile)
	if err != nil {
		t.Errorf("Failed to read the temporary file: %v", err)
	}

	if !containsSubstring(string(yamlContent), "DGRAPH_ALPHA_SECURITY") {
		t.Errorf("dgraphValuesFile does not contain 'DGRAPH_ALPHA_SECURITY'")
	}

	if !containsSubstring(string(yamlContent), "value: whitelist=0.0.0.0/0") {
		t.Errorf("dgraphValuesFile does not contain 'value: whitelist=0.0.0.0/0' in ExtraEnvs")
	}
}

func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}
