package client

import (
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
	SendRequest(msg sdk.Msg, gasPrice float64, key keyring.Record) (*sdk.TxResponse, error)
	GetRequestProofByID(reqID uint64) ([]byte, error)
	Subscribe(name, query string) (*SubscriptionInfo, error)
	Unsubscribe(name string) error
}
