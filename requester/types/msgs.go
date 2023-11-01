package types

import (
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
)

// type Task struct {
// 	Id     int64 // Auto increment
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
	Id uint64 // ID to identify request with response later

	Msg *oracletypes.MsgRequestData
}

type Response struct {
	Id uint64 // ID to identify response with original request

	TxHash    string
	RequestId uint64

	ResolveStatus uint8
	Result        []byte
	SignatureId   uint64

	// TODO
	Signature []byte
}

type FailedRequest struct {
	Request

	Error Error
}

func NewFailedRequest(req Request, err Error) FailedRequest {
	return FailedRequest{Request: req, Error: err}
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
