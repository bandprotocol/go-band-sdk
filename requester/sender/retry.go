package sender

import (
	"time"

	"github.com/bandprotocol/go-band-sdk/requester/types"
)

var (
	_ RetryMiddleware = &MaxRetryMiddleware{}
	_ RetryMiddleware = &InsufficientPrepareGasRetryMiddleware{}
	_ RetryMiddleware = &RetryWithDelayMiddleware{}
)

type RetryMiddleware interface {
	Call(request *types.FailedRequest) bool
}

type MaxRetryMiddleware struct {
	MaxTry uint64
	seen   map[uint64]uint64
}

func (r *MaxRetryMiddleware) Call(request *types.FailedRequest) bool {
	attempt := r.seen[request.Id]
	if attempt < r.MaxTry {
		r.seen[request.Id] = attempt + 1
		return true
	} else {
		return false
	}
}

type InsufficientPrepareGasRetryMiddleware struct {
	GasMultiplier float64
}

func (i InsufficientPrepareGasRetryMiddleware) Call(request *types.FailedRequest) bool {
	if request.Error.Code == 2 {
		request.Request.Msg.PrepareGas = uint64(float64(request.Request.Msg.PrepareGas) * i.GasMultiplier)
	}
	return true
}

type RetryWithDelayMiddleware struct {
	Delay time.Duration
}

func (r RetryWithDelayMiddleware) Call(_ *types.FailedRequest) bool {
	time.Sleep(r.Delay)
	return true
}
