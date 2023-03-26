package nodeagent

import (
	"path/filepath"

	"github.com/kube-tarian/tarian/pkg/nodeagent/ebpf/exec2"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Exec2Event struct {
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

type CaptureExec2 struct {
	eventsChan     chan Exec2Event
	shouldClose    bool
	bpfCaptureExec *exec2.BpfExec2
	nodeName       string
}

func NewCaptureExec2() (*CaptureExec2, error) {
	bpfCaptureExec2, err := exec2.NewBpfExec2()

	if err != nil {
		return nil, err
	}

	return &CaptureExec2{
		eventsChan:     make(chan Exec2Event, 1000),
		bpfCaptureExec: bpfCaptureExec2,
	}, nil
}

func (c *CaptureExec2) SetNodeName(name string) {
	c.nodeName = name
}

func (c *CaptureExec2) Start() {
	config, err := rest.InClusterConfig()

	if err != nil {
		panic(err)
	}

	k8sClient := kubernetes.NewForConfigOrDie(config)
	watcher := NewPodWatcher(k8sClient, c.nodeName)
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

		filename := unix.ByteSliceToString(bpfEvt.BinaryFilepath[:])
		comm := filepath.Base(filename)

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

		exec2Event := Exec2Event{
			Pid:               bpfEvt.Pid,
			Comm:              comm,
			Filename:          filename,
			ContainerID:       containerID,
			K8sPodName:        podName,
			K8sPodUID:         podUID,
			K8sNamespace:      namespace,
			K8sPodLabels:      podLabels,
			K8sPodAnnotations: podAnnotations,
		}

		c.eventsChan <- exec2Event
	}
}

func (c *CaptureExec2) Close() {
	c.shouldClose = true
	c.bpfCaptureExec.Close()
}

func (c *CaptureExec2) GetEventsChannel() chan Exec2Event {
	return c.eventsChan
}
