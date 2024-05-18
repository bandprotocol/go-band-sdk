package request

import (
	"github.com/bandprotocol/go-band-sdk/client"
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
}

func NewTask(id, requestID uint64) Task {
	return Task{
		id:        id,
		RequestID: requestID,
	}
}

func (t Task) ID() uint64 {
	return t.id
}

type SuccessResponse struct {
	Task
	client.OracleResult
}

type FailResponse struct {
	Task
	client.OracleResult
	error types.Error
}

func (r FailResponse) Error() string {
	return r.error.Error()
}
