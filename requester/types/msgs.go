package types

type Task interface {
	ID() uint64
}

type SuccessResponse interface {
	Task
}

type FailResponse interface {
	Task
	Error() string
}

//
//// type EventHandler interface {
//// 	AfterTransactionSuccess(hash string, requestId uint64)
//// 	AfterTransactionFailed(hash string, code uint64, reason string) (retry bool)
//// 	AfterRequestResolve(requestId uint64, status uint8, result []byte) (retry bool)
//// }
//
//// Channel receive request
//// Pick a sender + create message/transaction -> broadcast
//// Wait for block confirm -> If fail what we will do?
//// Got request id wait resolve -> If fail what we will do?
//// Got result then wait on signature -> If fail what we will do?
