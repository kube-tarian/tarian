package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/devopstoday11/tarian/pkg/podagent"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestPodAgentReportViolationsToClusterAgent(t *testing.T) {
	// Setup tarian-server and tarian-cluster-agent, and tell pod-agent to connect to it
	e2eHelper := NewE2eHelper(t)
	e2eHelper.PrepareDatabase()
	e2eHelper.Run()
	defer e2eHelper.DropDatabase()
	defer e2eHelper.Stop()

	violatingProcesses := make(map[int32]*podagent.Process)
	violatingProcesses[1] = &podagent.Process{Pid: 1, Name: "unknown_process"}
	violatingProcesses[2] = &podagent.Process{Pid: 2, Name: "unknown_process_2"}

	// Report violations to cluster agent
	podAgent := e2eHelper.podAgent
	podAgent.ReportViolationsToClusterAgent(violatingProcesses)

	// Verify get events
	grpcConn, err := grpc.Dial(":"+e2eServerPort, grpc.WithInsecure())
	require.Nil(t, err)
	eventClient := tarianpb.NewEventClient(grpcConn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	getEventsResponse, err := eventClient.GetEvents(ctx, &tarianpb.GetEventsRequest{})
	require.Nil(t, err)

	events := getEventsResponse.GetEvents()
	require.NotNil(t, events)
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, "violation", event.Type)
	assert.Len(t, event.GetTargets(), 1)

	pids := []int32{}
	for _, process := range event.GetTargets()[0].GetViolatingProcesses() {
		pids = append(pids, process.GetPid())
	}

	assert.Contains(t, pids, int32(1))
	assert.Contains(t, pids, int32(2))
}
