package rmq

import "github.com/streadway/amqp"

type fakeAmqpConn struct {
	ch  *amqp.Channel
	err error
}

func (f *fakeAmqpConn) Channel() (*amqp.Channel, error) {
	return f.ch, f.err
}

func (f *fakeAmqpConn) Close() error {
	// dont really care about this.
	return nil
}

func (f *fakeAmqpConn) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	// similar we dont use this return value for now.
	return nil
}
