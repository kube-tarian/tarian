package podagent

import psutil "github.com/shirou/gopsutil/process"

type Process struct {
	Pid  int32
	Name string
}

func (p *Process) GetPid() int32 {
	return p.Pid
}

func (p *Process) GetName() string {
	return p.Name
}

func NewProcessFromPsutil(p *psutil.Process) *Process {
	name, _ := p.Name()

	return &Process{
		Pid:  p.Pid,
		Name: name,
	}
}

func NewProcessesFromPsutil(ps []*psutil.Process) []*Process {
	s := make([]*Process, len(ps))

	for i, p := range ps {
		s[i] = NewProcessFromPsutil(p)
	}

	return s
}
