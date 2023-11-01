package main

import (
	"encoding/hex"
	"fmt"

	"github.com/bandprotocol/band-sdk/requester/client"
	"github.com/bandprotocol/band-sdk/requester/sender"
	"github.com/bandprotocol/band-sdk/requester/types"
	"github.com/bandprotocol/band-sdk/utils"
	band "github.com/bandprotocol/chain/v2/app"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func main() {
	appConfig := sdk.GetConfig()
	band.SetBech32AddressPrefixesAndBip44CoinTypeAndSeal(appConfig)

	requests := make(chan types.Request, 5)
	results := make(chan types.Response, 5)
	failedRequests := make(chan types.FailedRequest, 5)

	logger := utils.NewLogrusLogger("debug")
	kb := keyring.NewInMemory()
	mnemonic := "child across insect stone enter jacket bitter citizen inch wear breeze adapt come attend vehicle caught wealth junk cloth velvet wheat curious prize panther"
	hdPath := hd.CreateHDPath(band.Bip44CoinType, 0, 0)
	info, _ := kb.NewAccount("sender1", mnemonic, "", hdPath.String(), hd.Secp256k1)
	client, err := client.NewMultipleClient(
		logger,
		[]string{"https://rpc.laozi-testnet6.bandchain.org:443"},
		"band-laozi-testnet6",
		"10s",
		"0.0025uband",
		kb,
	)
	fmt.Println("DDD")
	if err != nil {
		panic(err)
	}
	fmt.Println(info.GetAddress().String())
	s := sender.NewSender(
		client,
		logger,
		requests,
		results,
		failedRequests,
		[]keyring.Info{info},
		utils.MustParseDuration("1s"),
		utils.MustParseDuration("1m"),
	)
	fmt.Println("Sender done")
	go s.Start()

	cd, _ := hex.DecodeString(
		"0000000e00000004414c435800000005435245414d0000000343524f00000004435553440000000446524158000000054845474943000000034a4f45000000034d494d000000045045525000000003534649000000045354524b00000004535553440000000454555344000000045742544301",
	)
	requests <- types.Request{Id: 1, Msg: oracletypes.NewMsgRequestData(401, cd, 16, 10, "test", sdk.NewCoins(sdk.NewInt64Coin("uband", 2000)), 10000, 42000, info.GetAddress())}

	select {
	case res := <-results:
		fmt.Println("R", res)
	case fail := <-failedRequests:
		fmt.Println("F", fail)
	}

}
