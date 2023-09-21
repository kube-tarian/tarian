package protoqueue

import "google.golang.org/protobuf/proto"

// ChannelQueue represents a message queue based on Go channels.
type ChannelQueue struct {
	channel chan proto.Message // A buffered channel for storing protobuf messages
}

// NewChannelQueue creates and returns a new ChannelQueue instance.
func NewChannelQueue() *ChannelQueue {
	// Create a buffered channel with a capacity of 1000 to store protobuf messages.
	channel := make(chan proto.Message, 1000)
	cq := &ChannelQueue{channel: channel}

	return cq
}

// Publish adds a protobuf message to the message queue.
//
// Parameters:
//   - message: The protobuf message to be added to the queue.
//
// Returns:
//   - error: An error if there is any issue adding the message to the queue.
func (cq *ChannelQueue) Publish(message proto.Message) error {
	// Put the provided protobuf message into the channel, effectively adding it to the queue.
	cq.channel <- message

	return nil
}

// NextMessage retrieves the next available protobuf message from the queue.
//
// Parameters:
//   - message: A protobuf message that will be populated with the retrieved message.
//
// Returns:
//   - proto.Message: The retrieved protobuf message.
//   - error: An error if there is any issue retrieving the message from the queue.
func (cq *ChannelQueue) NextMessage(_ proto.Message) (proto.Message, error) {
	// Retrieve the next message from the channel and assign it to the provided 'message' parameter.
	return <-cq.channel, nil
}
