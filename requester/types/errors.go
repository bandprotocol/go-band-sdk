package types

import "fmt"

// TODO: Move to re-use on other service
type Error struct {
	Code   int
	Reason string
}

func New(code int, reason string) Error {
	return Error{Code: code, Reason: reason}
}

func (e Error) Error() string {
	return e.Reason
}

// Wrapf extends this error with an additional information.
// It's a handy function to call Wrapf with sdk errors.
func (e *Error) Wrapf(desc string, args ...interface{}) Error {
	return New(e.Code, fmt.Sprintf(e.Reason+": "+desc, args...))
}

// Define errors from all requester services
var (
	ErrBroadcast         = New(1, "fail to broadcast")
	ErrInsufficientFunds = New(2, "insufficient funds to create request")
	ErrTxNotConfirm      = New(3, "tx takes too long to confirm")

	ErrUnexpected = New(999, "unexpected error please fix immediately")
)
