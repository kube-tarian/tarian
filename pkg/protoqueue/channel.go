package protoqueue

import "google.golang.org/protobuf/proto"

type ChannelQueue struct {
	channel chan proto.Message
}

func NewChannelQueue() *ChannelQueue {
	channel := make(chan proto.Message, 1000)
	cq := &ChannelQueue{channel: channel}

	return cq
}

func (cq *ChannelQueue) Publish(message proto.Message) error {
	cq.channel <- message

	return nil
}

func (cq *ChannelQueue) NextMessage(message proto.Message) (proto.Message, error) {
	m := <-cq.channel

	return m, nil
}
