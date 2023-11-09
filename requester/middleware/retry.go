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
	_ Retry = &MaxRetry{}
	_ Retry = &RetryWithInsufficientPrepareGas{}
	_ Retry = &RetryWithDelay{}
)

type Retry interface {
	Call(r types.FailedRequest) (types.FailedRequest, bool)
}

type MaxRetry struct {
	MaxTry uint64
	cache  *lru.TwoQueueCache[uint64, uint64]
	logger logger.Logger
}

func NewMaxRetry(maxTry uint64, cacheSize uint64, logger logger.Logger) *MaxRetry {
	c, _ := lru.New2Q[uint64, uint64](int(cacheSize))
	return &MaxRetry{MaxTry: maxTry, cache: c, logger: logger}
}

func (m *MaxRetry) Call(r types.FailedRequest) (types.FailedRequest, bool) {
	attempt, ok := m.cache.Peek(r.ID)
	if !ok {
		m.cache.Add(r.ID, 1)
		return r, true
	}

	if attempt < m.MaxTry {
		m.cache.Add(r.ID, attempt+1)
		return r, true
	} else {
		m.logger.Error("max retry exceeded", "request %s failed after %s tries", r.ID, attempt)
		return r, false
	}
}

type RetryWithInsufficientPrepareGas struct {
	gasMultiplier float64
	logger        logger.Logger
}

func NewRetryWithInsufficientPrepareGas(gasMultiplier float64, logger logger.Logger) *RetryWithInsufficientPrepareGas {
	return &RetryWithInsufficientPrepareGas{gasMultiplier: gasMultiplier, logger: logger}
}

func (m RetryWithInsufficientPrepareGas) Call(r types.FailedRequest) (types.FailedRequest, bool) {
	resp := r.TxResponse
	if resp.Codespace == oracletypes.ModuleName && resp.Code == oracletypes.ErrBadWasmExecution.ABCICode() {
		if strings.Contains(resp.RawLog, api.ErrOutOfGas.Error()) {
			r.Request.Msg.PrepareGas = uint64(float64(r.Request.Msg.PrepareGas) * m.gasMultiplier)
		}
		m.logger.Info("bump prepare gas", "bumping request %s prepare gas to %s", r.ID, r.Request.Msg.PrepareGas)
	}
	return r, true
}

type RetryWithDelay struct {
	delay time.Duration
}

func NewRetryWithDelay(delay time.Duration) *RetryWithDelay {
	return &RetryWithDelay{delay: delay}
}

func (m RetryWithDelay) Call(r types.FailedRequest) (types.FailedRequest, bool) {
	time.Sleep(m.delay)
	return r, true
}
