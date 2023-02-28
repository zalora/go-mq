package mq

// Message is the primary citizen of mq. It is the m in the mq.
type Message struct {
	// Type is usually an identifier for the type of message/topic.
	Type string

	// Headers are like the http/tcp headers and can carry information
	// about the message itself.
	Header Header

	// Body is the actual body of the message.
	Body []byte

	// Ack while self explanatory can also be a destructor(like sqs delete).
	Ack func() error

	// Requeue is a specification/function that lets the caller put
	// the message back in the queue.
	// mq implementations that support a return to queue can go
	// ahead and use this feature.
	Requeue func() error

	// Deadletter is a specification/function that lets the caller put
	// the message in a deadletter queue.
	Deadletter func() error

	// Error holds the underlying error in case consume encountered an error.
	Error error
}

// Header represents the key-value pairs in an mq header.
type Header map[string]interface{}

// Subscriber is a representation of a consumer of queues. Its job
// is to return messages (order depends on the type of mq implemented).
type Subscriber interface {
	Subscribe() (<-chan Message, error)
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
