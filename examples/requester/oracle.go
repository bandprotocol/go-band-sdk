package main

import (
	"encoding/hex"
	"fmt"
	"time"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	tsstypes "github.com/bandprotocol/chain/v2/x/tss/types"
	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/middleware"
	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/delay"
	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/gas"
	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/retry"
	"github.com/bandprotocol/go-band-sdk/requester/middleware/parser"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/signing"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func getOracleMsgRequestData(reqConf RequestConfig, sender string) (oracletypes.MsgRequestData, error) {
	calldataBytes, err := hex.DecodeString(reqConf.Calldata)
	if err != nil {
		return oracletypes.MsgRequestData{}, err
	}

	return oracletypes.MsgRequestData{
		OracleScriptID: oracletypes.OracleScriptID(reqConf.OracleScriptID),
		Calldata:       calldataBytes,
		AskCount:       1,
		MinCount:       1,
		ClientID:       "test",
		FeeLimit:       sdk.NewCoins(sdk.NewInt64Coin("uband", 10000)),
		PrepareGas:     3000,
		ExecuteGas:     10000,
		Sender:         sender,
		TSSEncodeType:  oracletypes.ENCODE_TYPE_FULL_ABI,
	}, nil
}

func requestOracleData(
	cli *client.RPC,
	log *logging.Logrus,
	kr keyring.Keyring,
	msg oracletypes.MsgRequestData,
) (client.OracleResult, error) {
	// Setup Sender and Watcher
	senderCh := make(chan sender.Task, 5)
	watcherCh := make(chan request.Task, 5)
	resCh := make(chan client.OracleResult, 5)

	s, err := sender.NewSender(cli, log, kr, 0.0025, 60*time.Second, 3*time.Second, senderCh)
	if err != nil {
		return client.OracleResult{}, err
	}
	w := request.NewWatcher(cli, log, 60*time.Second, 3*time.Second, watcherCh)

	// Setup handlers and middlewares
	f := retry.NewHandlerFactory(3, log)
	retryHandler := retry.NewCounterHandler[sender.FailResponse, sender.Task](f)
	resolveHandler := retry.NewResolverHandler[request.SuccessResponse, client.OracleResult](f)
	delayHandler := delay.NewHandler[sender.FailResponse, sender.Task](3 * time.Second)
	retryFromRequestHandler := retry.NewCounterHandler[request.FailResponse, sender.Task](f)
	delayFromRequestHandler := delay.NewHandler[request.FailResponse, sender.Task](3 * time.Second)
	adjustPrepareGasHandler := gas.NewInsufficientPrepareGasHandler(1.3, log)
	adjustExecuteGasHandler := gas.NewInsufficientExecuteGasHandler(1.3, log)

	retrySenderMw := middleware.New(
		s.FailedRequestsCh(),
		senderCh,
		parser.IntoSenderTaskHandler,
		retryHandler,
		delayHandler,
		adjustPrepareGasHandler,
	)
	retryRequestMw := middleware.New(
		w.FailedRequestsCh(),
		senderCh,
		parser.IntoSenderTaskHandlerFromRequest,
		retryFromRequestHandler,
		delayFromRequestHandler,
		adjustExecuteGasHandler,
	)
	senderToRequestMw := middleware.New(
		s.SuccessfulRequestsCh(), watcherCh, parser.IntoRequestWatcherTaskHandler,
	)
	resolveMw := middleware.New(
		w.SuccessfulRequestsCh(),
		resCh,
		func(ctx request.SuccessResponse) (client.OracleResult, error) { return ctx.OracleResult, nil },
		resolveHandler,
	)

	// start
	go s.Start()
	go w.Start()
	go retrySenderMw.Start()
	go retryRequestMw.Start()
	go senderToRequestMw.Start()
	go resolveMw.Start()

	senderCh <- sender.NewTask(2, &msg)

	select {
	case <-time.After(100 * time.Second):
		return client.OracleResult{}, fmt.Errorf("timeout")
	case errResp := <-retrySenderMw.ErrOutCh():
		return client.OracleResult{}, errResp
	case errResp := <-retryRequestMw.ErrOutCh():
		return client.OracleResult{}, errResp
	case errResp := <-senderToRequestMw.ErrOutCh():
		return client.OracleResult{}, errResp
	case errResp := <-resolveMw.ErrOutCh():
		return client.OracleResult{}, errResp
	case resp := <-resCh:
		return resp, nil
	}
}

func getSigningResult(
	cli *client.RPC,
	log *logging.Logrus,
	signingID uint64,
	msg sdk.Msg,
) (client.SigningResult, error) {
	// Setup watcher.
	watcherCh := make(chan signing.Task, 5)
	w := signing.NewWatcher(cli, log, 60*time.Second, 3*time.Second, watcherCh)

	// start
	go w.Start()

	// new task to query tss signing result.
	watcherCh <- signing.NewTask(2, signingID, msg)
	select {
	case resp := <-w.SuccessfulRequestsCh():
		return resp.SigningResult, nil
	case failResp := <-w.FailedRequestsCh():
		return client.SigningResult{}, failResp
	}
}

func requestOracleExample(
	l *logging.Logrus,
	rc RequestConfig,
	cl *client.RPC,
	kr keyring.Keyring,
	addr sdk.AccAddress,
) error {
	requestMsg, err := getOracleMsgRequestData(rc, addr.String())
	if err != nil {
		l.Error("example", "Failed to get oracle request message: %v", err)
		return err
	}

	oracleResult, err := requestOracleData(cl, l, kr, requestMsg)
	if err != nil {
		l.Error("example", "Failed to request oracle data: %v", err)
		return err
	}

	signingID := oracleResult.SigningID
	if signingID == 0 {
		l.Info("example", "No tss signing is requested")
		return nil
	}

	signingResult, err := getSigningResult(cl, l, uint64(signingID), &requestMsg)
	if err != nil {
		l.Error("example", "Failed to get signing result: %v", err)
		return err
	}

	l.Info("example", "Get Signing result successfully")
	l.Info("example", "Signing of the current group %+v", signingResult.CurrentGroup.EVMSignature)
	if signingResult.ReplacingGroup.Status == tsstypes.SIGNING_STATUS_SUCCESS {
		l.Info("example", "Signing of the replacing group %+v", signingResult.ReplacingGroup.EVMSignature)
	}

	return nil
}
