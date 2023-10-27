package kubeclient

import "github.com/sirupsen/logrus"

type fakeClient struct {
	logger *logrus.Logger
}

// NewFakeClient returns a new fake Kubernetes client.
func NewFakeClient(logger *logrus.Logger) Client {
	return &fakeClient{
		logger: logger,
	}
}

// ExecPodWithOneContainer implements Client.
func (f *fakeClient) ExecPodWithOneContainer(namespace string, podName string, cmd []string) (string, error) {
	f.logger.Infof("Executing command %s in pod %s in namespace %s", cmd, podName, namespace)
	return "", nil
}

// GetPodName implements Client.
func (f *fakeClient) GetPodName(namespace string, labelSelector string) (string, error) {
	f.logger.Infof("Getting pod name in namespace %s with label selector %s", namespace, labelSelector)
	return "", nil
}

// WaitForPodsToBeReady implements Client.
func (f *fakeClient) WaitForPodsToBeReady(namespace string, labelSelector string) error {
	f.logger.Infof("Waiting for pods in namespace %s with label selector %s to be ready", namespace, labelSelector)
	return nil
}
