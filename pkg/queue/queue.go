// Package queue provides queue interface and various implementation
package queue

type QueuePublisher interface {
	Publish(queuedMessage any) error
}

type QueueSubscriber interface {
	NextMessage() (any, error)
}
