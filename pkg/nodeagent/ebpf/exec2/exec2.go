// Package exec2 wraps exec ebpf program and provides simpler abstraction
package exec2

import (
	"bytes"
	"encoding/binary"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"go.uber.org/zap"
)

// Ebpf map item structure
type BpfExec2EventData struct {
	Pid            uint32
	Tgid           uint32
	UID            uint32
	Gid            uint32
	SyscallNr      int32
	Comm           [16]uint8
	Cwd            [32]uint8
	BinaryFilepath [256]uint8
	UserComm       [256][256]uint8
}

type BpfExec2 struct {
	shouldClose bool

	bpfEventsChan  chan []byte
	execEventsChan chan *BpfExec2EventData

	logger *zap.SugaredLogger

	bpfLink       link.Link
	ringbufReader *ringbuf.Reader
}

//go:generate bpf2go -cc clang -cflags $BPF_CFLAGS bpf index.bpf.c -- -I../../../../output -I../../../../output/bpf
func NewBpfExec2() (*BpfExec2, error) {
	l, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	//Loads ebpf objects(programs, maps)
	ebpfColl := bpfObjects{}
	err = loadBpfObjects(&ebpfColl, nil)
	if err != nil {
		return nil, err
	}

	//Attach program to a hook
	hook, err := link.Tracepoint("syscalls", "sys_enter_execve", ebpfColl.EbpfExecve, nil)
	if err != nil {
		return nil, err
	}

	//Create ringbuffer map reader
	ringbufReader, err := ringbuf.NewReader(ebpfColl.Event)
	if err != nil {
		return nil, err
	}

	execEventsChan := make(chan *BpfExec2EventData, 1000)
	b := &BpfExec2{
		bpfEventsChan:  make(chan []byte, 1000),
		execEventsChan: execEventsChan,
		logger:         l.Sugar(),
		bpfLink:        hook,
		ringbufReader:  ringbufReader,
	}

	return b, nil
}

func (b *BpfExec2) SetLogger(l *zap.SugaredLogger) {
	b.logger = l
}

func (b *BpfExec2) Start() {
	for {
		if b.shouldClose {
			break
		}

		record, err := b.ringbufReader.Read()
		if err != nil {
			b.logger.Errorw("error reading ringbufReader", "err", err)
		}

		var row BpfExec2EventData
		err = binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &row)
		if err != nil {
			b.logger.Errorw("error parsing ringbuf event", "err", err)
		}

		b.execEventsChan <- &row
	}
}

func (b *BpfExec2) Close() {
	b.shouldClose = true
	b.bpfLink.Close()

	if b.ringbufReader != nil {
		b.ringbufReader.Close()
	}
}

func (b *BpfExec2) GetExecEventsChannel() chan *BpfExec2EventData {
	return b.execEventsChan
}
