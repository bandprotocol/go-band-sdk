package parser

import (
	"fmt"
	"strconv"

	bandtsstypes "github.com/bandprotocol/chain/v2/x/bandtss/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/signing"
)

func IntoSigningWatcherTaskHandler(ctx sender.SuccessResponse) (signing.Task, error) {
	signingID, err := getSigningID(ctx.TxResponse.Logs[0].Events)
	if err != nil {
		return signing.Task{}, err
	}

	msg, ok := ctx.Msg.(*bandtsstypes.MsgRequestSignature)
	if !ok {
		return signing.Task{}, fmt.Errorf("message type is not MsgRequestData")
	}

	return signing.NewTask(ctx.ID(), signingID, msg), nil
}

func IntoSenderTaskHandlerFromSigning(ctx signing.FailResponse) (sender.Task, error) {
	return sender.NewTask(ctx.ID(), ctx.Msg), nil
}

func getSigningID(events []sdk.StringEvent) (uint64, error) {
	for _, event := range events {
		if event.Type == bandtsstypes.EventTypeSigningRequestCreated {
			sid, err := strconv.ParseUint(event.Attributes[0].Value, 10, 64)
			if err != nil {
				return 0, err
			}

			return sid, nil
		}
	}
	return 0, fmt.Errorf("cannot find signing id")
}
