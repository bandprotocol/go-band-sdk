package parser

import (
	"fmt"

	bandtsstypes "github.com/bandprotocol/chain/v2/x/bandtss/types"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/signing"
)

func IntoSigningWatcherTaskHandler(ctx sender.SuccessResponse) (signing.Task, error) {
	signingID, err := client.GetSigningID(ctx.TxResponse.Logs[0].Events)
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
