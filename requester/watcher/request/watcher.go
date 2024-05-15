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
	watchQueueCh        <-chan Task
	successfulRequestCh chan SuccessResponse
	failedRequestCh     chan FailResponse
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
		client:              client,
		logger:              logger,
		timeout:             timeout,
		pollingDelay:        pollingDelay,
		watchQueueCh:        watchQueueCh,
		successfulRequestCh: make(chan SuccessResponse, successChBufferSize),
		failedRequestCh:     make(chan FailResponse, failureChBufferSize),
	}
}

func (w *Watcher) SuccessfulRequestCh() <-chan SuccessResponse {
	return w.successfulRequestCh
}

func (w *Watcher) FailedRequestCh() <-chan FailResponse {
	return w.failedRequestCh
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
			w.logger.Info("Watcher", "request ID(%d) has been resolved with result: %s", task.ID(), string(b))
			w.successfulRequestCh <- SuccessResponse{task, *res}
			return
		default:
			// Assume all results can be marshalled
			b, _ := json.Marshal(res)
			w.logger.Info("Watcher", "request ID(%d) has failed with result: %s", task.ID(), string(b))
			wrappedErr := types.ErrUnknown.Wrapf("request ID %d failed with unknown reason: %s", task.RequestID, err)
			w.failedRequestCh <- FailResponse{task, *res, wrappedErr}
			return
		}
	}

	w.failedRequestCh <- FailResponse{task, client.OracleResult{}, types.ErrTimedOut}
}
