package main

import (
	band "github.com/bandprotocol/chain/v2/app"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

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

	// example for requesting feeds signature
	err = requestFeedsSignatureExample(l, cl, kr, addr)
	if err != nil {
		panic(err)
	}

	l.Info("example", "=============================================")

	// example for requesting oracle data
	err = requestOracleExample(l, config.Request, cl, kr, addr)
	if err != nil {
		panic(err)
	}
}
