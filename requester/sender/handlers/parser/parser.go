package parser

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
)

func IntoRequestWatcherTaskHandler(ctx sender.SuccessResponse) (request.Task, error) {
	requestID, err := getRequestID(ctx.TxResponse.Logs[0].Events)
	if err != nil {
		return request.Task{}, err
	}

	return request.NewTask(ctx.ID(), requestID, ctx.Msg), nil
}

func IntoSenderTaskHandler(ctx sender.FailResponse) (sender.Task, error) {
	return ctx.Task, nil
}

func getRequestID(events []sdk.StringEvent) (uint64, error) {
	for _, event := range events {
		if event.Type == "request" {
			rid, err := strconv.ParseUint(event.Attributes[0].Value, 10, 64)
			if err != nil {
				return 0, err
			}

			return rid, nil
		}
	}
	return 0, fmt.Errorf("cannot find request id")
}
