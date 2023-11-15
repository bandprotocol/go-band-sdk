package middleware

type Middleware[T, U any] struct {
	inCh  <-chan T
	outCh chan<- U
	chain HandlerFunc[T, U]
}

func New[T, U any](
	inCh <-chan T,
	outCh chan<- U,
	parser HandlerFunc[T, U],
	handlers ...Handler[T, U],
) *Middleware[T, U] {
	handlerChain := make([]HandlerFunc[T, U], len(handlers)+1)
	handlerChain[len(handlers)] = parser
	for i := len(handlers) - 1; i >= 0; i-- {
		handlerChain[i] = func(ctx T) (U, error) {
			return handlers[i].Handle(ctx, handlerChain[i+1])
		}
	}

	return &Middleware[T, U]{inCh: inCh, outCh: outCh, chain: handlerChain[0]}
}

func (m *Middleware[T, U]) Run() {
	// Note: Add non-blocking middleware (eg. success middleware
	// that we just want to log/save to DB but don't want to wait to finish)
	for {
		in := <-m.inCh
		out, err := m.chain(in)
		if err != nil {
			continue
		}
		m.outCh <- out
	}
}
