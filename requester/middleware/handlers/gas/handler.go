package gas

import (
	"errors"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
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

// NewInsufficientPrepareGasHandler creates a new handler for InsufficientPrepareGas events, specifically for the oracletypes.MsgRequestData message type.
func NewInsufficientPrepareGasHandler(gasMultiplier float64, logger logging.Logger) *InsufficientPrepareGasHandler {
	return &InsufficientPrepareGasHandler{gasMultiplier: gasMultiplier, logger: logger}
}

func (m *InsufficientPrepareGasHandler) Handle(
	ctx sender.FailResponse,
	next middleware.HandlerFunc[sender.FailResponse, sender.Task],
) (sender.Task, error) {
	req, err := next(ctx)

	if errors.Is(&ctx.Err, &types.ErrOutOfPrepareGas) {
		msg, ok := req.Msg.(*oracletypes.MsgRequestData)
		if !ok {
			return req, errors.New("message type is not MsgRequestData")
		}

		msg.PrepareGas = uint64(float64(msg.PrepareGas) * m.gasMultiplier)
		req.Msg = msg
		m.logger.Debug("bump prepare gas", "bumping request %d prepare gas to %d", req.ID(), msg.PrepareGas)
	}
	return req, err
}

type InsufficientExecuteGasHandler struct {
	gasMultiplier float64
	logger        logging.Logger
}

// NewInsufficientExecuteGasHandler creates a new handler for InsufficientExecuteGas events, specifically for the oracletypes.MsgRequestData message type.
func NewInsufficientExecuteGasHandler(gasMultiplier float64, logger logging.Logger) *InsufficientExecuteGasHandler {
	return &InsufficientExecuteGasHandler{gasMultiplier: gasMultiplier, logger: logger}
}

func (m *InsufficientExecuteGasHandler) Handle(
	ctx request.FailResponse,
	next middleware.HandlerFunc[request.FailResponse, sender.Task],
) (sender.Task, error) {
	task, err := next(ctx)

	if errors.Is(&ctx.Err, &types.ErrOutOfExecuteGas) {
		msg, ok := task.Msg.(*oracletypes.MsgRequestData)
		if !ok {
			return task, errors.New("message type is not MsgRequestData")
		}
		msg.ExecuteGas = uint64(float64(msg.ExecuteGas) * m.gasMultiplier)
		task.Msg = msg
		m.logger.Debug("bump execute gas", "bumping request %d execute gas to %d", task.ID(), msg.ExecuteGas)
	}

	return task, err
}
