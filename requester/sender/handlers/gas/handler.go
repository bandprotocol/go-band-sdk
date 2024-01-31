package gas

import (
	"strings"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/bandprotocol/go-owasm/api"

	"github.com/bandprotocol/go-band-sdk/requester/middleware"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

var _ middleware.Handler[sender.FailResponse, sender.Task] = &InsufficientPrepareGasHandler{}

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

	txResp := ctx.TxResponse
	if txResp.Codespace == oracletypes.ModuleName && txResp.Code == oracletypes.ErrBadWasmExecution.ABCICode() {
		if strings.Contains(txResp.RawLog, api.ErrOutOfGas.Error()) {
			req.Msg.PrepareGas = uint64(float64(req.Msg.PrepareGas) * m.gasMultiplier)
		}
		m.logger.Info("bump prepare gas", "bumping request %s prepare gas to %s", req.ID, req.Msg.PrepareGas)
	}
	return req, err
}
