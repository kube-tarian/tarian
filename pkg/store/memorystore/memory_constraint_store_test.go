package memorystore

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

	// Order is not guaranteed. So, need to use contains.
	namespaces := []string{}
	for _, constraint := range constraints {
		namespaces = append(namespaces, constraint.Namespace)
	}

	assert.Contains(t, namespaces, "default")
	assert.Contains(t, namespaces, "ns2")
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
