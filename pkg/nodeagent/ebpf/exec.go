package ebpf

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/aquasecurity/libbpfgo"
	"github.com/sirupsen/logrus"

	_ "embed"
)

var bpfObjName = "capture_exec.bpf.o"

//go:embed capture_exec.bpf.o
var captureExecBpfObj []byte

// BpfExecEvent represents the structure of an eBPF execution event.
type BpfExecEvent struct {
	Pid      uint32
	Comm     [80]uint8
	Filename [1024]uint8
}

// BpfCaptureExec handles the capturing and processing of eBPF events.
type BpfCaptureExec struct {
	shouldClose bool

	bpfEventsChan  chan []byte
	execEventsChan chan BpfExecEvent

	bpfModule     *libbpfgo.Module
	bpfProg       *libbpfgo.BPFProg
	bpfRingBuffer *libbpfgo.RingBuffer

	logger *logrus.Logger
}

// NewBpfCaptureExec creates a new BpfCaptureExec instance for capturing and processing eBPF events.
// It takes a logger as input.
//
// Parameters:
//   - logger: A logger instance for logging.
//
// Returns:
//   - *BpfCaptureExec: A new instance of BpfCaptureExec.
//   - error: An error if loading the eBPF object or initializing the capture fails.
func NewBpfCaptureExec(logger *logrus.Logger) (*BpfCaptureExec, error) {
	b := &BpfCaptureExec{
		bpfEventsChan:  make(chan []byte, 1000),
		execEventsChan: make(chan BpfExecEvent, 1000),
		logger:         logger,
	}

	// Load the eBPF object and initialize the capture.
	err := b.loadBpfObject()
	if err != nil {
		return nil, fmt.Errorf("NewBpfCaptureExec: failed to load bpf object: %w", err)
	}

	return b, nil
}

// loadBpfObject loads the eBPF object and sets up the eBPF program and ring buffer.
// It returns an error if any of these steps fails.
func (b *BpfCaptureExec) loadBpfObject() error {
	var err error
	b.bpfModule, err = libbpfgo.NewModuleFromBuffer(captureExecBpfObj, bpfObjName)
	if err != nil {
		return err
	}

	b.bpfModule.BPFLoadObject()

	b.bpfRingBuffer, err = b.bpfModule.InitRingBuf("events", b.bpfEventsChan)
	if err != nil {
		return err
	}

	b.bpfProg, err = b.bpfModule.GetProgram("enter_execve")
	if err != nil {
		return err
	}

	_, err = b.bpfProg.AttachTracepoint("syscalls", "sys_enter_execve")
	if err != nil {
		return err
	}

	return nil
}

// Start starts the eBPF ring buffer and processes captured events.
// It continues processing events until the shouldClose flag is set to true.
func (b *BpfCaptureExec) Start() {
	b.bpfRingBuffer.Start()

	for {
		evt := <-b.bpfEventsChan

		if b.shouldClose {
			break
		}

		var bpfExecEvent BpfExecEvent
		if err := binary.Read(bytes.NewBuffer(evt), binary.LittleEndian, &bpfExecEvent); err != nil {
			b.logger.WithError(err).Error("error parsing ringbuf event")
			continue
		}

		b.execEventsChan <- bpfExecEvent
	}
}

// Close stops the eBPF ring buffer and closes the eBPF module.
func (b *BpfCaptureExec) Close() {
	b.shouldClose = true
	b.bpfRingBuffer.Close()
	b.bpfModule.Close()
}

// GetExecEventsChannel returns the channel for receiving eBPF execution events.
func (b *BpfCaptureExec) GetExecEventsChannel() chan BpfExecEvent {
	return b.execEventsChan
}
