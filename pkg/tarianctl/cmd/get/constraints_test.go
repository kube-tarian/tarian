package get

import (
	"testing"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
)

func TestMatchLabelsToString(t *testing.T) {
	matchLabels := []*tarianpb.MatchLabel{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
	}

	str := matchLabelsToString(matchLabels)
	assert.Equal(t, "matchLabels:key1=value1,key2=value2", str)
}

func TestMatchLabelsToStringReturnEmpty(t *testing.T) {
	assert.Equal(t, "", matchLabelsToString([]*tarianpb.MatchLabel{}))
}

func TestAllowedProcessesToString(t *testing.T) {
	pause := "pause"
	sleep := "sleep"

	rules := []*tarianpb.AllowedProcessRule{
		{Regex: &pause},
		{Regex: &sleep},
	}

	assert.Equal(t, "regex:pause,regex:sleep", allowedProcessesToString(rules))
}
