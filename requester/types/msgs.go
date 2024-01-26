package types

type Task interface {
	ID() uint64
}

type SuccessResponse interface {
	Task
}

type FailResponse interface {
	Task
	Error() string
}
