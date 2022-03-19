package clusteragent

import (
	"testing"
	"time"

	"github.com/falcosecurity/falcosidekick/types"
	"github.com/kube-tarian/tarian/pkg/clusteragent"
	v1 "k8s.io/api/core/v1"
)

func FuzzNewEventFromTarianRuleSpawnedProcessAlert(f *testing.F) {
	seedPodName := "nginx-abcdef-ghijklm"
	seedNamespace := "ns1"
	seedProcPid := 1
	seedProcName := "nginx"
	seedLabelKey := "app"
	seedLabelValue := "nginx"
	seedExtraKey := "extraKey"
	seedExtraValue := "extraValue"

	f.Add(seedPodName, seedNamespace, seedProcPid, seedProcName, seedLabelKey, seedLabelValue, seedExtraKey, seedExtraValue)

	f.Fuzz(func(t *testing.T, podName string, namespace string, procPid int, procName string, labelKey string, labelValue string, extraKey string, extraValue string) {

		falcoPayload := &types.FalcoPayload{Time: time.Time{}, OutputFields: make(map[string]interface{}, 0)}
		falcoPayload.OutputFields["k8s.pod.name"] = podName
		falcoPayload.OutputFields["k8s.ns.name"] = namespace
		falcoPayload.OutputFields["proc.pid"] = procPid
		falcoPayload.OutputFields["proc.name"] = procName

		pod := &v1.Pod{}
		pod.ObjectMeta.Labels = make(map[string]string)
		pod.ObjectMeta.Labels[labelKey] = labelValue
		pod.ObjectMeta.Labels[extraKey] = extraValue

		clusteragent.NewEventFromTarianRuleSpawnedProcessAlert(falcoPayload, pod)
	})
}

func FuzzNewEventFromGenericFalcoAlert(f *testing.F) {
	seedPodName := "nginx-abcdef-ghijklm"
	seedNamespace := "ns1"
	seedLabelKey := "app"
	seedLabelValue := "nginx"
	seedExtraKey := "extraKey"
	seedExtraValue := "extraValue"

	f.Add(seedPodName, seedNamespace, seedLabelKey, seedLabelValue, seedExtraKey, seedExtraValue)

	f.Fuzz(func(t *testing.T, podName string, namespace string, labelKey string, labelValue string, extraKey string, extraValue string) {

		falcoPayload := &types.FalcoPayload{Time: time.Time{}, OutputFields: make(map[string]interface{}, 0)}
		falcoPayload.OutputFields["k8s.pod.name"] = podName
		falcoPayload.OutputFields["k8s.ns.name"] = namespace

		pod := &v1.Pod{}
		pod.ObjectMeta.Labels = make(map[string]string)
		pod.ObjectMeta.Labels[labelKey] = labelValue
		pod.ObjectMeta.Labels[extraKey] = extraValue

		clusteragent.NewEventFromGenericFalcoAlert(falcoPayload, pod)
	})
}
