package webhookserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/inject-pod-agent,mutating=true,sideEffects=none,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,admissionReviewVersions=v1,name=pod-agent.k8s.tarian.dev

// PodAgentInjector represents an admission webhook injector for pod agents.
type PodAgentInjector struct {
	Client  client.Client
	decoder *admission.Decoder
	config  PodAgentContainerConfig
	logger  *logrus.Logger
}

// PodAgentContainerConfig defines the configuration for a pod agent container.
type PodAgentContainerConfig struct {
	Name        string
	Image       string
	LogEncoding string
	Host        string
	Port        string
}

const (
	// ThreatScanAnnotation is the annotation used to enable threat scanning for a pod.
	ThreatScanAnnotation = "pod-agent.k8s.tarian.dev/threat-scan"

	// RegisterAnnotation is the annotation used to mark a pod for agent registration.
	RegisterAnnotation = "pod-agent.k8s.tarian.dev/register"

	// FileValidationIntervalAnnotation is the annotation used to specify file validation interval for a pod.
	FileValidationIntervalAnnotation = "pod-agent.k8s.tarian.dev/file-validation-interval"

	// RegisterFileIgnorePathsAnnotation is the annotation used to specify paths to ignore during agent registration.
	RegisterFileIgnorePathsAnnotation = "pod-agent.k8s.tarian.dev/register-file-ignore-paths"
)

// Handle processes a webhook request, adding a sidecar container to a Pod based on annotations.
// It decodes the incoming request, checks for relevant annotations, and adds the sidecar container accordingly.
// If no annotations are found or if the specified sidecar container already exists, it allows the request.
// The added sidecar container is responsible for threat scanning or registration, depending on the annotations.
// Parameters:
//   - ctx: The context for the request.
//   - req: The admission request containing the Pod to be modified.
//
// Returns:
//   - admission.Response: The response indicating the result of the webhook request.
func (p *PodAgentInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	p.logger.Debug("handling a webhook request")

	pod := &corev1.Pod{}

	err := p.decoder.Decode(req, pod)
	if err != nil {
		p.logger.WithError(err).Error("error while decoding webhook request payload")
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil {
		p.logger.WithField("pod_name", pod.GetObjectMeta().GetName()).Debug("not injecting container because no annotation found")
		return admission.Allowed("no annotation found")
	}

	_, threatScanAnnotationPresent := pod.Annotations[ThreatScanAnnotation]
	registerAnnotationValue, registerAnnotationPresent := pod.Annotations[RegisterAnnotation]
	registerFileIgnorePathsAnnotationValue, registerFileIgnorePathsAnnotationPresent := pod.Annotations[RegisterFileIgnorePathsAnnotation]

	if !threatScanAnnotationPresent && !registerAnnotationPresent {
		p.logger.WithField("pod_name", pod.GetObjectMeta().GetName()).Debug("not injecting container because no tarian annotation found")
		return admission.Allowed("annotation " + ThreatScanAnnotation + " or " + RegisterAnnotation + " not found")
	}

	for _, c := range pod.Spec.Containers {
		if c.Name == p.config.Name {
			return admission.Allowed("container with name " + p.config.Name + " already exists")
		}
	}

	podInfoPath := "/etc/podinfo"
	// mount all volumes into pod agent
	volumeMounts := []corev1.VolumeMount{{Name: "podinfo", MountPath: podInfoPath}}
	mountNamesAdded := make(map[string]struct{})
	for _, c := range pod.Spec.Containers {
		for _, vm := range c.VolumeMounts {
			if _, found := mountNamesAdded[vm.Name]; found {
				continue
			}

			volumeMounts = append(volumeMounts, vm)
			mountNamesAdded[vm.Name] = struct{}{}
		}
	}

	podAgentCommand := "threat-scan"
	if registerAnnotationPresent {
		podAgentCommand = "register"
	}

	podAgentArgs := []string{
		"--log-formatter=" + p.config.LogEncoding,
		podAgentCommand,
		"--host=" + p.config.Host,
		"--port=" + p.config.Port,
		"--namespace=$(NAMESPACE)",
		"--pod-name=$(POD_NAME)",
		"--pod-uid=$(POD_UID)",
		"--pod-labels-file=/etc/podinfo/labels",
	}

	ignorePaths := []string{}

	if registerFileIgnorePathsAnnotationPresent {
		ignorePaths = append(ignorePaths, strings.Split(registerFileIgnorePathsAnnotationValue, ",")...)
	}

	if registerAnnotationPresent {
		podAgentArgs = append(podAgentArgs, "--register-rules="+registerAnnotationValue)

		mountPathsAdded := make(map[string]struct{})
		mountPaths := []string{}
		for _, vm := range volumeMounts {
			if _, found := mountPathsAdded[vm.MountPath]; found {
				continue
			}

			mountNamesAdded[vm.MountPath] = struct{}{}

			if vm.MountPath != podInfoPath && vm.MountPath != "/var/run/secrets/kubernetes.io/serviceaccount" {
				mountPaths = append(mountPaths, vm.MountPath)

				// Ignore config map links
				ignorePaths = append(ignorePaths, vm.MountPath+"/..*")
				ignorePaths = append(ignorePaths, vm.MountPath+"/..**/*")
			}
		}

		podAgentArgs = append(podAgentArgs, "--register-file-paths="+strings.Join(mountPaths, ","))
	}

	if len(ignorePaths) > 0 {
		podAgentArgs = append(podAgentArgs, "--register-file-ignore-paths="+strings.Join(ignorePaths, ","))
	}

	if fileValidationInterval, ok := pod.Annotations[FileValidationIntervalAnnotation]; ok {
		podAgentArgs = append(podAgentArgs, "--file-validation-interval="+fileValidationInterval)
	}

	podAgentContainer := corev1.Container{
		Name:  p.config.Name,
		Image: p.config.Image,
		Env: []corev1.EnvVar{
			{
				Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
			},
			{
				Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
			},
			{
				Name: "POD_UID", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"}},
			},
		},
		Args:         podAgentArgs,
		VolumeMounts: volumeMounts,
	}
	pod.Spec.Containers = append(pod.Spec.Containers, podAgentContainer)

	podInfoVolume := corev1.Volume{
		Name: "podinfo",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{{Path: "labels", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.labels"}}},
			},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, podInfoVolume)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"pod_name": pod.GetObjectMeta().GetName(),
			"err":      err,
		}).Error("error while marshalling pod into json")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	p.logger.WithField("pod_name", pod.GetObjectMeta().GetName()).Debug("responding webhook with a sidecar container")
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectDecoder injects the decoder.
func (p *PodAgentInjector) InjectDecoder(d *admission.Decoder) error {
	p.decoder = d
	return nil
}
