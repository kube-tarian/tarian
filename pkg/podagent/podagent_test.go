package podagent

import (
	"testing"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
)

func TestValidateProcesses(t *testing.T) {
	podAgent := NewPodAgent(":50051") // dummy port, not gonna connect

	processes := []*Process{}

	procNginx := &Process{Pid: 1, Name: "nginx"}
	processes = append(processes, procNginx)

	procDockerProxy := &Process{Pid: 2, Name: "docker-proxy"}
	processes = append(processes, procDockerProxy)

	procEvil := &Process{Pid: 3, Name: "evil"}
	processes = append(processes, procEvil)

	nginxStr := "nginx"
	dockerStr := "docker.*"
	constraintNginx := tarianpb.Constraint{AllowedProcesses: []*tarianpb.AllowedProcessRule{{Regex: &nginxStr}}}
	constraintBash := tarianpb.Constraint{AllowedProcesses: []*tarianpb.AllowedProcessRule{{Regex: &dockerStr}}}

	constraints := []*tarianpb.Constraint{&constraintNginx, &constraintBash}
	podAgent.SetConstraints(constraints)

	violations := podAgent.ValidateProcesses(processes)

	assert.Len(t, violations, 1)
	assert.Equal(t, procEvil, violations[3])
}
