package rmq

import (
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/zalora/go-mq"
)

// Connection is a representation of an rmq connection. It has internal
// specifications that enable it to be threadsafe and fault tolerant
// of connection drops.
type Connection struct {
	amqpConn          atomic.Value
	reconnectInterval time.Duration
	closeCh           chan struct{}
	logger            mq.Logger
	retryForever      bool
}

// NewConnection returns a new instance of Connection.
// It takes a reconnectInterval to attempt reconnections in case of a
// connection failure. It accepts a logger that conforms to mq.Logger.
// logger is optional and if nil, would result in the rmq connection
// not logging anything.
// It also accepts a serviceName and commitID
// that it logs to the exchange while making the connection.
// These can be passed as blanks.
func NewConnection(
	url string,
	reconnectInterval time.Duration,
	logger mq.Logger,
	serviceName string,
	commitID string,
) (*Connection, error) {

	cfg := amqp.Config{
		Properties: amqp.Table{
			"service": serviceName,
			"version": commitID,
		}}

	amqpConn, err := amqp.DialConfig(url, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error dialing to rmq url")
	}

	conn := &Connection{
		reconnectInterval: reconnectInterval,
		closeCh:           make(chan struct{}, 1),
		logger:            logger,
	}

	conn.amqpConn.Store(amqpConn)
	go conn.handleConnectionErr(url, cfg)
	return conn, nil
}

// Close closes the connection and usually is safe to call multiple times
// and out of order.
func (c *Connection) Close() {
	if c.closeCh != nil {
		close(c.closeCh)
	}
	return
}

// RetryForever keeps indefinitely trying to reconnect in case of failures.
func (c *Connection) RetryForever() *Connection {
	c.retryForever = true
	return c
}

func (c *Connection) amqpConnection() *amqp.Connection {
	return c.amqpConn.Load().(*amqp.Connection)
}

// NewChannel is an amqp channel wrapped with retrying logic.
// It is the part that makes this implementation fault tolerant.
func (c *Connection) NewChannel() (*amqp.Channel, error) {
	ch, err := c.amqpConnection().Channel()
	if err != nil {
		for {
			time.Sleep(2 * c.reconnectInterval)
			ch, err := c.amqpConnection().Channel()
			if err == nil {
				return ch, nil
			}
			if !c.retryForever {
				return nil, err
			}
		}
	}
	return ch, nil
}

func (c *Connection) handleConnectionErr(url string, cfg amqp.Config) {
	connErrCh := make(chan *amqp.Error, 1)
	isConnAlive := true
	c.amqpConnection().NotifyClose(connErrCh)
	for {
		select {
		case <-c.closeCh:
			if isConnAlive {
				c.amqpConnection().Close()
			}
			c.closeCh = nil
			return
		case <-connErrCh:
			isConnAlive = false
			if c.logger != nil {
				c.logger.Info("rabbitmq connection lost")
			}
			amqpConn, err := amqp.DialConfig(url, cfg)
			if err != nil {
				time.Sleep(c.reconnectInterval)
				continue
			}
			isConnAlive = true
			connErrCh = make(chan *amqp.Error, 1)
			amqpConn.NotifyClose(connErrCh)
			if c.logger != nil {
				c.logger.Info("rabbitmq connection restored")
			}
			c.amqpConn.Store(amqpConn)
		}
	}
}
