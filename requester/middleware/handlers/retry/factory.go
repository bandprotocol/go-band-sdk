package retry

import (
	"sync"

	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
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

func NewCounterHandlerWithCondition[T types.Task, U any](
	factory *HandlerFactory,
	retryCond func(T) bool,
	errFromCtx func(T) error,
) *CounterHandlerWithCondition[T, U] {
	return &CounterHandlerWithCondition[T, U]{
		logger:     factory.logger,
		counter:    &factory.counter,
		maxTry:     factory.maxTry,
		retryCond:  retryCond,
		errFromCtx: errFromCtx,
	}
}

func NewResolverHandler[T types.Task, U any](factory *HandlerFactory) *ResolverHandler[T, U] {
	return &ResolverHandler[T, U]{
		logger:  factory.logger,
		counter: &factory.counter,
	}
}

func NewRequestRetryHandler(factory *HandlerFactory) *CounterHandlerWithCondition[request.FailResponse, sender.Task] {
	return &CounterHandlerWithCondition[request.FailResponse, sender.Task]{
		logger:  factory.logger,
		counter: &factory.counter,
		maxTry:  factory.maxTry,
		retryCond: func(ctx request.FailResponse) bool {
			return ctx.Err.Is(types.ErrRequestExpired) || ctx.Err.Is(types.ErrOutOfExecuteGas)
		},
		errFromCtx: func(ctx request.FailResponse) error { return ctx.Err },
	}
}

func NewSenderRetryHandler(factory *HandlerFactory) *CounterHandlerWithCondition[sender.FailResponse, sender.Task] {
	return &CounterHandlerWithCondition[sender.FailResponse, sender.Task]{
		logger:  factory.logger,
		counter: &factory.counter,
		maxTry:  factory.maxTry,
		retryCond: func(ctx sender.FailResponse) bool {
			return ctx.Err.Is(types.ErrOutOfPrepareGas)
		},
		errFromCtx: func(ctx sender.FailResponse) error { return ctx.Err },
	}
}
