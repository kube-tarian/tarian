package clusteragent

import (
	"testing"

	ca "github.com/kube-tarian/tarian/pkg/clusteragent"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

func FuzzActionMatchesPod(f *testing.F) {
	namespace := "fuzz"

	f.Add("app", "nginx", "extraKey", "extraValue")

	f.Fuzz(func(t *testing.T, key string, value string, key2 string, value2 string) {
		action := &tarianpb.Action{
			Namespace: namespace,
			Selector:  &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: key, Value: value}}},
		}
		pod := &tarianpb.Pod{
			Namespace: namespace,
			Labels:    []*tarianpb.Label{{Key: key, Value: value}, {Key: key2, Value: value2}},
		}

		result := ca.ActionMatchesPod(action, pod)
		if !result {
			t.Errorf("Action matches pod return false, key: %s, value: %s.\n", key, value)
		}
	})
}
