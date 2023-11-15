package parser

import (
	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
)

func IntoRequestWatcherTaskHandler(ctx sender.SuccessResponse) (request.Task, error) {
	requestID, err := client.GetRequestID(ctx.TxResponse.Logs[0].Events)
	if err != nil {
		return request.Task{}, err
	}

	return request.NewTask(ctx.ID(), requestID, ctx.Msg), nil
}

func IntoSenderTaskHandler(ctx sender.FailResponse) (sender.Task, error) {
	return ctx.Task, nil
}
