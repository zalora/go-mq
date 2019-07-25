package rmq

import (
	"testing"
	"time"

	"errors"

	pkgerrors "github.com/pkg/errors"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/zalora/go-mq"
)

func Test_Subscribe(t *testing.T) {
	var testCases = []struct {
		desc             string
		retryInterval    time.Duration
		amqpDeliveryList []amqp.Delivery
		expectedMessages []mq.Message
		channelErr       error
		notifyErr        *amqp.Error
	}{
		{
			desc:          "amqp type messages get converted to appropriate mq.Messages",
			retryInterval: 1 * time.Second,
			amqpDeliveryList: []amqp.Delivery{
				amqp.Delivery{
					RoutingKey: "test.event",
					Headers:    map[string]interface{}{"some_random_header": "random_header_value"},
					Body:       []byte(`{"body": "lots of content"}`),
				},
			},
			expectedMessages: []mq.Message{
				mq.Message{
					Type:   "test.event",
					Header: map[string]interface{}{"some_random_header": "random_header_value"},
					Body:   []byte(`{"body": "lots of content"}`),
				},
			},
		},
		{
			desc:          "in case of an error from the consume, appropriate error messages are dispatched",
			retryInterval: 1 * time.Second,
			channelErr:    errors.New("consume error"),
			expectedMessages: []mq.Message{
				mq.Message{
					Type:  "error-rmq",
					Error: pkgerrors.Wrap(errors.New("consume error"), "error during consume"),
				},
			},
		},
		{
			desc:          "if notifyerror returns an error, its handled and retried",
			retryInterval: 1 * time.Second,
			amqpDeliveryList: []amqp.Delivery{
				amqp.Delivery{
					RoutingKey: "test.event",
					Headers:    map[string]interface{}{"some_random_header": "random_header_value"},
					Body:       []byte(`{"body": "lots of content"}`),
				},
			},
			expectedMessages: []mq.Message{
				mq.Message{
					Type:   "test.event",
					Header: map[string]interface{}{"some_random_header": "random_header_value"},
					Body:   []byte(`{"body": "lots of content"}`),
				},
			},
			notifyErr: &amqp.Error{
				Code:   123,
				Reason: "jk lol",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			assert := assert.New(t)
			fakeCh := &fakeAmqpChannel{
				deliveryList: testCase.amqpDeliveryList,
				err:          testCase.channelErr,
				notifyErr:    testCase.notifyErr,
			}
			fakeConn := &fakeConnection{
				fakeAmqpChannel: fakeCh,
			}
			subscriber, err := NewSubscriber("", true, fakeConn, testCase.retryInterval)
			assert.Nil(err)

			messageCh, err := subscriber.Subscribe()
			assert.Nil(err)

			var i int
			for message := range messageCh {
				message.Ack = nil
				message.Requeue = nil
				if testCase.expectedMessages[i].Error != nil {
					assert.EqualError(message.Error, testCase.expectedMessages[i].Error.Error())
				} else {
					assert.Equal(testCase.expectedMessages[i], message)
					assert.Nil(message.Error)
				}
				i++
			}
			subscriber.Close()
		})
	}
}
