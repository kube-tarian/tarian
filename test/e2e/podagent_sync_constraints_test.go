package e2e

import (
	"fmt"
	"testing"
	"time"
)

func TestPodAgentSyncConstraints(t *testing.T) {
	// setup tarian-server and tarian-cluster-agent, and tell pod-agent to connect to it
	e2eHelper := NewE2eHelper(t)
	e2eHelper.Run()
	defer e2eHelper.Stop()

	podAgent := e2eHelper.podAgent
	podAgent.SyncConstraints()

	// wait for response
	time.Sleep(3 * time.Second)

	fmt.Printf("%v\n", podAgent.GetConstraints())
}
