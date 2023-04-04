package mq

// NewConnTerminatedError returns a new instance of ConnTerminatedError.
func NewConnTerminatedError(message string) *ConnTerminatedError {
	return &ConnTerminatedError{Message: message}
}

// ConnTerminatedError indicates the termination of an mq connection.
type ConnTerminatedError struct {
	Message string
}

// Error is the builtin error interface implementation.
func (e *ConnTerminatedError) Error() string {
	return e.Message
}
