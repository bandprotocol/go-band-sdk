package middleware

import (
	"strings"
	"time"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/bandprotocol/go-owasm/api"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logger"
)

var (
	_ RetryMiddleware = &MaxRetryMiddleware{}
	_ RetryMiddleware = &InsufficientPrepareGasRetryMiddleware{}
	_ RetryMiddleware = &RetryWithDelayMiddleware{}
)

type RetryMiddleware interface {
	Call(r *types.FailedRequest, l logger.Logger) bool
}

type MaxRetryMiddleware struct {
	MaxTry uint64
	cache  *lru.TwoQueueCache[uint64, uint64]
}

func NewRetryMiddleware(maxTry uint64, cacheSize uint64) *MaxRetryMiddleware {
	c, _ := lru.New2Q[uint64, uint64](int(cacheSize))
	return &MaxRetryMiddleware{MaxTry: maxTry, cache: c}
}

func (m *MaxRetryMiddleware) Call(r *types.FailedRequest, l logger.Logger) bool {
	attempt, ok := m.cache.Peek(r.ID)
	if !ok {
		m.cache.Add(r.ID, 1)
		return true
	}

	if attempt < m.MaxTry {
		m.cache.Add(r.ID, attempt+1)
		return true
	} else {
		l.Error("max retry exceeded", "request %s failed after %s tries", r.ID, attempt)
		return false
	}
}

type InsufficientPrepareGasRetryMiddleware struct {
	GasMultiplier float64
}

func (i InsufficientPrepareGasRetryMiddleware) Call(r *types.FailedRequest, l logger.Logger) bool {
	resp := r.TxResponse
	if resp.Codespace == oracletypes.ModuleName && resp.Code == oracletypes.ErrBadWasmExecution.ABCICode() {
		if strings.Contains(resp.RawLog, api.ErrOutOfGas.Error()) {
			r.Request.Msg.PrepareGas = uint64(float64(r.Request.Msg.PrepareGas) * i.GasMultiplier)
		}
		l.Info("bump prepare gas", "bumping request %s prepare gas to %s", r.ID, r.Request.Msg.PrepareGas)
	}
	return true
}

type RetryWithDelayMiddleware struct {
	Delay time.Duration
}

func (r RetryWithDelayMiddleware) Call(_ *types.FailedRequest, _ logger.Logger) bool {
	time.Sleep(r.Delay)
	return true
}
