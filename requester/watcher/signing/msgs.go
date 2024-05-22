package signing

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
	id        uint64
	SigningID uint64
}

func NewTask(id, signingID uint64) Task {
	return Task{
		id:        id,
		SigningID: signingID,
	}
}

func (t Task) ID() uint64 {
	return t.id
}

type SuccessResponse struct {
	Task
	client.SigningResult
}

type FailResponse struct {
	Task
	client.SigningResult
	Err types.Error
}

func (r FailResponse) Error() string {
	return r.Err.Error()
}
