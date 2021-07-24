package e2e

import (
	"testing"

	"github.com/devopstoday11/tarian/pkg/podagent"
)

func TestPodAgentReportViolationsToClusterAgent(t *testing.T) {
	// Setup tarian-server and tarian-cluster-agent, and tell pod-agent to connect to it
	e2eHelper := NewE2eHelper(t)
	e2eHelper.PrepareDatabase()
	e2eHelper.Run()
	// defer e2eHelper.DropDatabase()
	defer e2eHelper.Stop()

	violatingProcesses := make(map[int32]*podagent.Process)
	violatingProcesses[1] = &podagent.Process{Pid: 1, Name: "unknown_process"}
	violatingProcesses[2] = &podagent.Process{Pid: 2, Name: "unknown_process_2"}

	// Report violations to cluster agent
	podAgent := e2eHelper.podAgent
	podAgent.ReportViolationsToClusterAgent(violatingProcesses)

	// TODO: verify get events
}
