package rmq

import (
	"testing"
	"time"

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
	}{
		{
			desc:          "",
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
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			assert := assert.New(t)
			fakeCh := &fakeAmqpChannel{
				deliveryList: testCase.amqpDeliveryList,
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
				if i >= len(testCase.amqpDeliveryList) {
					break
				}
				message.Ack = nil
				message.Requeue = nil
				assert.Equal(testCase.expectedMessages[i], message)
				i++
			}
		})
	}
}
