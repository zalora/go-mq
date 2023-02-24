package rmq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type fakeAmqpConn struct {
	ch                   *amqp.Channel
	calls                int
	noOfCallsToReturnErr int
	err                  error
	channelErrors        []*amqp.Error
}

func (f *fakeAmqpConn) Channel() (*amqp.Channel, error) {
	f.calls++
	if f.calls > f.noOfCallsToReturnErr {
		return f.ch, nil
	}

	return f.ch, f.err
}

func (f *fakeAmqpConn) Close() error {
	// dont really care about this.
	return nil
}

func (f *fakeAmqpConn) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	// similar we dont use this return value for now.
	go func() {
		for _, amqpErr := range f.channelErrors {
			receiver <- amqpErr
		}
	}()
	return nil
}

type fakeDialler struct {
	amqpConn AmqpConn
}

func (f *fakeDialler) DialConfig(url string, config amqp.Config) (AmqpConn, error) {
	return f.amqpConn, nil
}

type fakeConnection struct {
	Connection
	fakeAmqpChannel      AmqpChannel
	calls                int
	noOfCallsToReturnErr int
	err                  error
}

func (f *fakeConnection) NewChannel() (AmqpChannel, error) {
	f.calls++
	if f.calls > f.noOfCallsToReturnErr {
		return f.fakeAmqpChannel, nil
	}
	return f.fakeAmqpChannel, f.err
}

type fakeAmqpChannel struct {
	AmqpChannel
	deliveryList         []amqp.Delivery
	calls                int
	noOfCallsToReturnErr int
	err                  error
	notifyErr            *amqp.Error
}

func (f *fakeAmqpChannel) Consume(queue string,
	consumer string,
	autoAck bool,
	exclusive bool,
	noLocal bool,
	noWait bool,
	args amqp.Table) (<-chan amqp.Delivery, error) {
	deliveryCh := make(chan amqp.Delivery)
	f.calls++
	if f.calls > f.noOfCallsToReturnErr && f.calls == 0 && f.noOfCallsToReturnErr == 0 {
		return deliveryCh, nil
	}

	go func() {
		defer close(deliveryCh)
		for _, delivery := range f.deliveryList {
			deliveryCh <- delivery
		}
	}()

	return deliveryCh, f.err
}

func (f *fakeAmqpChannel) Close() error {
	return nil
}

func (f *fakeAmqpChannel) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	if f.notifyErr != nil {
		receiver <- f.notifyErr
	}
	return nil
}
