package rmq

import (
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/sudarshan-reddy/impetus"
	"github.com/zalora/go-mq"
)

// Subscriber is a rmq specific type that implements mq.Subscriber.
type Subscriber struct {
	channel       AmqpChannel
	queueName     string
	exclusive     bool
	conn          Connection
	retryInterval time.Duration

	msgCh   chan mq.Message
	closeCh chan struct{}
	logger  mq.Logger
}

// NewSubscriber returns a new instance of Subscriber that can be used as an
// mq.Subscriber.
//
// parameters:
// queueName is the queue the Subscriber is listening to/consuming from.
//
// exclusive would make the Subscriber lock the channel its listening to
// and not let any other Subscriber connect.
//
// conn is an rmq.Connection that the client has to initiate and supply.
//
// Sending a retryInterval of 0 disables retrying
func NewSubscriber(queueName string,
	exclusive bool,
	conn Connection,
	retryInterval time.Duration) (*Subscriber, error) {

	ch, err := conn.NewChannel()
	if err != nil {
		return nil, errors.Wrap(err, "error creating/acquiring new rmq channel")
	}

	return &Subscriber{
		channel:       ch,
		queueName:     queueName,
		exclusive:     exclusive,
		conn:          conn,
		retryInterval: retryInterval,
	}, nil
}

//QoS is a thin wrapper on amqp's Qos
func (s *Subscriber) QoS(prefetchCount, prefetchSize int, global bool) error {
	return s.channel.Qos(prefetchCount, prefetchSize, global)
}

// Subscribe implements mq.Subscriber's Subscribe() method.
func (s *Subscriber) Subscribe() (<-chan mq.Message, error) {
	if s.msgCh == nil {
		s.msgCh = make(chan mq.Message)
		s.closeCh = make(chan struct{}, 1)
		go func() {
			defer close(s.msgCh)
			s.consume()
		}()

		return s.msgCh, nil
	}
	return nil, errors.New("queue already being consumed")
}

// consume is an amqp term for subscribe. Well... sort of.
func (s *Subscriber) consume() {
	ticker := impetus.NewImmediateTicker(s.retryInterval)
	for {
		select {
		case <-s.closeCh:
			return
		case <-ticker.C:
			// Channels in rabbitmq/amqp are irrecoverable on most failures.
			// So we create new channels for retries after failure.
			amqpCh, err := s.conn.NewChannel()
			if err != nil {
				s.dispatchError(errors.Wrap(err, "error creating new channel at consume()"))
				return
			}
			consumeCh, err := amqpCh.Consume(
				s.queueName,
				"",
				false,
				s.exclusive,
				false,
				false,
				nil,
			)
			if err != nil {
				if isAmqpAccessRefusedError(err) {
					continue
				}
				s.dispatchError(errors.Wrap(err, "error during consume"))
				return
			}
			s.channel = amqpCh
			s.dispatchMessages(consumeCh)
			return
		}
	}
}

func (s *Subscriber) dispatchError(err error) {
	s.msgCh <- mq.Message{Type: "error-rmq", Error: err, Ack: func() error { return nil }}
	close(s.closeCh)
	s.closeCh = nil
}

func (s *Subscriber) dispatchMessages(mqDeliveryCh <-chan amqp.Delivery) {
	errCh := make(chan *amqp.Error, 1)
	s.channel.NotifyClose(errCh)
	for {
		select {
		case <-s.closeCh:
			return
		case <-errCh:
			amqpCh, err := s.conn.NewChannel()
			if err != nil {
				s.dispatchError(errors.Wrap(err, "error creating new channel at dispatchMessages()"))
				return
			}
			s.channel = amqpCh
			go s.consume()
			return
		case msg, ok := <-mqDeliveryCh:
			if !ok {
				return
			}
			ack := func() error { return msg.Ack(false) }
			requeue := func() error { return msg.Nack(false, true) }
			headers := map[string]interface{}(msg.Headers)
			finalMsg := mq.Message{
				Body:    msg.Body,
				Ack:     ack,
				Requeue: requeue,
				Header:  headers,
				Type:    msg.RoutingKey}
			s.msgCh <- finalMsg
		}
	}
}

// Close closes closeCh which makes the select statement in the goroutines stop.
func (s *Subscriber) Close() error {
	if s.closeCh != nil {
		close(s.closeCh)
		return s.channel.Close()
	}
	return nil
}
