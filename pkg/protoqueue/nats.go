package protoqueue

import (
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// JetStreamConnection represents a connection to NATS JetStream.
type JetStreamConnection struct {
	NATSConn  *nats.Conn            // The NATS connection.
	JSContext nats.JetStreamContext // The JetStream context.
}

// JetStream represents a message queue using NATS JetStream.
type JetStream struct {
	URL          string              // The NATS server URL.
	Options      []nats.Option       // NATS options.
	StreamName   string              // The name of the JetStream stream.
	Conn         JetStreamConnection // The JetStream connection.
	Subscription *nats.Subscription  // NATS subscription for consuming messages.
	channel      chan any            // A channel for enqueuing messages.
	logger       *logrus.Logger      // A logger for logging messages and errors.
}

// NewJetstream creates and returns a new JetStream instance.
//
// Parameters:
//   - logger: A logger for logging messages and errors.
//   - url: The NATS server URL.
//   - options: NATS options for connecting to the server.
//   - streamName: The name of the JetStream stream.
//
// Returns:
//   - *JetStream: The new JetStream instance.
//   - error: An error if there is any issue creating the JetStream instance.
func NewJetstream(logger *logrus.Logger, url string, options []nats.Option, streamName string) (*JetStream, error) {
	channel := make(chan any, 1000)
	return &JetStream{
		URL:        url,
		Options:    options,
		StreamName: streamName,
		channel:    channel,
		logger:     logger,
	}, nil
}

// Connect establishes a connection to NATS JetStream.
//
// Returns:
//   - error: An error if there is any issue connecting to NATS JetStream.
func (j *JetStream) Connect() error {
	nc, err := nats.Connect(j.URL, j.Options...)
	if err != nil {
		j.logger.WithError(err).Error("failed to connect to NATS server")
		return fmt.Errorf("nats: jetstream connect: failed to connect to NATS server: %w", err)
	}

	j.logger.Info("successfully connected to NATS server")

	jetStreamContext, err := nc.JetStream()
	if err != nil {
		j.logger.WithError(err).Error("failed to get JetStream context")
		return fmt.Errorf("nats: jetstream connect: failed to get JetStream context: %w", err)
	}

	j.logger.Info("successfully got JetStream context")
	j.Conn = JetStreamConnection{NATSConn: nc, JSContext: jetStreamContext}
	return nil
}

// Init initializes the JetStream queue and consumer.
//
// Parameters:
//   - streamConfig: Configuration for creating the JetStream stream.
//
// Returns:
//   - error: An error if there is any issue initializing the JetStream queue and consumer.
func (j *JetStream) Init(streamConfig nats.StreamConfig) error {
	if err := j.CreateStreamIfNotExist(streamConfig); err != nil {
		return fmt.Errorf("nats: jetstream init: failed to create stream: %w", err)
	}

	if _, err := j.CreateConsumer(); err != nil {
		return fmt.Errorf("nats: jetstream init: failed to create consumer: %w", err)
	}

	return j.CreateSubscription()
}

// CreateStreamIfNotExist creates a JetStream stream if it doesn't already exist.
//
// Parameters:
//   - streamConfig: Configuration for creating the JetStream stream.
//
// Returns:
//   - error: An error if there is any issue creating the stream or if the stream already exists.
func (j *JetStream) CreateStreamIfNotExist(streamConfig nats.StreamConfig) error {
	if j.Conn.JSContext == nil {
		err := errors.New("cannot create stream due to nil connection")
		return fmt.Errorf("nats: jetstream CreateStreamIfNotExist: %w", err)
	}

	var err error

	streamInfo, err := j.Conn.JSContext.StreamInfo(j.StreamName)
	if streamInfo != nil && err == nil {
		j.logger.WithField("stream", j.StreamName).Info("stream already exists, skipping creation")
		return nil
	}

	if err != nil && err != nats.ErrStreamNotFound {
		j.logger.WithFields(logrus.Fields{
			"stream": j.StreamName,
			"error":  err,
		}).Warn("error calling JetStream StreamInfo")
	}

	j.logger.WithFields(logrus.Fields{
		"stream": j.StreamName,
		"config": streamConfig,
	}).Info("creating stream")

	_, err = j.Conn.JSContext.AddStream(&streamConfig)
	if err != nil {
		errStr := fmt.Errorf("error while creating stream %s. %s", j.StreamName, err)
		return fmt.Errorf("nats: jetstream CreateStreamIfNotExist: %w", errStr)
	}

	j.logger.WithField("stream", j.StreamName).Info("stream created")
	return nil
}

// CreateConsumer creates a JetStream consumer for the stream.
//
// Returns:
//   - *nats.ConsumerInfo: Information about the created JetStream consumer.
//   - error: An error if there is any issue creating the consumer.
func (j *JetStream) CreateConsumer() (*nats.ConsumerInfo, error) {
	return j.Conn.JSContext.AddConsumer(j.StreamName, &nats.ConsumerConfig{
		Durable:        j.StreamName + "-TODO",
		DeliverSubject: j.StreamName + "-DeliverSubject",
		DeliverGroup:   j.StreamName + "-TODO",
		AckPolicy:      nats.AckExplicitPolicy,
	})
}

// CreateSubscription creates a NATS subscription for consuming messages from JetStream.
//
// Returns:
//   - error: An error if there is any issue creating the subscription.
func (j *JetStream) CreateSubscription() error {
	subscription, err := j.Conn.NATSConn.QueueSubscribeSync(j.StreamName+"-DeliverSubject", j.StreamName+"-TODO")
	if err != nil {
		return fmt.Errorf("nats: jetstream CreateSubscription: failed to create subscription: %w", err)
	}
	j.Subscription = subscription
	return nil
}

// Publish publishes a protobuf message to the JetStream stream.
//
// Parameters:
//   - queuedMessage: The protobuf message to be published.
//
// Returns:
//   - error: An error if there is any issue publishing the message.
func (j *JetStream) Publish(queuedMessage proto.Message) error {
	data, err := proto.Marshal(queuedMessage)
	if err != nil {
		return fmt.Errorf("nats: jetstream publish: failed to marshal queued message: %w", err)
	}

	err = j.publishWithRetry(j.StreamName, data)
	if err != nil {
		return fmt.Errorf("nats: jetstream publish: failed to publish message: %w", err)
	}

	return nil
}

func (j *JetStream) publishWithRetry(subject string, data []byte) error {
	maxRetries := 5
	RetryInterval := 5 * time.Second
	var err error
	for i := 0; i < maxRetries; i++ {
		_, err = j.Conn.JSContext.Publish(subject, data)
		if err == nil {
			return nil
		}

		j.logger.Warn("Publish attempt failed")
		j.logger.WithError(err).Warnf("Publish attempt %d failed", i+1)

		time.Sleep(RetryInterval)
	}
	return err
}

// NextMessage retrieves the next message from the JetStream queue and unmarshals it into the provided protobuf message.
//
// Parameters:
//   - message: A protobuf message where the retrieved message will be unmarshaled.
//
// Returns:
//   - proto.Message: The unmarshaled protobuf message.
//   - error: An error if there is any issue retrieving or unmarshaling the message.
func (j *JetStream) NextMessage(message proto.Message) (proto.Message, error) {
	msg, err := j.Subscription.NextMsg(1 * time.Hour)
	if errors.Is(err, nats.ErrTimeout) {
		return nil, fmt.Errorf("nats: jetstream NextMessage: no message in the queue until the timeout is reached: %w", err)
	}
	if err != nil {
		return nil, err
	}

	err = msg.Ack()
	if err != nil {
		j.logger.WithError(err).Error("failed to ack message")
	}

	err = proto.Unmarshal(msg.Data, message)
	if err != nil {
		return nil, fmt.Errorf("nats: jetstream NextMessage: failed to unmarshal message: %w", err)
	}
	return message, nil
}
