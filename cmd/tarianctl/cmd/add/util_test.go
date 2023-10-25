package add

import (
	"testing"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
)

func TestMatchLabelsFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []*tarianpb.MatchLabel
	}{
		{
			name:  "MatchLabelsFromString should return valid Key=Value pairs",
			input: []string{"key1=value1", "key2=value2"},
			expected: []*tarianpb.MatchLabel{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
		},
		{
			name:  "MatchLabelsFromString should return valid Key=Value pairs and ignore invalid labels",
			input: []string{"key1=value1", "key2=value2", "invalid"},
			expected: []*tarianpb.MatchLabel{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
		},
		{
			name:     "MatchLabelsFromString should return nil if no valid Key=Value pairs are found",
			input:    []string{"invalid"},
			expected: nil,
		},
		{
			name:     "MatchLabelsFromString should return nil if input is nil",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchLabelsFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
