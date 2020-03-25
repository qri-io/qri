package errors

// Error wraps an error and satisfies the error interface
// It couples more developer focused errors with more
// user-friendly errors. If a msg exists, you can send an
// e.Message() to the user, rather than the standard error
type Error struct {
	err error
	msg string
}

// New creates an Error from an error and string
func New(err error, msg string) Error {
	return Error{
		err: err,
		msg: msg,
	}
}

// Error let's the Error struct satisfy the error interface
func (e Error) Error() string {
	return e.err.Error()
}

// Unwrap implements error unwrapping
func (e Error) Unwrap() error {
	return e.err
}

// Message returns the e.msg string
func (e Error) Message() string {
	return e.msg
}
