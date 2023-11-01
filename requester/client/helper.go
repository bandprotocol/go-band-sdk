package client

import (
	"context"
	"time"

	band "github.com/bandprotocol/chain/v2/app"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	libclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
)

func NewClientCtx(chainId string) client.Context {
	cfg := band.MakeEncodingConfig()

	return client.Context{}.
		WithChainID(chainId).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastSync).
		WithCodec(cfg.Marshaler).
		WithInterfaceRegistry(cfg.InterfaceRegistry).
		WithTxConfig(cfg.TxConfig).
		WithLegacyAmino(cfg.Amino).
		WithViper("requester")
}

func newRPCClient(addr, timeout string) (*rpchttp.HTTP, error) {
	to, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, err
	}

	httpClient, err := libclient.DefaultHTTPClient(addr)
	if err != nil {
		return nil, err
	}

	httpClient.Timeout = to
	rpcClient, err := rpchttp.NewWithClient(addr, "/websocket", httpClient)
	if err != nil {
		return nil, err
	}

	return rpcClient, nil
}

func createTxFactory(chainId, gasPrice string, keyring keyring.Keyring) tx.Factory {
	return tx.Factory{}.
		WithChainID(chainId).
		WithTxConfig(band.MakeEncodingConfig().TxConfig).
		WithGasAdjustment(1.1).
		WithGasPrices(gasPrice).
		WithKeybase(keyring).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
}

func getAccount(clientCtx client.Context, account sdk.AccAddress) (client.Account, error) {
	return authtypes.AccountRetriever{}.GetAccount(clientCtx, account)
}

func getTx(clientCtx client.Context, txHash string) (*sdk.TxResponse, error) {
	return authtx.QueryTx(clientCtx, txHash)
}

func getRequest(clientCtx client.Context, id uint64) (*oracletypes.QueryRequestResponse, error) {
	queryClient := oracletypes.NewQueryClient(clientCtx)
	return queryClient.Request(context.Background(), &oracletypes.QueryRequestRequest{RequestId: id})
}

func estimateGas(clientCtx client.Context, txf tx.Factory, msgs ...sdk.Msg) (uint64, error) {
	_, gas, err := tx.CalculateGas(clientCtx, txf, msgs...)
	return gas, err
}
