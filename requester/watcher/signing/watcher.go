package signing

import (
	"encoding/json"
	"time"

	tsstypes "github.com/bandprotocol/chain/v2/x/tss/types"

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
	if task.SigningID == 0 {
		w.failedRequestCh <- FailResponse{task, client.SigningResult{}, types.ErrUnknown.Wrapf("signing ID %d is invalid", task.SigningID)}
		return
	}

	et := time.Now().Add(w.timeout)

	for time.Now().Before(et) {
		res, err := w.client.GetSignature(task.SigningID)
		if err != nil {
			time.Sleep(w.pollingDelay)
			continue
		}

		isWaiting := res.CurrentGroup.Status == tsstypes.SIGNING_STATUS_WAITING ||
			res.ReplacingGroup.Status == tsstypes.SIGNING_STATUS_WAITING

		done := res.CurrentGroup.Status == tsstypes.SIGNING_STATUS_SUCCESS &&
			(res.ReplacingGroup.Status == tsstypes.SIGNING_STATUS_SUCCESS ||
				res.ReplacingGroup.Status == tsstypes.SIGNING_STATUS_UNSPECIFIED)

		switch {
		case isWaiting:
			time.Sleep(w.pollingDelay)
		case done:
			// Assume all results can be marshalled
			b, _ := json.Marshal(res)
			w.logger.Info("Watcher", "task ID(%d) has been resolved with result: %s", task.ID(), string(b))
			w.successfulRequestCh <- SuccessResponse{task, *res}
			return
		default:
			// Assume all results can be marshalled
			b, _ := json.Marshal(res)
			w.logger.Info("Watcher", "task ID(%d) has failed with result: %s", task.ID(), string(b))
			wrappedErr := types.ErrUnknown.Wrapf("signign ID %d failed with unknown reason: %s", task.SigningID, err)
			w.failedRequestCh <- FailResponse{task, *res, wrappedErr}
			return
		}
	}

	w.failedRequestCh <- FailResponse{task, client.SigningResult{}, types.ErrTimedOut}
}