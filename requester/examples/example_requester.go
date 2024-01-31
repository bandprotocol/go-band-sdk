package main

import (
	"time"

	band "github.com/bandprotocol/chain/v2/app"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/middleware"
	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/delay"
	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/retry"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/sender/handlers/parser"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

func main() {
	// Setup
	appConfig := sdk.GetConfig()
	band.SetBech32AddressPrefixesAndBip44CoinTypeAndSeal(appConfig)

	// Setup common
	l := logging.NewLogrus("info")
	kb := keyring.NewInMemory()
	mnemonic := "child across insect stone enter jacket bitter citizen inch wear breeze adapt come attend vehicle caught wealth junk cloth velvet wheat curious prize panther"
	hdPath := hd.CreateHDPath(band.Bip44CoinType, 0, 0)
	info, _ := kb.NewAccount("sender1", mnemonic, "", hdPath.String(), hd.Secp256k1)

	cl, err := client.NewRPC(
		l,
		[]string{"https://rpc.laozi-testnet6.bandchain.org:443"},
		"band-laozi-testnet6",
		"10s",
		"0.0025uband",
		kb,
	)
	if err != nil {
		panic(err)
	}

	// Setup Sender
	senderCh := make(chan sender.Task, 100)
	s, err := sender.NewSender(cl, l, senderCh, 100, 100, 0.0025, kb)

	// Setup Watcher
	watcherCh := make(chan request.Task, 100)
	rw := request.NewWatcher(cl, l, 3*time.Second, 60*time.Second, watcherCh, 100, 100)

	// Setup retry and delay handlers
	factory := retry.NewHandlerFactory(3, l)
	retryCounter := retry.NewCounterHandler[sender.FailResponse, sender.Task](factory)
	retryResolver := retry.NewResolverHandler[sender.SuccessResponse, request.Task](factory)

	delayHandler := delay.NewHandler[sender.FailResponse, sender.Task](3 * time.Second)

	// Setup Sender Middleware
	failureMw := middleware.New[sender.FailResponse, sender.Task](
		s.FailedRequestsCh(), senderCh, parser.IntoSenderTaskHandler, retryCounter, delayHandler,
	)
	successMw := middleware.New[sender.SuccessResponse, request.Task](
		s.SuccessRequestsCh(),
		watcherCh,
		parser.IntoRequestWatcherTaskHandler,
		retryResolver,
	)

	// start
	go s.Start()
	go rw.Start()
	go failureMw.Run()
	go successMw.Run()

	// Send request
	requestMsg := oracletypes.MsgRequestData{
		OracleScriptID: 401,
		Calldata:       []byte{0, 0, 0, 1, 0, 0, 0, 3, 66, 84, 67, 3},
		AskCount:       1,
		MinCount:       1,
		ClientID:       "test",
		FeeLimit:       sdk.NewCoins(sdk.NewInt64Coin("uband", 10000)),
		PrepareGas:     3000,
		ExecuteGas:     10000,
		Sender:         info.GetAddress().String(),
	}

	senderCh <- sender.NewTask(1, requestMsg)
	for {
		time.Sleep(10 * time.Second)
		if len(rw.FailedRequestCh()) != 0 || len(rw.SuccessfulRequestCh()) != 0 {
			break
		}
	}
}
