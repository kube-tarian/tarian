package add

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchLabelsFromString(t *testing.T) {
	matchLabels := matchLabelsFromString("key1=value1,key2=value2")

	assert.Len(t, matchLabels, 2)
	assert.Equal(t, "key1", matchLabels[0].Key)
	assert.Equal(t, "value1", matchLabels[0].Value)
	assert.Equal(t, "key2", matchLabels[1].Key)
	assert.Equal(t, "value2", matchLabels[1].Value)
}

func TestAllowedProcessesFromString(t *testing.T) {
	rules := allowedProcessesFromString("sleep, pause, tarian.*")
	assert.Len(t, rules, 3)
	assert.Equal(t, "sleep", rules[0].GetRegex())
	assert.Equal(t, "pause", rules[1].GetRegex())
	assert.Equal(t, "tarian.*", rules[2].GetRegex())
}
