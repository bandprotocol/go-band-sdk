package types

import (
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// type Task struct {
// 	ID     int64 // Auto increment
// 	UserId int64 // ID for track request from user

// 	// BandChain spec
// 	OracleScriptId int64
// 	Symbols        string // Comma separated eg. "BTC,ETH,BAND"

// 	// Target spec
// 	Network         string
// 	ContractAddress string

// 	// Misc
// 	CreatedAt     time.Time // Default Now
// 	LastUpdatedAt time.Time // Please set to zero when add (will relay update immediately)
// 	Interval      int       // Interval in seconds
// }

type Request struct {
	ID uint64 // ID to identify request with response later

	Msg *oracletypes.MsgRequestData
}

type RequestResult struct {
	Request

	TxResponse   sdk.TxResponse
	OracleResult oracletypes.Result
	//// TODO
	Signature interface{}
}

type FailedRequest struct {
	RequestResult
	Error error
}

// type EventHandler interface {
// 	AfterTransactionSuccess(hash string, requestId uint64)
// 	AfterTransactionFailed(hash string, code uint64, reason string) (retry bool)
// 	AfterRequestResolve(requestId uint64, status uint8, result []byte) (retry bool)
// }

// Channel receive request
// Pick a sender + create message/transaction -> broadcast
// Wait for block confirm -> If fail what we will do?
// Got request id wait resolve -> If fail what we will do?
// Got result then wait on signature -> If fail what we will do?
