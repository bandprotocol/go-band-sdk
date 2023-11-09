package main

import (
	"encoding/hex"
	"fmt"

	band "github.com/bandprotocol/chain/v2/app"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logger"
)

func main() {
	appConfig := sdk.GetConfig()
	band.SetBech32AddressPrefixesAndBip44CoinTypeAndSeal(appConfig)

	requestsCh := make(chan types.Request, 5)

	l := logger.NewLogrus("debug")
	kb := keyring.NewInMemory()
	mnemonic := "child across insect stone enter jacket bitter citizen inch wear breeze adapt come attend vehicle caught wealth junk cloth velvet wheat curious prize panther"
	hdPath := hd.CreateHDPath(band.Bip44CoinType, 0, 0)
	info, _ := kb.NewAccount("sender1", mnemonic, "", hdPath.String(), hd.Secp256k1)
	c, err := client.NewRPC(
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

	s, err := sender.NewSender(c, l, requestsCh, kb)
	go s.Start()

	cd, _ := hex.DecodeString(
		"0000000e00000004414c435800000005435245414d0000000343524f00000004435553440000000446524158000000054845474943000000034a4f45000000034d494d000000045045525000000003534649000000045354524b00000004535553440000000454555344000000045742544301",
	)
	requestsCh <- types.Request{
		ID: 1,
		Msg: oracletypes.NewMsgRequestData(
			401,
			cd,
			16,
			10,
			"test",
			sdk.NewCoins(sdk.NewInt64Coin("uband", 2000)),
			10000,
			42000,
			info.GetAddress(),
		),
	}

	results := s.SuccessRequestsCh()
	failedRequests := s.FailedRequestsCh()

	select {
	case res := <-results:
		fmt.Println("R", res)
	case fail := <-failedRequests:
		fmt.Println("F", fail)
	}
}
