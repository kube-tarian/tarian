// Package protoqueue provides queue interface and various implementation
package protoqueue

import "google.golang.org/protobuf/proto"

type QueuePublisher interface {
	Publish(message proto.Message) error
}

type QueueSubscriber interface {
	NextMessage(proto.Message) (proto.Message, error)
}
