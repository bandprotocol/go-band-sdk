package sender

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

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
	cache  *lru.TwoQueueCache[uint64, uint64]
}

func NewRetryMiddleware(maxTry uint64, cacheSize uint64) *MaxRetryMiddleware {
	c, _ := lru.New2Q[uint64, uint64](int(cacheSize))
	return &MaxRetryMiddleware{MaxTry: maxTry, cache: c}
}

func (r *MaxRetryMiddleware) Call(request *types.FailedRequest) bool {
	attempt, ok := r.cache.Peek(request.ID)
	if !ok {
		r.cache.Add(request.ID, 1)
		return true
	}

	if attempt < r.MaxTry {
		r.cache.Add(request.ID, attempt+1)
		return true
	} else {
		return false
	}
}

type InsufficientPrepareGasRetryMiddleware struct {
	GasMultiplier float64
}

func (i InsufficientPrepareGasRetryMiddleware) Call(request *types.FailedRequest) bool {
	if request.Error == types.ErrOutOfPrepareGas {
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
