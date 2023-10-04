package get

import (
	"testing"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

func TestMatchLabelsToString(t *testing.T) {
	tests := []struct {
		name     string
		labels   []*tarianpb.MatchLabel
		expected string
	}{
		{
			name:     "Empty Labels",
			labels:   []*tarianpb.MatchLabel{},
			expected: "",
		},
		{
			name: "Single Label",
			labels: []*tarianpb.MatchLabel{
				{Key: "app", Value: "nginx"},
			},
			expected: "matchLabels:app=nginx",
		},
		{
			name: "Multiple Labels",
			labels: []*tarianpb.MatchLabel{
				{Key: "app", Value: "nginx"},
				{Key: "env", Value: "production"},
			},
			expected: "matchLabels:app=nginx,env=production",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := matchLabelsToString(test.labels)
			if result != test.expected {
				t.Errorf("Expected: %s, Got: %s", test.expected, result)
			}
		})
	}
}
