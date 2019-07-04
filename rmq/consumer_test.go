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

func Test_Consume(t *testing.T) {
	var testCases = []struct {
		desc             string
		retryInterval    time.Duration
		amqpDeliveryList []amqp.Delivery
		expectedMessages []mq.Message
		channelErr       error
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
					Body:   map[string]interface{}{"body": "lots of content"},
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
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			assert := assert.New(t)
			fakeCh := &fakeAmqpChannel{
				deliveryList: testCase.amqpDeliveryList,
				err:          testCase.channelErr,
			}
			fakeConn := &fakeConnection{
				fakeAmqpChannel: fakeCh,
			}
			consumer, err := NewConsumer("", true, fakeConn, testCase.retryInterval)
			assert.Nil(err)

			messageCh, err := consumer.Consume()
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
		})
	}
}
