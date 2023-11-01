package types

type RetryHandler struct {
	FailedRequestc <-chan FailedRequest
	Requestc       chan<- Request
	Abortc         chan<- uint64

	SeenRequest map[uint64]uint
	MaxTry      uint
}

func NewRetryHandler(
	in <-chan FailedRequest,
	out chan<- Request,
	abort chan<- uint64,
	maxTry uint,
) *RetryHandler {
	seens := make(map[uint64]uint)

	return &RetryHandler{
		FailedRequestc: in,
		Requestc:       out,
		Abortc:         abort,
		SeenRequest:    seens,
		MaxTry:         maxTry,
	}
}
