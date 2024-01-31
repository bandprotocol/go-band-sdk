package retry

import (
	"sync"

	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

type HandlerFactory struct {
	counter Counter
	maxTry  uint64
	logger  logging.Logger
}

func NewHandlerFactory(maxTry uint64, logger logging.Logger) *HandlerFactory {
	return &HandlerFactory{
		counter: Counter{cache: sync.Map{}},
		maxTry:  maxTry,
		logger:  logger,
	}
}

func NewCounterHandler[T types.Task, U any](factory *HandlerFactory) *CounterHandler[T, U] {
	return &CounterHandler[T, U]{
		logger:  factory.logger,
		counter: &factory.counter,
		maxTry:  factory.maxTry,
	}
}

func NewResolverHandler[T types.Task, U any](factory *HandlerFactory) *ResolverHandler[T, U] {
	return &ResolverHandler[T, U]{
		logger:  factory.logger,
		counter: &factory.counter,
	}
}
