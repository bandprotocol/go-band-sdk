package main

import (
	"fmt"
	"time"

	bandtsstypes "github.com/bandprotocol/chain/v2/x/bandtss/types"
	"github.com/bandprotocol/chain/v2/x/feeds/types"
	tsstypes "github.com/bandprotocol/chain/v2/x/tss/types"
	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/middleware"
	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/delay"
	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/retry"
	"github.com/bandprotocol/go-band-sdk/requester/middleware/parser"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/signing"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func getFeedsMsgRequestSignature(
	from sdk.AccAddress,
	signalIDs []string,
	feedType types.FeedType,
) (*bandtsstypes.MsgRequestSignature, error) {
	content := types.NewFeedSignatureOrder(signalIDs, feedType)
	return bandtsstypes.NewMsgRequestSignature(content, sdk.NewCoins(sdk.NewInt64Coin("uband", 100)), from)
}

func requestBandtssSignature(
	cli *client.RPC,
	log *logging.Logrus,
	kr keyring.Keyring,
	msg *bandtsstypes.MsgRequestSignature,
) (client.SigningResult, error) {
	// Setup Sender and Watcher
	senderCh := make(chan sender.Task, 5)
	watcherCh := make(chan signing.Task, 5)
	resCh := make(chan client.SigningResult, 5)

	s, err := sender.NewSender(cli, log, kr, 0.0025, 60*time.Second, 3*time.Second, senderCh)
	if err != nil {
		return client.SigningResult{}, err
	}
	w := signing.NewWatcher(cli, log, 60*time.Second, 3*time.Second, watcherCh)

	// Setup handlers and middlewares
	f := retry.NewHandlerFactory(3, log)
	retryHandler := retry.NewCounterHandler[sender.FailResponse, sender.Task](f)
	resolveHandler := retry.NewResolverHandler[signing.SuccessResponse, client.SigningResult](f)
	delayHandler := delay.NewHandler[sender.FailResponse, sender.Task](3 * time.Second)
	retryFromSigningHandler := retry.NewCounterHandler[signing.FailResponse, sender.Task](f)
	delayFromSigningHandler := delay.NewHandler[signing.FailResponse, sender.Task](3 * time.Second)

	retrySenderMw := middleware.New(
		s.FailedRequestsCh(),
		senderCh,
		parser.IntoSenderTaskHandler,
		retryHandler,
		delayHandler,
	)
	retrySigningMw := middleware.New(
		w.FailedRequestsCh(),
		senderCh,
		parser.IntoSenderTaskHandlerFromSigning,
		retryFromSigningHandler,
		delayFromSigningHandler,
	)
	senderToSigningMw := middleware.New(
		s.SuccessfulRequestsCh(), watcherCh, parser.IntoSigningWatcherTaskHandler,
	)
	resolveMw := middleware.New(
		w.SuccessfulRequestsCh(),
		resCh,
		func(ctx signing.SuccessResponse) (client.SigningResult, error) { return ctx.SigningResult, nil },
		resolveHandler,
	)

	// start
	go s.Start()
	go w.Start()
	go retrySenderMw.Start()
	go retrySigningMw.Start()
	go senderToSigningMw.Start()
	go resolveMw.Start()

	senderCh <- sender.NewTask(1, msg)

	select {
	case <-time.After(100 * time.Second):
		return client.SigningResult{}, fmt.Errorf("timeout")
	case errResp := <-retrySenderMw.ErrOutCh():
		return client.SigningResult{}, errResp
	case errResp := <-senderToSigningMw.ErrOutCh():
		return client.SigningResult{}, errResp
	case errResp := <-resolveMw.ErrOutCh():
		return client.SigningResult{}, errResp
	case resp := <-resCh:
		return resp, nil
	}
}

func requestFeedsSignatureExample(
	l *logging.Logrus,
	cl *client.RPC,
	kr keyring.Keyring,
	addr sdk.AccAddress,
) error {
	feedsSignatureMsg, err := getFeedsMsgRequestSignature(
		addr,
		[]string{"crypto_price.ethusd", "crypto_price.usdtusd"},
		types.FEED_TYPE_DEFAULT,
	)
	if err != nil {
		l.Error("example", "Failed to get feeds signature message: %v", err)
		return err
	}

	signatureResult, err := requestBandtssSignature(cl, l, kr, feedsSignatureMsg)
	if err != nil {
		return err
	}

	l.Info("example", "Get feeds signature result successfully")
	l.Info("example", "Signing of the current group %+v", signatureResult.CurrentGroup.EVMSignature)
	if signatureResult.ReplacingGroup.Status == tsstypes.SIGNING_STATUS_SUCCESS {
		l.Info("example", "Signing of the replacing group %+v", signatureResult.ReplacingGroup.EVMSignature)
	}

	return nil
}
