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

// Conn is a representation of an rmq connection. It has internal
// specifications that enable it to be threadsafe and fault tolerant
// of connection drops.
type Conn struct {
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

	// RetryForever makes the connection keep trying infinitely to reconnect.
	// TODO: provide retry schemes like backoff.
	RetryForever bool
}

// Connection specifies the structure for an rmq Connection. It is less than
// ideal to have here as we would like a unified connection interface
// for all mqs in this library. However, amqp connections do not
// implement net.Conn or do something similar. The same seems to be
// the case with kafka et all.
// To this end, Connection is a representation of what an rmq.Connection
// looks like.
// Longterm TODO: vision for this is a higher order mq.Connection one day that would
// be the abstract any new implementation of mq conforms to.
type Connection interface {
	NewChannel() (AmqpChannel, error)
}

// NewConnection returns a new instance of Connection.
func NewConnection(
	url string,
	cfg Config,
) (*Conn, error) {

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

	conn := &Conn{
		reconnectInterval: cfg.ReconnectInterval,
		closeCh:           make(chan struct{}, 1),
		logger:            cfg.Logger,
		amqpConn:          amqpConn,
		amqpDialler:       cfg.AmqpDialler,
		retryForever:      cfg.RetryForever,
	}

	go conn.handleConnectionErr(url, amqpConfig)
	return conn, nil
}

// Close closes the connection and usually is safe to call multiple times
// and out of order.
func (c *Conn) Close() {
	if c.closeCh != nil {
		close(c.closeCh)
	}
	return
}

// AmqpChannel is an implementation of the functions we need from
// *amqp.Channel.Its kept minimal on purpose. Feel free to add more on demand.
type AmqpChannel interface {
	Consume(queue string,
		consumer string,
		autoAck bool,
		exclusive bool,
		noLocal bool,
		noWait bool,
		args amqp.Table) (<-chan amqp.Delivery, error)
	Qos(prefetchCount, prefetchSize int, global bool) error
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	Close() error
}

// NewChannel is an amqp channel wrapped with retrying logic.
// It is the part that makes this implementation fault tolerant.
func (c *Conn) NewChannel() (AmqpChannel, error) {
	c.Lock()
	defer c.Unlock()
	ch, err := c.amqpConn.Channel()
	if err != nil {
		for {
			time.Sleep(c.reconnectInterval)
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

func (c *Conn) handleConnectionErr(url string, cfg amqp.Config) {
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
				var err error
				c.amqpConn, err = c.amqpDialler.DialConfig(url, cfg)
				if err != nil {
					time.Sleep(c.reconnectInterval)
					return
				}
				isConnAlive = true
				connErrCh = make(chan *amqp.Error, 1)
				c.amqpConn.NotifyClose(connErrCh)
				if c.logger != nil {
					c.logger.Info("rabbitmq connection restored")
				}
			}()
		}
	}
}

func isAmqpAccessRefusedError(err error) bool {
	if amqpError, ok := err.(*amqp.Error); ok {
		if amqpError.Code == amqp.AccessRefused {
			return true
		}
	}
	return false
}
