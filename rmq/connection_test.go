package rmq

import (
	"errors"
	"testing"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

func Test_NewChannel(t *testing.T) {
	var testCases = []struct {
		desc                     string
		reconnectInterval        time.Duration
		channelError             error
		expectedError            error
		numberOfTimesToReturnErr int
		retryForever             bool
		channel                  *amqp.Channel
		notifyErrors             []*amqp.Error
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
		{
			desc: "if the notify function has recoverable errors when the connection is active, they are handled without a crash",
			notifyErrors: []*amqp.Error{
				&amqp.Error{Code: 1, Reason: "no reason", Recover: true},
				&amqp.Error{Code: 2, Reason: "no reason", Recover: true},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			assert := assert.New(t)

			fakeAmqp := &fakeAmqpConn{
				ch:                   testCase.channel,
				err:                  testCase.channelError,
				noOfCallsToReturnErr: testCase.numberOfTimesToReturnErr,
				channelErrors:        testCase.notifyErrors,
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

			// ugly yes. But in a real production scenario, this call will
			// be part of a daemon so it makes sense to add this sleep.
			if len(testCase.notifyErrors) > 0 {
				time.Sleep(1 * time.Second)
			}

			conn.Close()

		})
	}
}
