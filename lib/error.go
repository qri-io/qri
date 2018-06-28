package lib

// Error wraps an error and satisfies the error interface
// It couples more developer focused errors with more
// user-friendly errors. If a msg exists, you can send an
// e.Message() to the user, rather than the standard error
type Error struct {
	err error
	msg string
}

// Error let's the Error struct satisfy the error interface
func (e Error) Error() string {
	return e.err.Error()
}

// Message returns the e.msg string
func (e Error) Message() string {
	return e.msg
}

// NewError creates an Error from an error and string
func NewError(err error, msg string) Error {
	return Error{
		err: err,
		msg: msg,
	}
}
