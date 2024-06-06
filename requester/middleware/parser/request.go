package parser

import (
	"fmt"
	"strconv"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
)

func IntoRequestWatcherTaskHandler(ctx sender.SuccessResponse) (request.Task, error) {
	requestID, err := getRequestID(ctx.TxResponse.Logs[0].Events)
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

func getRequestID(events []sdk.StringEvent) (uint64, error) {
	for _, event := range events {
		if event.Type == oracletypes.EventTypeRequest {
			rid, err := strconv.ParseUint(event.Attributes[0].Value, 10, 64)
			if err != nil {
				return 0, err
			}

			return rid, nil
		}
	}
	return 0, fmt.Errorf("cannot find request id")
}
