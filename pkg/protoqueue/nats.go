package protoqueue

import (
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type JetStreamConnection struct {
	NATSConn  *nats.Conn
	JSContext nats.JetStreamContext
}

type JetStream struct {
	URL          string
	Options      []nats.Option
	StreamName   string
	Conn         JetStreamConnection
	Subscription *nats.Subscription
	channel      chan any
	logger       *logrus.Logger
}

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

func (j *JetStream) Connect() error {
	nc, err := nats.Connect(j.URL, j.Options...)
	if err != nil {
		j.logger.WithError(err).Error("failed to connect to NATS server")
		return fmt.Errorf("nats: jetstream connect: failed to connect to NATS server: %w", err)
	}

	j.logger.Info("successfully connected to NATS server")

	jetStreamContext, err := nc.JetStream()
	if err != nil {
		j.logger.WithError(err).Error("failed to get jetstream context")
		return fmt.Errorf("nats: jetstream connect: failed to get jetstream context: %w", err)
	}

	j.logger.Info("successfully got jetstream context")
	j.Conn = JetStreamConnection{NATSConn: nc, JSContext: jetStreamContext}
	return nil
}

func (j *JetStream) Init(streamConfig nats.StreamConfig) error {
	if err := j.CreateStreamIfNotExist(streamConfig); err != nil {
		return fmt.Errorf("nats: jetstream init: failed to create stream: %w", err)
	}

	if _, err := j.CreateConsumer(); err != nil {
		return fmt.Errorf("nats: jetstream init: failed to create consumer: %w", err)
	}

	return j.CreateSubscription()
}

func (j *JetStream) CreateStreamIfNotExist(streamConfig nats.StreamConfig) error {
	if j.Conn.JSContext == nil {
		err := errors.New("can not create stream due to nil connection")
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
		}).Warn("error calling jetstream StreamInfo")

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

func (j *JetStream) CreateConsumer() (*nats.ConsumerInfo, error) {
	return j.Conn.JSContext.AddConsumer(j.StreamName, &nats.ConsumerConfig{
		Durable:        j.StreamName + "-TODO",
		DeliverSubject: j.StreamName + "-DeliverSubject",
		DeliverGroup:   j.StreamName + "-TODO",
		AckPolicy:      nats.AckExplicitPolicy,
	})
}

func (j *JetStream) CreateSubscription() error {
	subscription, err := j.Conn.NATSConn.QueueSubscribeSync(j.StreamName+"-DeliverSubject", j.StreamName+"-TODO")
	if err != nil {
		return fmt.Errorf("nats: jetstream CreateSubscription: failed to create subscription: %w", err)
	}
	j.Subscription = subscription
	return nil
}

func (j *JetStream) Publish(queuedMessage proto.Message) error {
	data, err := proto.Marshal(queuedMessage)
	if err != nil {
		return fmt.Errorf("nats: jetstream publish: failed to marshal queued message: %w", err)
	}

	_, err = j.Conn.JSContext.Publish(j.StreamName, data)
	if err != nil {
		return fmt.Errorf("nats: jetstream publish: failed to publish message: %w", err)
	}

	return nil
}

func (j *JetStream) NextMessage(message proto.Message) (proto.Message, error) {
	msg, err := j.Subscription.NextMsg(1 * time.Hour)
	if errors.Is(err, nats.ErrTimeout) {
		return nil, fmt.Errorf("nats: jetstream NextMessage: no message in the queue until timeout is reached: %w", err)
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
