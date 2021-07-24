package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestPodAgentSyncConstraints(t *testing.T) {
	// Setup tarian-server and tarian-cluster-agent, and tell pod-agent to connect to it
	e2eHelper := NewE2eHelper(t)
	e2eHelper.PrepareDatabase()
	e2eHelper.Run()
	defer e2eHelper.DropDatabase()
	defer e2eHelper.Stop()

	// Add constraints
	grpcConn, err := grpc.Dial(":"+e2eServerPort, grpc.WithInsecure())
	require.Nil(t, err)
	configClient := tarianpb.NewConfigClient(grpcConn)

	allowedProcessRegex := "nginx.*"
	constraint1 := &tarianpb.Constraint{Namespace: "default", Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}
	constraint1.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}

	addConstraintRequest := &tarianpb.AddConstraintRequest{Constraint: constraint1}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	configClient.AddConstraint(ctx, addConstraintRequest)

	podAgent := e2eHelper.podAgent
	podAgent.SyncConstraints()

	// Retry up to 3 times if intermittent network error occurs
	retry := 0
	for retry < 3 && len(podAgent.GetConstraints()) == 0 {
		podAgent.SyncConstraints()

		retry++
		time.Sleep(2 * time.Second)
	}

	// Verify constraints

	fmt.Printf("%v\n", podAgent.GetConstraints())

}
