package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"time"

	band "github.com/bandprotocol/chain/v2/app"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/viper"

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
)

type ChainConfig struct {
	ChainID string        `yaml:"chain_id" mapstructure:"chain_id"`
	RPC     string        `yaml:"rpc"      mapstructure:"rpc"`
	Fee     string        `yaml:"fee"      mapstructure:"fee"`
	Timeout time.Duration `yaml:"timeout"  mapstructure:"timeout"`
}

type RequestConfig struct {
	OracleScriptID int    `yaml:"oracle_script_id" mapstructure:"oracle_script_id"`
	Calldata       string `yaml:"calldata"         mapstructure:"calldata"`
	Mnemonic       string `yaml:"mnemonic"         mapstructure:"mnemonic"`
}

type Config struct {
	Chain    ChainConfig   `yaml:"chain"     mapstructure:"chain"`
	Request  RequestConfig `yaml:"request"   mapstructure:"request"`
	LogLevel string        `yaml:"log_level" mapstructure:"log_level"`
	SDK      *sdk.Config
}

func GetConfig(name string) (Config, error) {
	viper.SetConfigType("yaml")
	viper.SetConfigName(name)
	viper.AddConfigPath("./requester/configs")

	if err := viper.ReadInConfig(); err != nil {
		return Config{}, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return Config{}, err
	}

	config.SDK = sdk.GetConfig()

	return config, nil
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

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

	senderCh <- sender.NewTask(1, &msg)

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
) (client.SigningResult, error) {
	// Setup watcher.
	watcherCh := make(chan signing.Task, 5)
	w := signing.NewWatcher(cli, log, 60*time.Second, 3*time.Second, watcherCh)

	// start
	go w.Start()

	// new task to query tss signing result.
	watcherCh <- signing.NewTask(2, signingID)
	select {
	case resp := <-w.SuccessfulRequestsCh():
		return resp.SigningResult, nil
	case failResp := <-w.FailedRequestsCh():
		return client.SigningResult{}, failResp
	}
}

func main() {
	// Setup
	config_file := GetEnv("CONFIG_FILE", "example_local.yaml")
	config, err := GetConfig(config_file)
	if err != nil {
		panic(err)
	}

	band.SetBech32AddressPrefixesAndBip44CoinTypeAndSeal(config.SDK)

	// Setup codec
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	// Setup common
	l := logging.NewLogrus(config.LogLevel)
	kr := keyring.NewInMemory(cdc)
	hdPath := hd.CreateHDPath(band.Bip44CoinType, 0, 0)
	info, _ := kr.NewAccount("sender1", config.Request.Mnemonic, "", hdPath.String(), hd.Secp256k1)

	cl, err := client.NewRPC(
		l,
		[]string{config.Chain.RPC},
		config.Chain.ChainID,
		config.Chain.Timeout,
		config.Chain.Fee,
		kr,
	)
	if err != nil {
		panic(err)
	}

	// construct message and send request
	addr, err := info.GetAddress()
	if err != nil {
		panic(err)
	}
	requestMsg, err := getOracleMsgRequestData(config.Request, addr.String())
	if err != nil {
		panic(err)
	}
	oracleResult, err := requestOracleData(cl, l, kr, requestMsg)
	if err != nil {
		panic(err)
	}

	signingID := oracleResult.SigningID
	if signingID == 0 {
		l.Info("example", "No tss signing is requested")
		return
	}

	signingResult, err := getSigningResult(cl, l, uint64(signingID))
	if err != nil {
		panic(err)
	}

	l.Info("example", "Signing result: %v", signingResult)
}
