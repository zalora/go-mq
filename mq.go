package mq

// Message is the primary citizen of mq. It is the m in the mq.
// Type is usually an identifier for the type of message/topic.
// Headers are like the http/tcp headers and can carry information
// about the message itself.
// Ack while self explanatory can also be a destructor(like sqs delete).
// Requeue is a specification/function that lets the caller put
// the message back in the queue.
type Message struct {
	Type   string
	Header Header
	Body   MessageBody
	Ack    func() error

	// mq implementations that support a return to queue can go
	// ahead and use this feature.
	Requeue func() error
	Error   error
}

// Header represents the key-value pairs in an mq header.
type Header map[string]interface{}

// MessageBody denotes the contents carried inside a Message.
type MessageBody map[string]interface{}

// Consumer is a representation of a consumer of queues. Its job
// is to return messages (order depends on the type of mq implemented).
type Consumer interface {
	Consume() (<-chan Message, error)
	Close() error
}

// Publisher is a representation of a pusher of queues. Its job
// is to be able to push messages to the mq itself.
type Publisher interface {
	Publish(messageType string, message Message) error
	Close() error
}

// Logger is a very basic pluggable module some rmq implementations
// optionally accept to log certain important events. Note that this
// should always be an optional feature.
type Logger interface {
	Info(args ...interface{})
	Debug(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
}
