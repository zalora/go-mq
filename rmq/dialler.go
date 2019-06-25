package rmq

import "github.com/streadway/amqp"

// AmqpDialler is a abstraction of a dial function to obtain
// AmqpConn.
type AmqpDialler interface {
	// The amqp.Config is non negotiable here as this interface
	// has to adhere to streadway/amqp standards and not the
	// other way around if we want to use their dial.
	DialConfig(url string, config amqp.Config) (AmqpConn, error)
}

// DefaultAmqpDialler returns an AmqpConn implemented by streadway/amqp.
type DefaultAmqpDialler struct{}

// DialConfig is just a light wrapper on amqp.DialConfig.
func (d *DefaultAmqpDialler) DialConfig(url string, config amqp.Config) (AmqpConn, error) {
	return amqp.DialConfig(url, config)
}
