package parser

import (
	"fmt"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
)

func IntoRequestWatcherTaskHandler(ctx sender.SuccessResponse) (request.Task, error) {
	requestID, err := client.GetRequestID(ctx.TxResponse.Logs[0].Events)
	if err != nil {
		return request.Task{}, err
	}

	msg, ok := ctx.Msg.(*oracletypes.MsgRequestData)
	if !ok {
		return request.Task{}, fmt.Errorf("message type is not MsgRequestData")
	}

	return request.NewTask(ctx.ID(), requestID, *msg), nil
}

func IntoSenderTaskHandler(ctx sender.FailResponse) (sender.Task, error) {
	return ctx.Task, nil
}

func IntoSenderTaskHandlerFromRequest(ctx request.FailResponse) (sender.Task, error) {
	return sender.NewTask(ctx.ID(), &ctx.Msg), nil
}
