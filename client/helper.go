package client

import (
	"context"
	"fmt"
	"strconv"
	"time"

	band "github.com/bandprotocol/chain/v2/app"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	libclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func NewClientCtx(chainID string) client.Context {
	cfg := band.MakeEncodingConfig()

	return client.Context{}.
		WithChainID(chainID).
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

func createTxFactory(chainID, gasPrice string, keyring keyring.Keyring) tx.Factory {
	return tx.Factory{}.
		WithChainID(chainID).
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

func GetRequestID(events []sdk.StringEvent) (uint64, error) {
	for _, event := range events {
		if event.Type == oracletypes.EventTypeRequest {
			rid, err := strconv.ParseUint(event.Attributes[0].Value, 10, 64)
			if err != nil {
				return 0, err
			}

			return rid, nil
		}
	}
	return 0, fmt.Errorf("cannot find request id")
}
