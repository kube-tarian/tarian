package helm

import "github.com/sirupsen/logrus"

type fakeClient struct {
	logger *logrus.Logger
}

// NewFakeClient returns a new fake Helm client.
func NewFakeClient(logger *logrus.Logger) Client {
	return &fakeClient{
		logger: logger,
	}
}

// AddRepo implements Client.
func (f *fakeClient) AddRepo(name string, url string) error {
	f.logger.Infof("Adding Helm repository %s with URL %s", name, url)
	return nil
}

// Install implements Client.
func (f *fakeClient) Install(name string, chart string, namespace string, valuesFiles []string, version string, setArgs []string) error {
	f.logger.Infof("Installing Helm chart %s with name %s in namespace %s", chart, name, namespace)
	return nil
}

// UnInstall implements Client.
func (f *fakeClient) Uninstall(name string, namespace string) error {
	f.logger.Infof("Uninstalling Helm chart %s in namespace %s", name, namespace)
	return nil
}
