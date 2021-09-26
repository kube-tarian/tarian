package get

import (
	"testing"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
)

func TestViolatedProcessesToStringShort(t *testing.T) {
	processes := []*tarianpb.Process{
		{Pid: 1, Name: "pause1"},
		{Pid: 2, Name: "pause2"},
		{Pid: 3, Name: "pause3"},
	}

	str := violatedProcessesToString(processes)
	assert.Equal(t, "1:pause1, 2:pause2, 3:pause3", str)
}

func TestViolatedProcessesToStringLong(t *testing.T) {
	processes := []*tarianpb.Process{
		{Pid: 1, Name: "pause1"},
		{Pid: 2, Name: "pause2"},
		{Pid: 3, Name: "pause3"},
		{Pid: 4, Name: "pause4"},
		{Pid: 5, Name: "pause5"},
		{Pid: 6, Name: "pause6"},
		{Pid: 7, Name: "pause7"},
		{Pid: 8, Name: "pause8"},
		{Pid: 9, Name: "pause9"},
		{Pid: 10, Name: "pause10"},
		{Pid: 11, Name: "pause11"},
		{Pid: 12, Name: "pause12"},
		{Pid: 13, Name: "pause13"},
	}

	expected := "1:pause1, 2:pause2, 3:pause3, 4:pause4, 5:pause5, 6:pause6, "
	expected += "7:pause7, 8:pause8, 9:pause9, 10:pause10, 11:pause11, "
	expected += "... 2 more"
	assert.Equal(t, expected, violatedProcessesToString(processes))
}
