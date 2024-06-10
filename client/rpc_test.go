package client

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	band "github.com/bandprotocol/chain/v2/app"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TODO: Rewrite test into real test
func TestEstimateGas(t *testing.T) {
	appConfig := sdk.GetConfig()
	band.SetBech32AddressPrefixesAndBip44CoinTypeAndSeal(appConfig)
	chainID := "band-laozi-testnet6"
	rpc, err := newRPCClient("https://rpc.laozi-testnet6.bandchain.org:443", 10*time.Second)
	require.NoError(t, err)

	// Setup codec
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	// Setup keybase
	kb := keyring.NewInMemory(cdc)
	mnemonic := "child across insect stone enter jacket bitter citizen inch wear breeze " +
		"adapt come attend vehicle caught wealth junk cloth velvet wheat curious prize panther"
	hdPath := hd.CreateHDPath(band.Bip44CoinType, 0, 0)
	info, err := kb.NewAccount("sender1", mnemonic, "", hdPath.String(), hd.Secp256k1)
	require.NoError(t, err)

	ctx := NewClientCtx(chainID).WithClient(rpc)

	addr, err := info.GetAddress()
	require.NoError(t, err)

	acc, err := getAccount(ctx, addr)
	require.NoError(t, err)

	txf := createTxFactory(
		chainID,
		"0.0025uband",
		kb,
	).WithAccountNumber(acc.GetAccountNumber()).
		WithSequence(acc.GetSequence())

	cd, _ := hex.DecodeString(
		"0000000e00000004414c435800000005435245414d0000000343524f000000044355534400000004465241580000000548" +
			"45474943000000034a4f45000000034d494d000000045045525000000003534649000000045354524b000000045355534" +
			"40000000454555344000000045742544301",
	)

	sdkAddr, err := info.GetAddress()
	require.NoError(t, err)

	gas, err := estimateGas(
		ctx, txf, oracletypes.NewMsgRequestData(
			401, cd, 16, 10, "test", sdk.NewCoins(sdk.NewInt64Coin("uband", 2000)), 10000, 42000, sdkAddr, 0,
		),
	)
	fmt.Println(gas)
	require.NoError(t, err)
}
