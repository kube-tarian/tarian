package protoqueue

import "google.golang.org/protobuf/proto"

// QueuePublisher is an interface for components that publish messages to a queue.
type QueuePublisher interface {
	// Publish publishes the given protobuf message to the queue.
	Publish(message proto.Message) error
}

// QueueSubscriber is an interface for components that consume messages from a queue.
type QueueSubscriber interface {
	// NextMessage retrieves the next message from the queue and unmarshals it into the provided protobuf message.
	NextMessage(proto.Message) (proto.Message, error)
}
