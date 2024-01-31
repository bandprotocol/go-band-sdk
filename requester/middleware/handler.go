package middleware

type HandlerFunc[T, U any] func(ctx T) (U, error)

type Handler[T, U any] interface {
	Handle(ctx T, next HandlerFunc[T, U]) (U, error)
}
