package types

import "fmt"

type Error struct {
	Code   int
	Reason string
}

func New(code int, reason string) Error {
	return Error{Code: code, Reason: reason}
}

// Error returns the error message.
func (e Error) Error() string {
	return e.Reason
}

// Wrapf extends the error with additional information.
func (e Error) Wrapf(desc string, args ...interface{}) Error {
	return New(e.Code, fmt.Sprintf(e.Reason+": "+desc, args...))
}

func (e Error) Is(err error) bool {
	if err == nil {
		return false
	}
	if e2, ok := err.(*Error); ok {
		return e2.Code == e.Code
	}
	return false
}

// Define errors from all requester services
var (
	ErrBroadcastFailed   = New(1, "failed to broadcast")
	ErrOutOfPrepareGas   = New(2, "out of prepare gas")
	ErrInsufficientFunds = New(3, "insufficient funds")
	ErrUnconfirmedTx     = New(4, "tx wasn't confirmed within timeout")
	ErrTimedOut          = New(5, "timed out")
	ErrRequestExpired    = New(6, "request expired")
	ErrOutOfExecuteGas   = New(7, "out of execute gas")
	ErrMaxRetryExceeded  = New(8, "max retry exceeded")
	ErrUnknown           = New(999, "unexpected error")
)
