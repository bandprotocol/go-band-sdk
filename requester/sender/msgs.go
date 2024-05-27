package sender

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/go-band-sdk/requester/types"
)

var (
	_ types.Task            = &Task{}
	_ types.SuccessResponse = &SuccessResponse{}
	_ types.FailResponse    = &FailResponse{}
)

type Task struct {
	id  uint64
	Msg sdk.Msg
}

func NewTask(id uint64, msg sdk.Msg) Task {
	return Task{
		id:  id,
		Msg: msg,
	}
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
	Err        types.Error
}

func (r FailResponse) Error() string {
	return r.Err.Error()
}
