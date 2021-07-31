package webhookserver

import (
	"context"
	"encoding/json"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/inject-pod-agent,mutating=true,sideEffects=none,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,admissionReviewVersions=v1,name=pod-agent.k8s.tarian.io

type PodAgentInjector struct {
	Client  client.Client
	decoder *admission.Decoder
}

// podAnnotator adds an annotation to every incoming pods.
func (a *PodAgentInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if pod.Annotations == nil {
		return admission.Allowed("no annotation found")
	}

	if _, ok := pod.Annotations["pod-agent.k8s.tarian.io/inject"]; !ok {
		return admission.Allowed("annotation pod-agent.k8s.tarian.io/inject not found")
	}

	sidecarContainer := corev1.Container{
		Name:         "tarian-pod-agent",
		Image:        "localhost:5000/tarian-pod-agent:latest",
		Env:          []corev1.EnvVar{{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}}},
		Args:         []string{"--log-encoding=json", "run", "--host=tarian-cluster-agent.tarian-system.svc", "--port=80", "--namespace=$(NAMESPACE)", "--pod-labels-file==/etc/podinfo/labels"},
		VolumeMounts: []corev1.VolumeMount{{Name: "podinfo", MountPath: "/etc/podinfo"}},
	}
	pod.Spec.Containers = append(pod.Spec.Containers, sidecarContainer)
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: "podinfo", VolumeSource: corev1.VolumeSource{DownwardAPI: &corev1.DownwardAPIVolumeSource{Items: []corev1.DownwardAPIVolumeFile{{Path: "labels", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.labels"}}}}}})

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectDecoder injects the decoder.
func (a *PodAgentInjector) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
