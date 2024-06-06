package main

import (
	"fmt"
	"time"

	band "github.com/bandprotocol/chain/v2/app"
	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"

	subscriber "github.com/bandprotocol/go-band-sdk/subscriber"
)

type ChainConfig struct {
	ChainID string        `yaml:"chain_id" mapstructure:"chain_id"`
	RPC     string        `yaml:"rpc" mapstructure:"rpc"`
	Fee     string        `yaml:"fee" mapstructure:"fee"`
	Timeout time.Duration `yaml:"timeout" mapstructure:"timeout"`
}

func DefaultChainConfig() ChainConfig {
	return ChainConfig{
		RPC:     "tcp://localhost:26657",
		ChainID: "bandchain",
		Fee:     "0.0025uband",
		Timeout: 10 * time.Second,
	}
}

func parseEvent(eventData types.EventDataTx) {
	for _, event := range eventData.Result.Events {
		if event.Type == "transfer" {
			var sender, recipient, amount string
			for _, attribute := range event.Attributes {
				key := string(attribute.Key)
				value := string(attribute.Value)
				switch key {
				case "sender":
					sender = value
				case "recipient":
					recipient = value
				case "amount":
					amount = value
				}
			}
			fmt.Printf("Sender: %s, Recipient: %s, Amount: %s\n", sender, recipient, amount)
		}
	}
}

func main() {
	sdkConfig := sdk.GetConfig()
	conf := DefaultChainConfig()

	band.SetBech32AddressPrefixesAndBip44CoinTypeAndSeal(sdkConfig)

	// Setup codec
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	// Setup common
	l := logging.NewLogrus("debug")
	kr := keyring.NewInMemory(cdc)

	rpcEndponts := []string{conf.RPC}
	cli, err := client.NewRPC(l, rpcEndponts, conf.ChainID, conf.Timeout, conf.Fee, kr)
	if err != nil {
		panic(err)
	}

	sub := subscriber.NewSubscriber(cli, l, subscriber.DefaultConfig())

	query := "message.action='/cosmos.bank.v1beta1.MsgSend'"
	evInfo, err := sub.Subscribe("transfer", query, func(e ctypes.ResultEvent) (string, error) {
		return "test", nil
	})
	if err != nil {
		panic(err)
	}

	l.Info("main", "start listening events with query \"%s\"", query)
	for {
		select {
		case event := <-evInfo.OutCh:
			l.Info("main", "Received transfer event:")
			parseEvent(event.Data.(types.EventDataTx))
		case <-time.Tick(5 * time.Minute):
			if err := sub.Unsubscribe("transfer"); err != nil {
				l.Info("main", "Failed to unsubscribe from events")
				return
			}

			l.Info("main", "Finished listening to events")
			return
		}
	}
}
