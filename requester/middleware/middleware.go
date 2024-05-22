package middleware

type ErrMiddleware[T any] struct {
	Err   error
	Input T
}

func (e ErrMiddleware[T]) Error() string {
	return e.Err.Error()
}

type Middleware[T, U any] struct {
	inCh     <-chan T
	errOutCh chan ErrMiddleware[T]
	outCh    chan<- U
	chain    HandlerFunc[T, U]
}

func New[T, U any](
	inCh <-chan T,
	outCh chan<- U,
	parser HandlerFunc[T, U],
	handlers ...Handler[T, U],
) *Middleware[T, U] {
	errOutCh := make(chan ErrMiddleware[T], cap(outCh))
	handlerChain := make([]HandlerFunc[T, U], len(handlers)+1)
	if len(handlers) == 0 {
		return &Middleware[T, U]{inCh: inCh, outCh: outCh, chain: parser}
	}

	handlerChain[len(handlers)] = parser
	for i := len(handlers) - 1; i >= 0; i-- {
		i := i
		handlerChain[i] = func(ctx T) (U, error) {
			return handlers[i].Handle(ctx, handlerChain[i+1])
		}
	}

	return &Middleware[T, U]{inCh: inCh, outCh: outCh, errOutCh: errOutCh, chain: handlerChain[0]}
}

func (m *Middleware[T, U]) Start() {
	// Note: Add non-blocking middleware (eg. success middleware
	// that we just want to log/save to DB but don't want to wait to finish)
	for {
		in := <-m.inCh
		go func(in T) {
			out, err := m.chain(in)
			if err != nil {
				m.errOutCh <- ErrMiddleware[T]{Err: err, Input: in}
				return
			}
			m.outCh <- out
		}(in)
	}
}

func (m *Middleware[T, U]) ErrOutCh() <-chan ErrMiddleware[T] {
	return m.errOutCh
}
