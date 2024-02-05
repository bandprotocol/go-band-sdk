package request

import (
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"

	"github.com/bandprotocol/go-band-sdk/requester/types"
)

var (
	_ types.Task            = &Task{}
	_ types.SuccessResponse = &SuccessResponse{}
	_ types.FailResponse    = &FailResponse{}
)

type Task struct {
	id        uint64 // ID to identify request with response later
	RequestID uint64
	Msg       oracletypes.MsgRequestData
}

func NewTask(id, requestID uint64, msg oracletypes.MsgRequestData) Task {
	return Task{
		id:        id,
		RequestID: requestID,
		Msg:       msg,
	}
}

func (t Task) ID() uint64 {
	return t.id
}

type SuccessResponse struct {
	Task
	oracletypes.Result
}

type FailResponse struct {
	Task
	oracletypes.Result
	error types.Error
}

func (r FailResponse) Error() string {
	return r.error.Error()
}
