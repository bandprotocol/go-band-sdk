package retry

import (
	"errors"

	"github.com/bandprotocol/go-band-sdk/requester/middleware"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logger"
)

type CounterHandler[T types.Task, U any] struct {
	logger  logger.Logger
	counter *Counter
	maxTry  uint64
}

func (c *CounterHandler[T, U]) Handle(ctx T, next middleware.HandlerFunc[T, U]) (U, error) {
	resp, _ := next(ctx)
	c.counter.Inc(ctx.ID())
	v, ok := c.counter.Peek(ctx.ID())

	if ok && v > c.maxTry {
		return resp, errors.New("max retry exceeded")
	}

	// Override all other errors if max try is not met
	return resp, nil
}

type ResolverHandler[T types.Task, U any] struct {
	logger  logger.Logger
	counter *Counter
}

func (c *ResolverHandler[T, U]) Handle(ctx T, next middleware.HandlerFunc[T, U]) (U, error) {
	resp, err := next(ctx)
	c.counter.Clear(ctx.ID())

	return resp, err
}
