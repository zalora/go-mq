package rmq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zalora/go-mq"
)

func Test_Consume(t *testing.T) {
	var testCases = []struct {
		desc            string
		retryInterval   time.Duration
		expectedMessage mq.Message
	}{}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			assert := assert.New(t)
			fakeConn := &fakeConnection{}
			consumer, err := NewConsumer("", true, fakeConn, testCase.retryInterval)
			assert.Nil(err)

			messageCh, err := consumer.Consume()
			assert.Nil(err)

			for message := range messageCh {
				assert.Equal(testCase.expectedMessage, message)
			}
		})
	}
}
