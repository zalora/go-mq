package rmq

import (
	"errors"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

type fakeAmqpConn struct {
	ch                   *amqp.Channel
	calls                int
	noOfCallsToReturnErr int
	err                  error
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
	return nil
}

type fakeDialler struct {
	amqpConn AmqpConn
}

func (f *fakeDialler) DialConfig(url string, config amqp.Config) (AmqpConn, error) {
	return f.amqpConn, nil
}

func Test_NewChannel(t *testing.T) {
	var testCases = []struct {
		desc                     string
		reconnectInterval        time.Duration
		channelError             error
		expectedError            error
		numberOfTimesToReturnErr int
		retryForever             bool
		channel                  *amqp.Channel
	}{
		{
			desc:              "for a general case, when amqp returns a channel, return a new usable channel to the caller",
			reconnectInterval: 1 * time.Second,
		},
		{
			desc:                     "if channel returns an error, more than twice, we don't retry after that",
			channelError:             errors.New("channel error"),
			expectedError:            pkgerrors.Wrap(errors.New("channel error"), "error getting channel from amqp"),
			numberOfTimesToReturnErr: 2,
		},
		{
			desc:                     "if channel returns an error, newchannel tries again to get a success",
			channelError:             errors.New("channel error"),
			numberOfTimesToReturnErr: 1,
		},
		{
			desc:                     "if channel returns an error, more than twice and retry forever is chosen, we retry till theres no error",
			channelError:             errors.New("channel error"),
			numberOfTimesToReturnErr: 2,
			retryForever:             true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			assert := assert.New(t)

			fakeAmqp := &fakeAmqpConn{
				ch:                   testCase.channel,
				err:                  testCase.channelError,
				noOfCallsToReturnErr: testCase.numberOfTimesToReturnErr,
			}
			fd := &fakeDialler{amqpConn: fakeAmqp}
			conn, err := NewConnection("some-url", testCase.reconnectInterval, nil, fd, "", "")
			assert.Nil(err)

			if testCase.retryForever {
				conn = conn.RetryForever()
			}
			ch, err := conn.NewChannel()

			assert.IsType(testCase.expectedError, err)
			if err != nil {
				assert.Equal(testCase.expectedError.Error(), err.Error())
			}
			assert.Equal(testCase.channel, ch)
			conn.Close()

		})
	}
}
