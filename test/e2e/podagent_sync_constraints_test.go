package e2e

import (
	"fmt"
	"testing"
	"time"
)

func TestPodAgentSyncConstraints(t *testing.T) {
	// Setup tarian-server and tarian-cluster-agent, and tell pod-agent to connect to it
	e2eHelper := NewE2eHelper(t)
	e2eHelper.PrepareDatabase()
	e2eHelper.Run()
	defer e2eHelper.Stop()
	defer e2eHelper.DropDatabase()

	podAgent := e2eHelper.podAgent
	podAgent.SyncConstraints()

	// Retry up to 3 times if intermittent network error occurs
	retry := 0
	for retry < 3 && len(podAgent.GetConstraints()) == 0 {
		podAgent.SyncConstraints()

		retry++
		time.Sleep(2 * time.Second)
	}

	// Wait for response

	fmt.Printf("%v\n", podAgent.GetConstraints())
}
