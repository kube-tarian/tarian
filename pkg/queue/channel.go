package queue

type ChannelQueue struct {
	channel chan any
}

func NewChannelQueue() *ChannelQueue {
	channel := make(chan any, 1000)
	cq := &ChannelQueue{channel: channel}

	return cq
}

func (cq *ChannelQueue) Publish(queuedMessage any) error {
	cq.channel <- queuedMessage

	return nil
}

func (cq *ChannelQueue) NextMessage() (any, error) {
	queuedMessage := <-cq.channel

	return queuedMessage, nil
}
