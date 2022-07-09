package nodeagent

import (
	"github.com/kube-tarian/tarian/pkg/nodeagent/ebpf"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ExecEvent struct {
	Pid               uint32
	Comm              string
	Filename          string
	ContainerID       string
	K8sPodUID         string
	K8sPodName        string
	K8sNamespace      string
	K8sPodLabels      map[string]string
	K8sPodAnnotations map[string]string
}

type CaptureExec struct {
	eventsChan     chan ExecEvent
	shouldClose    bool
	bpfCaptureExec *ebpf.BpfCaptureExec
}

func NewCaptureExec() (*CaptureExec, error) {
	bpfCaptureExec, err := ebpf.NewBpfCaptureExec()
	if err != nil {
		return nil, err
	}

	return &CaptureExec{
		eventsChan:     make(chan ExecEvent, 1000),
		bpfCaptureExec: bpfCaptureExec,
	}, nil
}

func (c *CaptureExec) Start() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	k8sClient := kubernetes.NewForConfigOrDie(config)
	watcher := NewPodWatcher(k8sClient)
	watcher.Start()

	go c.bpfCaptureExec.Start()

	bpfExecEventsChan := c.bpfCaptureExec.GetExecEventsChannel()
	for {
		bpfEvt := <-bpfExecEventsChan

		if c.shouldClose {
			break
		}

		containerID, err := procsContainerID(bpfEvt.Pid)
		if err != nil {
			continue
		}

		pod := watcher.FindPod(containerID)
		var podName string
		var podUID string
		var namespace string
		var podLabels map[string]string
		var podAnnotations map[string]string
		if pod != nil {
			podName = pod.GetName()
			podUID = string(pod.GetUID())
			namespace = pod.GetNamespace()
			podLabels = pod.GetLabels()
			podAnnotations = pod.GetAnnotations()
		}

		execEvent := ExecEvent{
			Pid:               bpfEvt.Pid,
			Comm:              unix.ByteSliceToString(bpfEvt.Comm[:]),
			Filename:          unix.ByteSliceToString(bpfEvt.Filename[:]),
			ContainerID:       containerID,
			K8sPodName:        podName,
			K8sPodUID:         podUID,
			K8sNamespace:      namespace,
			K8sPodLabels:      podLabels,
			K8sPodAnnotations: podAnnotations,
		}

		c.eventsChan <- execEvent
	}
}

func (c *CaptureExec) Close() {
	c.shouldClose = true
	c.bpfCaptureExec.Close()
}

func (c *CaptureExec) GetEventsChannel() chan ExecEvent {
	return c.eventsChan
}
