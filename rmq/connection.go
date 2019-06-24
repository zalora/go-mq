package rmq

import (
	"sync/atomic"
	"time"

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
// It takes a rabbitmq url, an amqp.Config (TODO: make this optional
// and a reconnectInterval to attempt reconnections in case of a
// connection failure.
func NewConnection(url string, cfg amqp.Config,
	reconnectInterval time.Duration,
	logger mq.Logger) (*Connection, error) {
	amqpConn, err := amqp.DialConfig(url, cfg)
	if err != nil {
		return nil, err
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
