package store

import (
	"testing"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStoreGetAll(t *testing.T) {
	store := NewMemoryConstraintStore()

	{
		exampleConstraint := tarianpb.Constraint{Namespace: "default", Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}
		allowedProcessRegex := "nginx"
		exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}
		store.Add(&exampleConstraint)
	}

	{
		exampleConstraint := tarianpb.Constraint{Namespace: "ns2", Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}
		allowedProcessRegex := "nginx"
		exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}
		store.Add(&exampleConstraint)
	}

	constraints, _ := store.GetAll()
	require.Len(t, constraints, 2)

	assert.Equal(t, "default", constraints[0].Namespace)
	assert.Equal(t, "ns2", constraints[1].Namespace)
}

func TestMemoryStoreFindByNamespace(t *testing.T) {
	store := NewMemoryConstraintStore()

	{
		exampleConstraint := tarianpb.Constraint{Namespace: "default", Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}
		allowedProcessRegex := "nginx"
		exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}
		store.Add(&exampleConstraint)
	}

	{
		exampleConstraint := tarianpb.Constraint{Namespace: "ns2", Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}
		allowedProcessRegex := "nginx"
		exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}
		store.Add(&exampleConstraint)
	}

	constraints, _ := store.FindByNamespace("ns2")
	require.Len(t, constraints, 1)

	assert.Equal(t, "ns2", constraints[0].Namespace)
}
