package gas

import (
	"errors"

	"github.com/bandprotocol/go-band-sdk/requester/middleware"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

var (
	_ middleware.Handler[sender.FailResponse, sender.Task]  = &InsufficientPrepareGasHandler{}
	_ middleware.Handler[request.FailResponse, sender.Task] = &InsufficientExecuteGasHandler{}
)

type InsufficientPrepareGasHandler struct {
	gasMultiplier float64
	logger        logging.Logger
}

func NewInsufficientPrepareGasHandler(gasMultiplier float64, logger logging.Logger) *InsufficientPrepareGasHandler {
	return &InsufficientPrepareGasHandler{gasMultiplier: gasMultiplier, logger: logger}
}

func (m *InsufficientPrepareGasHandler) Handle(
	ctx sender.FailResponse,
	next middleware.HandlerFunc[sender.FailResponse, sender.Task],
) (sender.Task, error) {
	req, err := next(ctx)

	if errors.Is(&ctx.Err, &types.ErrOutOfPrepareGas) {
		req.Msg.PrepareGas = uint64(float64(req.Msg.PrepareGas) * m.gasMultiplier)
		m.logger.Info("bump prepare gas", "bumping request %d prepare gas to %d", req.ID(), req.Msg.PrepareGas)
	}
	return req, err
}

type InsufficientExecuteGasHandler struct {
	gasMultiplier float64
	logger        logging.Logger
}

func NewInsufficientExecuteGasHandler(gasMultiplier float64, logger logging.Logger) *InsufficientExecuteGasHandler {
	return &InsufficientExecuteGasHandler{gasMultiplier: gasMultiplier, logger: logger}
}

func (m *InsufficientExecuteGasHandler) Handle(
	ctx request.FailResponse,
	next middleware.HandlerFunc[request.FailResponse, sender.Task],
) (sender.Task, error) {
	task, err := next(ctx)

	if errors.Is(&ctx.Err, &types.ErrOutOfExecuteGas) {
		task.Msg.ExecuteGas = uint64(float64(task.Msg.ExecuteGas) * m.gasMultiplier)
		m.logger.Info("bump execute gas", "bumping request %d execute gas to %d", task.ID(), task.Msg.ExecuteGas)
	}

	return task, err
}
