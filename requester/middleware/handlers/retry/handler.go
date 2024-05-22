package retry

import (
	"github.com/bandprotocol/go-band-sdk/requester/middleware"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

type CounterHandler[T types.Task, U any] struct {
	logger  logging.Logger
	counter *Counter
	maxTry  uint64
}

func (c *CounterHandler[T, U]) Handle(ctx T, next middleware.HandlerFunc[T, U]) (U, error) {
	resp, _ := next(ctx)
	c.counter.Inc(ctx.ID())
	v, ok := c.counter.Peek(ctx.ID())

	if ok && v > c.maxTry {
		return resp, types.ErrMaxRetryExceeded
	}

	// Override all other errors if max try is not met
	return resp, nil
}

type CounterHandlerWithCondition[T types.Task, U any] struct {
	logger     logging.Logger
	counter    *Counter
	maxTry     uint64
	retryCond  func(ctx T) bool
	errFromCtx func(ctx T) error
}

func (c *CounterHandlerWithCondition[T, U]) Handle(ctx T, next middleware.HandlerFunc[T, U]) (U, error) {
	resp, _ := next(ctx)
	c.counter.Inc(ctx.ID())

	if c.retryCond != nil && !c.retryCond(ctx) {
		return resp, c.errFromCtx(ctx)
	}

	v, ok := c.counter.Peek(ctx.ID())

	if ok && v > c.maxTry {
		return resp, types.ErrMaxRetryExceeded.Wrapf("with reason: %s", c.errFromCtx(ctx))
	}
	return resp, nil
}

type ResolverHandler[T types.Task, U any] struct {
	logger  logging.Logger
	counter *Counter
}

func (c *ResolverHandler[T, U]) Handle(ctx T, next middleware.HandlerFunc[T, U]) (U, error) {
	resp, err := next(ctx)
	c.counter.Clear(ctx.ID())

	return resp, err
}
