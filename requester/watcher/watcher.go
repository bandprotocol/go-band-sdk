package watcher

import (
	"time"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"

	"github.com/bandprotocol/go-band-sdk/requester/middleware"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logger"
	"github.com/bandprotocol/go-band-sdk/utils/txutil"
)

// Get request id from channel poll until resolved
// Send success request to result channel
// Send fail/timeout request to retry handler (via channel)

type Watcher struct {
	client client.Clienter
	logger logger.Logger

	pollingDelay time.Duration
	timeout      time.Duration

	retryMiddlewares []middleware.Retry

	// Channel
	watchQueueCh        chan types.RequestResult
	successfulRequestCh chan types.RequestResult
	failedRequestCh     chan types.FailedRequest
}

func New() *Watcher {
	return &Watcher{
		retryMiddlewares: make([]middleware.Retry, 0),
	}
}

func (w *Watcher) WithRetryMiddleware(middlewares []middleware.Retry) {
	w.retryMiddlewares = middlewares
}

func (w *Watcher) SuccessfulRequestCh() <-chan types.RequestResult {
	return w.successfulRequestCh
}

func (w *Watcher) FailedRequestCh() <-chan types.FailedRequest {
	return w.failedRequestCh
}

func (w *Watcher) Start() {
	for {
		request := <-w.watchQueueCh
		go w.watch(request)
	}
}

func (w *Watcher) watch(reqResult types.RequestResult) {
	var fr types.FailedRequest
	var r types.RequestResult
	var retry = true

	defer func() {
		fr.RequestResult = r
		if retry {
			w.executeRetryMiddleware(fr)
		}
	}()

	et := time.Now().Add(w.timeout)
	for time.Now().Before(et) {
		resp, err := w.client.GetTx(reqResult.TxResponse.TxHash)
		if resp.Code != 0 || err != nil {
			fr.Error = err
			return
		}
		r.TxResponse = *resp

		rid, err := txutil.GetRequestID(resp.Logs[0].Events)
		if err != nil {
			fr.Error = err
			return
		}

		res, err := w.client.GetResult(rid)
		r.OracleResult = *res
		if err != nil {
			fr.Error = err
			return
		}

		switch res.GetResolveStatus() {
		case oracletypes.RESOLVE_STATUS_OPEN:
			// if request ID found, poll till results gotten or timeout
			break
		case oracletypes.RESOLVE_STATUS_SUCCESS:
			retry = false
			w.successfulRequestCh <- r
		default:
			fr.Error = err
			return
		}
		time.Sleep(w.pollingDelay)
	}

	fr.Error = types.ErrUnconfirmedTx
}

func (w *Watcher) executeRetryMiddleware(fr types.FailedRequest) {
	for _, mw := range w.retryMiddlewares {
		fr, next := mw.Call(fr)
		if !next {
			w.failedRequestCh <- fr
			return
		}
	}
	w.watchQueueCh <- fr.RequestResult
}
