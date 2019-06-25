package rmq

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/zalora/go-mq"
)

// AmqpConn is an interface that abstracts out functions from
// amqp.Conn that are needed by this library itself.
type AmqpConn interface {
	Channel() (*amqp.Channel, error)
	Close() error
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
}

// Connection is a representation of an rmq connection. It has internal
// specifications that enable it to be threadsafe and fault tolerant
// of connection drops.
type Connection struct {
	sync.Mutex
	amqpConn          AmqpConn
	amqpDialler       AmqpDialler
	reconnectInterval time.Duration
	closeCh           chan struct{}
	logger            mq.Logger
	retryForever      bool
}

// Config is a set of optional configurations to tune up the
// rmq connection and its retries.
// It also accepts a serviceName and commitID
// that it logs to the exchange while making the connection.
// These can be passed as blanks.
type Config struct {
	// ReconnectInterval is set to attempt reconnections in case of a
	// connection failure.
	ReconnectInterval time.Duration

	// Logger is optional and if nil, would result in the rmq connection
	// not logging anything.
	Logger mq.Logger

	// AmqpDialler if nil would result in the library creating a
	// DefaultAmqpDialler.
	AmqpDialler AmqpDialler

	// ServiceName is listed as service in the connection.
	ServiceName string

	// CommmitID is listed as the version in the connection.
	CommitID string
}

// NewConnection returns a new instance of Connection.
func NewConnection(
	url string,
	cfg Config,
) (*Connection, error) {

	amqpConfig := amqp.Config{
		Properties: amqp.Table{
			"service": cfg.ServiceName,
			"version": cfg.CommitID,
		}}

	if cfg.AmqpDialler == nil {
		cfg.AmqpDialler = &DefaultAmqpDialler{}
	}

	amqpConn, err := cfg.AmqpDialler.DialConfig(url, amqpConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error dialing to rmq url")
	}

	conn := &Connection{
		reconnectInterval: cfg.ReconnectInterval,
		closeCh:           make(chan struct{}, 1),
		logger:            cfg.Logger,
		amqpConn:          amqpConn,
		amqpDialler:       cfg.AmqpDialler,
	}

	go conn.handleConnectionErr(url, amqpConfig)
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

// NewChannel is an amqp channel wrapped with retrying logic.
// It is the part that makes this implementation fault tolerant.
func (c *Connection) NewChannel() (*amqp.Channel, error) {
	c.Lock()
	defer c.Unlock()
	ch, err := c.amqpConn.Channel()
	if err != nil {
		for {
			time.Sleep(2 * c.reconnectInterval)
			ch, err := c.amqpConn.Channel()
			if err == nil {
				return ch, nil
			}
			if !c.retryForever {
				return nil, errors.Wrap(err, "error getting channel from amqp")
			}
		}
	}
	return ch, nil
}

func (c *Connection) handleConnectionErr(url string, cfg amqp.Config) {
	connErrCh := make(chan *amqp.Error, 1)
	isConnAlive := true
	c.amqpConn.NotifyClose(connErrCh)
	for {
		select {
		case <-c.closeCh:
			if isConnAlive {
				c.amqpConn.Close()
			}
			c.closeCh = nil
			return
		case connErr := <-connErrCh:
			// We don't want to try to recover if rmq tells us not to recover.
			if !connErr.Recover {
				return
			}
			isConnAlive = false
			if c.logger != nil {
				c.logger.Info("rabbitmq connection lost")
			}
			func() {
				c.Lock()
				defer c.Unlock()
				amqpConn, err := c.amqpDialler.DialConfig(url, cfg)
				if err != nil {
					time.Sleep(c.reconnectInterval)
					return
				}
				isConnAlive = true
				connErrCh = make(chan *amqp.Error, 1)
				amqpConn.NotifyClose(connErrCh)
				if c.logger != nil {
					c.logger.Info("rabbitmq connection restored")
				}
			}()
		}
	}
}
