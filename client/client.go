package client

import (
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Client interface {
	// Query functions
	GetAccount(account sdk.AccAddress) (client.Account, error)
	GetTx(txHash string) (*sdk.TxResponse, error)
	GetResult(id uint64) (*oracletypes.Result, error)
	GetSignature(id uint64) ([]byte, error)
	BlockSearch(query string, page *int, perPage *int, orderBy string) (*ctypes.ResultBlockSearch, error)
	GetBlockResult(height int64) (*ctypes.ResultBlockResults, error)
	QueryRequestFailureReason(id uint64) (string, error)
	// TODO: Implement get balance for monitoring
	// GetBalance(account sdk.AccAddress) (uint64, error)

	// Build, sign, and broadcast transaction
	SendRequest(msg *oracletypes.MsgRequestData, gasPrice float64, key keyring.Info) (*sdk.TxResponse, error)
}
