package sender

import (
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/go-band-sdk/requester/types"
)

var _ types.Task = &Task{}
var _ types.SuccessResponse = &SuccessResponse{}
var _ types.FailResponse = &FailResponse{}

type Task struct {
	id  uint64
	Msg oracletypes.MsgRequestData
}

func (t Task) ID() uint64 {
	return t.id
}

type SuccessResponse struct {
	Task
	TxResponse sdk.TxResponse
}

type FailResponse struct {
	Task
	TxResponse sdk.TxResponse
	error      types.Error
}

func (r FailResponse) Error() string {
	return r.error.Error()
}
