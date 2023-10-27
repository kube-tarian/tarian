package helm

// Client represents a Helm client for managing Helm charts.
type Client interface {
	// AddRepo adds a Helm repository.
	AddRepo(name string, url string) error
	// UpdateRepo updates a Helm repository.
	Install(name string, chart string, namespace string, valuesFiles []string, version string, setArgs []string) error
}
