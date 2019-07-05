package rmq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/sudarshan-reddy/impetus"
	"github.com/zalora/go-mq"
)

// Consumer is a rmq specific type that implements
// mq.Consumer.
type Consumer struct {
	channel       AmqpChannel
	queueName     string
	exclusive     bool
	conn          Connection
	retryInterval time.Duration

	msgCh       chan mq.Message
	closeCh     chan struct{}
	logger      mq.Logger
	unmarshaler json.Unmarshaler
}

// NewConsumer returns a new instance of Consumer that
// can be used as an mq.Consumer.
// queueName is the queue the consumer is listening to/
// consuming from.
// exclusive would make the consumer lock the  channel
// its listening to and not let any other consumer connect.
// conn is an rmq.Connection that the client has to initiate
// and supply.
// Sending a retryInterval of 0 is akin to disabling retry.
func NewConsumer(queueName string,
	exclusive bool,
	conn Connection,
	retryInterval time.Duration) (*Consumer, error) {

	ch, err := conn.NewChannel()
	if err != nil {
		return nil, errors.Wrap(err, "error creating/acquiring new rmq channel")
	}

	return &Consumer{
		channel:       ch,
		queueName:     queueName,
		exclusive:     exclusive,
		conn:          conn,
		retryInterval: retryInterval,
	}, nil
}

//QoS is a thin wrapper on amqp's Qos
func (c *Consumer) QoS(prefetchCount, prefetchSize int, global bool) error {
	return c.channel.Qos(prefetchCount, prefetchSize, global)
}

// Consume implements mq.Consumer's Consume() method.
func (c *Consumer) Consume() (<-chan mq.Message, error) {
	if c.msgCh == nil {
		c.msgCh = make(chan mq.Message)
		c.closeCh = make(chan struct{}, 1)
		go func() {
			defer close(c.msgCh)
			c.consume()
		}()

		return c.msgCh, nil
	}
	return nil, errors.New("queue already being consumed")
}

func (c *Consumer) consume() {
	ticker := impetus.NewImmediateTicker(c.retryInterval)
	for {
		select {
		case <-c.closeCh:
			return
		case <-ticker.C:
			amqpCh, err := c.conn.NewChannel()
			if err != nil {
				c.dispatchError(errors.Wrap(err, "error creating new channel at consume()"))
				return
			}
			consumeCh, err := amqpCh.Consume(
				c.queueName,
				"",
				false,
				c.exclusive,
				false,
				false,
				nil,
			)
			if err != nil {
				if isAmqpAccessRefusedError(err) {
					continue
				}
				c.dispatchError(errors.Wrap(err, "error during consume"))
				return
			}
			c.channel = amqpCh
			c.dispatchMessages(consumeCh)
			return
		}
	}
}

func (c *Consumer) dispatchError(err error) {
	c.msgCh <- mq.Message{Type: "error-rmq", Error: err, Ack: func() error { return nil }}
	close(c.closeCh)
	c.closeCh = nil
}

func (c *Consumer) dispatchMessages(mqDeliveryCh <-chan amqp.Delivery) {
	errCh := make(chan *amqp.Error, 1)
	c.channel.NotifyClose(errCh)
	for {
		select {
		case <-c.closeCh:
			return
		case <-errCh:
			amqpCh, err := c.conn.NewChannel()
			if err != nil {
				c.dispatchError(errors.Wrap(err, "error creating new channel at dispatchMessages()"))
				return
			}
			c.channel = amqpCh
			go c.consume()
			return
		case msg, ok := <-mqDeliveryCh:
			if !ok {
				return
			}
			ack := func() error { return msg.Ack(false) }
			requeue := func() error { return msg.Nack(false, true) }
			msgBody := map[string]interface{}{}
			if len(msg.Body) != 0 {
				if err := json.Unmarshal(msg.Body, &msgBody); err != nil {
					ack()
					errorWithBody := fmt.Errorf("invalid message format: %s: %s", err, string(msg.Body))
					errMsg := mq.Message{
						Error: errorWithBody,
						Ack:   func() error { return nil },
					}
					c.msgCh <- errMsg
					continue
				}
			}
			headers := map[string]interface{}(msg.Headers)
			finalMsg := mq.Message{
				Body:    msgBody,
				Ack:     ack,
				Requeue: requeue,
				Header:  headers,
				Type:    msg.RoutingKey}
			c.msgCh <- finalMsg
		}
	}
}

// Close closes closeCh which makes the select statement in the goroutines
// stop.
func (c *Consumer) Close() error {
	if c.closeCh != nil {
		close(c.closeCh)
		return c.channel.Close()
	}
	return nil
}
