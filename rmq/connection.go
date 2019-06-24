package rmq

import (
	"sync/atomic"
	"time"

	"github.com/streadway/amqp"
	"github.com/zalora/go-mq"
)

type Connection struct {
	amqpConn          atomic.Value
	reconnectInterval time.Duration
	closeCh           chan struct{}
	logger            mq.Logger
	retryForever      bool
}

func NewConnection(url string, cfg amqp.Config, reconnectInterval time.Duration) (*Connection, error) {
	amqpConn, err := amqp.DialConfig(url, cfg)
	if err != nil {
		return nil, err
	}
	conn := &Connection{reconnectInterval: reconnectInterval, closeCh: make(chan struct{}, 1)}
	conn.amqpConn.Store(amqpConn)
	go conn.handleConnectionErr(url, cfg)
	return conn, nil
}

func (c *Connection) Close() {
	if c.closeCh != nil {
		close(c.closeCh)
	}
	return
}

func (c *Connection) RetryForever() *Connection {
	c.retryForever = true
	return c
}

func (c *Connection) amqpConnection() *amqp.Connection {
	return c.amqpConn.Load().(*amqp.Connection)
}

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

func (c *Connection) SetLogger(logger mq.Logger) {
	c.logger = logger
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
