package request

import (
	"encoding/json"
	"time"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

type Watcher struct {
	client client.Client
	logger logging.Logger

	timeout      time.Duration
	pollingDelay time.Duration

	// Channel
	watchQueueCh         <-chan Task
	successfulRequestsCh chan SuccessResponse
	failedRequestsCh     chan FailResponse
}

func NewWatcher(
	client client.Client,
	logger logging.Logger,
	timeout time.Duration,
	pollingDelay time.Duration,
	watchQueueCh chan Task,
	successChBufferSize int,
	failureChBufferSize int,
) *Watcher {
	return &Watcher{
		client:               client,
		logger:               logger,
		timeout:              timeout,
		pollingDelay:         pollingDelay,
		watchQueueCh:         watchQueueCh,
		successfulRequestsCh: make(chan SuccessResponse, successChBufferSize),
		failedRequestsCh:     make(chan FailResponse, failureChBufferSize),
	}
}

func (w *Watcher) SuccessfulRequestsCh() <-chan SuccessResponse {
	return w.successfulRequestsCh
}

func (w *Watcher) FailedRequestsCh() <-chan FailResponse {
	return w.failedRequestsCh
}

func (w *Watcher) Start() {
	for request := range w.watchQueueCh {
		go w.watch(request)
	}
}

func (w *Watcher) watch(task Task) {
	et := time.Now().Add(w.timeout)

	for time.Now().Before(et) {
		res, err := w.client.GetResult(task.RequestID)
		if err != nil {
			time.Sleep(w.pollingDelay)
			continue
		}

		switch res.Result.GetResolveStatus() {
		case oracletypes.RESOLVE_STATUS_OPEN:
			// if request ID found, poll till results gotten or timeout
			time.Sleep(w.pollingDelay)
		case oracletypes.RESOLVE_STATUS_SUCCESS:
			// Assume all results can be marshalled
			b, _ := json.Marshal(res)
			w.logger.Info("Watcher", "task ID(%d) has been resolved with result: %s", task.ID(), string(b))
			w.successfulRequestsCh <- SuccessResponse{task, *res}
			return
		default:
			// Assume all results can be marshalled
			b, _ := json.Marshal(res)
			w.logger.Info("Watcher", "task ID(%d) has failed with result: %s", task.ID(), string(b))
			wrappedErr := types.ErrUnknown.Wrapf("request ID %d failed with unknown reason: %s", task.RequestID, err)
			w.failedRequestsCh <- FailResponse{task, *res, wrappedErr}
			return
		}
	}

	w.failedRequestsCh <- FailResponse{task, client.OracleResult{}, types.ErrTimedOut}
}
