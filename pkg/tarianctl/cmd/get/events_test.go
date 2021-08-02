package get

import (
	"testing"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
)

func TestViolatingProcessesToStringShort(t *testing.T) {
	processes := []*tarianpb.Process{
		{Id: 1, Name: "pause1"},
		{Id: 2, Name: "pause2"},
		{Id: 3, Name: "pause3"},
	}

	str := violatingProcessesToString(processes)
	assert.Equal(t, "1:pause1, 2:pause2, 3:pause3", str)
}

func TestViolatingProcessesToStringLong(t *testing.T) {
	processes := []*tarianpb.Process{
		{Id: 1, Name: "pause1"},
		{Id: 2, Name: "pause2"},
		{Id: 3, Name: "pause3"},
		{Id: 4, Name: "pause4"},
		{Id: 5, Name: "pause5"},
		{Id: 6, Name: "pause6"},
		{Id: 7, Name: "pause7"},
		{Id: 8, Name: "pause8"},
		{Id: 9, Name: "pause9"},
		{Id: 10, Name: "pause10"},
		{Id: 11, Name: "pause11"},
		{Id: 12, Name: "pause12"},
		{Id: 13, Name: "pause13"},
	}

	expected := "1:pause1, 2:pause2, 3:pause3, 4:pause4, 5:pause5, 6:pause6, "
	expected += "7:pause7, 8:pause8, 9:pause9, 10:pause10, 11:pause11, "
	expected += "... 2 more"
	assert.Equal(t, expected, violatingProcessesToString(processes))
}
