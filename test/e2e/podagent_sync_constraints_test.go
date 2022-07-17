package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestPodAgentSyncConstraints(t *testing.T) {
	// Setup tarian-server and tarian-cluster-agent, and tell pod-agent to connect to it
	e2eHelper := NewE2eHelper(t)
	e2eHelper.PrepareDatabase()
	e2eHelper.Run()
	defer e2eHelper.DropDatabase()
	defer e2eHelper.Stop()

	// Add constraints
	grpcConn, err := grpc.Dial(":"+e2eServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err)
	configClient := tarianpb.NewConfigClient(grpcConn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configClient.AddConstraint(ctx, createConstraintRequest("default", "nginx", "nginx.*", []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}))
	configClient.AddConstraint(ctx, createConstraintRequest("default2", "nginx", "nginx.*", []*tarianpb.MatchLabel{{Key: "app2", Value: "nginx2"}}))

	podAgent := e2eHelper.podAgent
	podAgent.SetNamespace("default")
	podAgent.SetPodLabels([]*tarianpb.Label{{Key: "app", Value: "nginx"}, {Key: "pod-template-hash", Value: "abcdef"}})
	podAgent.SyncConstraints()

	// Retry up to 3 times if intermittent network error occurs
	retry := 0
	for retry < 3 && len(podAgent.GetConstraints()) == 0 {
		podAgent.SyncConstraints()

		retry++
		time.Sleep(2 * time.Second)
	}

	// Verify constraints
	assert.Len(t, podAgent.GetConstraints(), 1)
	constraint := podAgent.GetConstraints()[0]
	assert.Equal(t, "default", constraint.GetNamespace())
	assert.Equal(t, "app", constraint.GetSelector().GetMatchLabels()[0].GetKey())
	assert.Equal(t, "nginx", constraint.GetSelector().GetMatchLabels()[0].GetValue())
	assert.Equal(t, "nginx.*", constraint.GetAllowedProcesses()[0].GetRegex())
}

func createConstraintRequest(namespace string, name string, allowedProcessRegex string, labels []*tarianpb.MatchLabel) *tarianpb.AddConstraintRequest {
	constraint := &tarianpb.Constraint{Namespace: namespace, Name: name, Selector: &tarianpb.Selector{MatchLabels: labels}}
	constraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}

	return &tarianpb.AddConstraintRequest{Constraint: constraint}
}
