package podagent

import (
	"testing"

	psutil "github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProcessFromPsutil(t *testing.T) {
	psutilProcesses, err := psutil.Processes()
	require.Nil(t, err)

	psutilProcess := psutilProcesses[0]

	name, err := psutilProcess.Name()
	require.Nil(t, err)

	process := NewProcessFromPsutil(psutilProcess)

	assert.Equal(t, psutilProcess.Pid, process.GetPid())
	assert.Equal(t, name, process.GetName())
}

func TestNewProcessesFromPsutil(t *testing.T) {
	psutilProcesses, err := psutil.Processes()
	require.Nil(t, err)

	processes := NewProcessesFromPsutil(psutilProcesses)
	require.Equal(t, len(psutilProcesses), len(processes))

	psutilProcess := psutilProcesses[len(psutilProcesses)-1]
	name, err := psutilProcess.Name()
	require.Nil(t, err)

	process := processes[len(processes)-1]
	assert.Equal(t, psutilProcess.Pid, process.GetPid())
	assert.Equal(t, name, process.GetName())
}
