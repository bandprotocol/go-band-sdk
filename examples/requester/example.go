package main

import (
	"encoding/hex"
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
	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/retry"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/sender/handlers/parser"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

type ChainConfig struct {
	ChainID string `yaml:"chain_id" mapstructure:"chain_id"`
	RPC     string `yaml:"rpc" mapstructure:"rpc"`
	Fee     string `yaml:"fee" mapstructure:"fee"`
	Timeout string `yaml:"timeout" mapstructure:"timeout"`
}

type RequestConfig struct {
	OracleScriptID int    `yaml:"oracle_script_id" mapstructure:"oracle_script_id"`
	Calldata       string `yaml:"calldata" mapstructure:"calldata"`
	Mnemonic       string `yaml:"mnemonic" mapstructure:"mnemonic"`
}

type Config struct {
	Chain    ChainConfig   `yaml:"chain" mapstructure:"chain"`
	Request  RequestConfig `yaml:"request" mapstructure:"request"`
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

func GetENV(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetOracleRequestData(reqConf RequestConfig, sender string) (oracletypes.MsgRequestData, error) {
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
	}, nil
}

func main() {
	// Setup
	config_file := GetENV("CONFIG_FILE", "example_band_laozi.yaml")
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
	kb := keyring.NewInMemory(cdc)
	hdPath := hd.CreateHDPath(band.Bip44CoinType, 0, 0)
	info, _ := kb.NewAccount("sender1", config.Request.Mnemonic, "", hdPath.String(), hd.Secp256k1)

	cl, err := client.NewRPC(
		l,
		[]string{config.Chain.RPC},
		config.Chain.ChainID,
		config.Chain.Timeout,
		config.Chain.Fee,
		kb,
	)
	if err != nil {
		panic(err)
	}

	// Setup Sender
	senderCh := make(chan sender.Task, 100)
	s, err := sender.NewSender(cl, l, kb, 0.0025, 60*time.Second, 3*time.Second, senderCh, 100, 100)
	if err != nil {
		panic(err)
	}

	// Setup Watcher
	watcherCh := make(chan request.Task, 100)
	rw := request.NewWatcher(cl, l, 60*time.Second, 3*time.Second, watcherCh, 100, 100)

	// Setup retry and delay handlers
	factory := retry.NewHandlerFactory(3, l)
	retryCounter := retry.NewCounterHandler[sender.FailResponse, sender.Task](factory)
	retryResolver := retry.NewResolverHandler[sender.SuccessResponse, request.Task](factory)

	delayHandler := delay.NewHandler[sender.FailResponse, sender.Task](3 * time.Second)

	// Setup Sender Middleware
	failureMw := middleware.New(
		s.FailedRequestsCh(), senderCh, parser.IntoSenderTaskHandler, retryCounter, delayHandler,
	)
	successMw := middleware.New(
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
	addr, err := info.GetAddress()
	if err != nil {
		panic(err)
	}
	requestMsg, err := GetOracleRequestData(config.Request, addr.String())
	if err != nil {
		panic(err)
	}

	senderCh <- sender.NewTask(1, requestMsg)
	for {
		time.Sleep(10 * time.Second)
		if len(rw.FailedRequestCh()) != 0 || len(rw.SuccessfulRequestCh()) != 0 {
			break
		}
	}
}
