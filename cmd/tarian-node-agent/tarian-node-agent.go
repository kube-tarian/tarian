package main

import "C"

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aquasecurity/libbpfgo"
	"golang.org/x/sys/unix"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	_ "embed"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed main.bpf.o
var mainBpfEmbedded []byte

type bpfEvent struct {
	Pid      uint32
	Comm     [80]uint8
	Filename [1024]uint8
}

const (
	containerIdx   = "containers-ids"
	containerIDLen = 15
)

var (
	errNoPod = errors.New("object is not a *corev1.Pod")
)

func main() {
	bpfModule, err := libbpfgo.NewModuleFromBuffer(mainBpfEmbedded, "main.bpf.o")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
	defer bpfModule.Close()

	fmt.Println("node agent")

	bpfModule.BPFLoadObject()
	prog, err := bpfModule.GetProgram("enter_execve")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	_, err = prog.AttachTracepoint("syscalls", "sys_enter_execve")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	eventsChannel := make(chan []byte)
	rb, err := bpfModule.InitRingBuf("events", eventsChannel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	// When we get a SIGTERM we should close the tracer so the loop exits.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	shouldClose := false

	go func() {
		<-signals

		shouldClose = true
		defer rb.Stop()
		if err != nil {
			log.Fatalf("error closing events channel: %+v", err)
		}
	}()

	defer rb.Close()
	rb.Start()

	fmt.Printf("%-10s %-30s %s \n", "Pid", "Comm", "Filename")
	fmt.Println(strings.Repeat("-", 100))

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	k8sClient := kubernetes.NewForConfigOrDie(config)
	watcher := NewK8sWatcher(k8sClient)

	for {
		b := <-eventsChannel

		if shouldClose {
			break
		}

		var event bpfEvent
		if err := binary.Read(bytes.NewBuffer(b), binary.LittleEndian, &event); err != nil {
			log.Printf("parsing ringbuf event: %s", err)
			continue
		}

		fmt.Printf("%-10d %-30s %s \n", event.Pid, unix.ByteSliceToString(event.Comm[:]), unix.ByteSliceToString(event.Filename[:]))
		dockerId, err := procsDockerId(event.Pid)
		if err != nil {
			log.Println(err)
			continue
		}

		fmt.Printf("Docker ID = %s \n", dockerId)
		pod := watcher.FindPod(dockerId)
		var podName string
		if pod != nil {
			podName = pod.GetName()
		}
		fmt.Printf("Pod Name = %s \n", podName)

	}
}

// containerIndexFunc index pod by container IDs.
func containerIndexFunc(obj interface{}) ([]string, error) {
	var containerIDs []string
	putContainer := func(fullContainerID string) error {
		if fullContainerID == "" {
			// This is expected if the container hasn't been started. This function
			// will get called again after the container starts, so we just need to
			// be patient.
			return nil
		}
		parts := strings.Split(fullContainerID, "//")
		if len(parts) != 2 {
			return fmt.Errorf("unexpected containerID format, expecting 'docker://<name>', got %q", fullContainerID)
		}
		cid := parts[1]
		if len(cid) > containerIDLen {
			cid = cid[:containerIDLen]
		}
		containerIDs = append(containerIDs, cid)
		return nil
	}

	switch t := obj.(type) {
	case *corev1.Pod:
		for _, container := range t.Status.InitContainerStatuses {
			err := putContainer(container.ContainerID)
			if err != nil {
				return nil, err
			}
		}
		for _, container := range t.Status.ContainerStatuses {
			err := putContainer(container.ContainerID)
			if err != nil {
				return nil, err
			}
		}
		for _, container := range t.Status.EphemeralContainerStatuses {
			err := putContainer(container.ContainerID)
			if err != nil {
				return nil, err
			}
		}
		return containerIDs, nil
	}
	return nil, fmt.Errorf("%w - found %T", errNoPod, obj)
}

type K8sPodWatcher interface {
	FindPod(containerID string) *corev1.Pod
}

type PodWatcher struct {
	podInformer cache.SharedIndexInformer
}

func NewK8sWatcher(k8sClient *kubernetes.Clientset) *PodWatcher {
	k8sInformerFactory := informers.NewSharedInformerFactoryWithOptions(k8sClient, 60*time.Second,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			// Watch local pods only.
			// options.FieldSelector = "spec.nodeName=" + os.Getenv("NODE_NAME")
		}))
	podInformer := k8sInformerFactory.Core().V1().Pods().Informer()
	err := podInformer.AddIndexers(map[string]cache.IndexFunc{
		containerIdx: containerIndexFunc,
	})
	if err != nil {
		// Panic during setup since this should never fail, if it fails is a
		// developer mistake.
		panic(err)
	}

	k8sInformerFactory.Start(wait.NeverStop)
	k8sInformerFactory.WaitForCacheSync(wait.NeverStop)

	fmt.Printf("NewK8sWatcher: num_pods %d\n", len(podInformer.GetStore().ListKeys()))

	return &PodWatcher{podInformer: podInformer}
}

func (watcher *PodWatcher) FindPod(containerID string) *corev1.Pod {
	indexedContainerID := containerID
	if len(containerID) > containerIDLen {
		indexedContainerID = containerID[:containerIDLen]
	}
	objs, err := watcher.podInformer.GetIndexer().ByIndex(containerIdx, indexedContainerID)
	if err != nil {
		return nil
	}
	// If we can't find any pod indexed then fall back to the entire pod list.
	// If we find more than 1 pods indexed also fall back to the entire pod list.
	if len(objs) != 1 {
		return findContainer(containerID, watcher.podInformer.GetStore().List())
	}
	return findContainer(containerID, objs)
}

func findContainer(containerID string, pods []interface{}) *corev1.Pod {
	if containerID == "" {
		return nil
	}
	for _, obj := range pods {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil
		}
		for _, container := range pod.Status.ContainerStatuses {
			parts := strings.Split(container.ContainerID, "//")
			if len(parts) == 2 && strings.HasPrefix(parts[1], containerID) {
				return pod
			}
		}
		for _, container := range pod.Status.InitContainerStatuses {
			parts := strings.Split(container.ContainerID, "//")
			if len(parts) == 2 && strings.HasPrefix(parts[1], containerID) {
				return pod
			}
		}
		for _, container := range pod.Status.EphemeralContainerStatuses {
			parts := strings.Split(container.ContainerID, "//")
			if len(parts) == 2 && strings.HasPrefix(parts[1], containerID) {
				return pod
			}
		}
	}
	return nil
}

func procsDockerId(pid uint32) (string, error) {
	pidstr := fmt.Sprint(pid)
	cgroups, err := ioutil.ReadFile(filepath.Join("/host/proc", pidstr, "cgroup"))
	if err != nil {
		return "", err
	}
	off, _ := procsFindDockerId(string(cgroups))
	return off, nil
}

func procsFindDockerId(cgroups string) (string, int) {
	cgrpPaths := strings.Split(cgroups, "\n")
	for _, s := range cgrpPaths {
		if strings.Contains(s, "pods") || strings.Contains(s, "docker") ||
			strings.Contains(s, "libpod") {
			// Get the container ID and the offset
			container, i := LookupContainerId(s, false, false)
			if container != "" {
				return container, i
			}
		}
	}
	return "", 0
}

const (
	// ContainerIDLength is the standard length of the Container ID
	ContainerIdLength = 64

	// BpfContainerIdLength Minimum 31 chars to assume it is a Container ID
	// in case it was truncated
	BpfContainerIdLength = 31

	DOCKER_ID_LENGTH = 128
)

// ProcsContainerIdOffset Returns the container ID and its offset
// This can fail, better use LookupContainerId to handle different container runtimes.
func ProcsContainerIdOffset(subdir string) (string, int) {
	// If the cgroup subdir contains ":" it means that we are dealing with
	// Linux.CgroupPath where the cgroup driver is cgroupfs
	// https://github.com/opencontainers/runc/blob/main/docs/systemd.md
	// In this case let's split the name and take the last one
	p := strings.LastIndex(subdir, ":") + 1
	fields := strings.Split(subdir, ":")
	idStr := fields[len(fields)-1]

	off := strings.LastIndex(idStr, "-") + 1
	s := strings.Split(idStr, "-")

	return s[len(s)-1], off + p
}

// LookupContainerId returns the container ID as a 31 character string length from the full cgroup path
// cgroup argument is the full cgroup path
// bpfSource is set to true if cgroup was obtained from BPF, otherwise false.
// walkParent if set then walk the parent hierarchy subdirs and try to find the container ID of the process,
//    this will allow to return the container id of services running inside, example: init.service etc.
// Returns the container ID as a string of 31 characters and its offset on the full cgroup path,
// otherwise on errors an empty string and 0 as offset.
func LookupContainerId(cgroup string, bpfSource bool, walkParent bool) (string, int) {
	idTruncated := false
	subDirs := strings.Split(cgroup, "/")
	subdir := subDirs[len(subDirs)-1]

	// Special case for syscont-cgroup-root installed by
	// sysbox nested containers. In this case set with
	// outermost container.
	if strings.Contains(subdir, "syscont-cgroup-root") {
		if len(subDirs) > 4 {
			subdir = subDirs[4]
			walkParent = false
		}
	}

	// Check if the cgroup was obtained from BPF and if the last subdir
	// cgroup length equals DOCKER_ID_LENGTH -1, then:
	// It was probably truncated to DOCKER_ID_LENGTH, let's be flexible
	// try to match containerID without asserting the
	// ContainerIDLength == 64 due to the truncation as it will be less anyway.
	// We trust BPF part that it will always return null terminated
	// DOCKER_ID_LENGTH. For other cases where we read through /proc/
	// strings are not truncated.
	if bpfSource == true && len(subdir) >= DOCKER_ID_LENGTH-1 {
		idTruncated = true
	}

	container, i := ProcsContainerIdOffset(subdir)

	// Let's first check if this was a valid container id, it can be only the id
	// or the id.scope
	// systemd units at the end of a cgroup path can only be a type .scope or .service
	// However if it is a service then it means some service inside the container, if
	// we are interested into it then we should walk the parent subdir with
	// walkParent argument set and get its parent cgroup
	if !strings.HasSuffix(container, "service") &&
		((len(container) >= ContainerIdLength) ||
			(idTruncated == true && len(container) >= BpfContainerIdLength)) {
		// Return first 31 chars. If the string is less than 31 chars
		// it's not a docker ID so skip it. For example docker.server
		// will get here.
		return container[:BpfContainerIdLength], i
	}

	// Podman may set the last subdir to 'container' so let's walk parent subdir
	if strings.Contains(cgroup, "libpod") && container == "container" {
		walkParent = true
	}

	// Should we walk the parent subdirs
	if !walkParent {
		return "", 0
	}

	// Walk the parent subdirs until the first ancestor which is not included
	for j := len(subDirs) - 2; j > 1; j-- {
		container, i = ProcsContainerIdOffset(subDirs[j])
		// Either container ID or the first transient scope unit
		if len(container) == ContainerIdLength || (len(container) > ContainerIdLength && strings.HasSuffix(container, "scope")) {
			// Return first 31 chars. If the string is less than 31 chars
			// it's not a docker ID so skip it. For example docker.server
			// will get here.
			return container[:BpfContainerIdLength], i
		}
	}

	return "", 0
}
