package client

import (
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Client interface {
	GetAccount(account sdk.AccAddress) (client.Account, error)
	GetTx(txHash string) (*sdk.TxResponse, error)
	GetResult(id uint64) (*OracleResult, error)
	GetSignature(id uint64) (*SigningResult, error)
	GetBlockResult(height int64) (*ctypes.ResultBlockResults, error)
	QueryRequestFailureReason(id uint64) (string, error)
	GetBalance(account sdk.AccAddress) (uint64, error)
	SendRequest(msg *oracletypes.MsgRequestData, gasPrice float64, key keyring.Record) (*sdk.TxResponse, error)
}
