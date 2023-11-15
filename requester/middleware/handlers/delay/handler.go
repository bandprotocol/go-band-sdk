package delay

import (
	"time"

	"github.com/bandprotocol/go-band-sdk/requester/middleware"
)

type Handler[T, U any] struct {
	delay time.Duration
}

func NewHandler[T, U any](delay time.Duration) *Handler[T, U] {
	return &Handler[T, U]{
		delay: delay,
	}
}

func (h *Handler[T, U]) Handle(ctx T, next middleware.HandlerFunc[T, U]) (U, error) {
	resp, err := next(ctx)
	// Only delay if there is no error
	if err == nil {
		time.Sleep(h.delay)
	}
	return resp, err
}
