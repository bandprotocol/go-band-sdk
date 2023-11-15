package request

import (
	"errors"
	"time"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/utils/logger"
)

type Watcher struct {
	client client.Client
	logger logger.Logger

	pollingDelay time.Duration
	timeout      time.Duration

	// Channel
	watchQueueCh        chan Task
	successfulRequestCh chan SuccessResponse
	failedRequestCh     chan FailResponse
}

func NewWatcher(
	client client.Client,
	logger logger.Logger,
	pollingDelay time.Duration,
	timeout time.Duration,
	watchQueueCh chan Task,
) *Watcher {
	return &Watcher{
		client:              client,
		logger:              logger,
		pollingDelay:        pollingDelay,
		timeout:             timeout,
		watchQueueCh:        watchQueueCh,
		successfulRequestCh: make(chan SuccessResponse),
		failedRequestCh:     make(chan FailResponse),
	}
}

func (w *Watcher) SuccessfulRequestCh() <-chan SuccessResponse {
	return w.successfulRequestCh
}

func (w *Watcher) FailedRequestCh() <-chan FailResponse {
	return w.failedRequestCh
}

func (w *Watcher) Start() {
	for {
		request := <-w.watchQueueCh
		go w.watch(request)
	}
}

func (w *Watcher) watch(task Task) {
	var r SuccessResponse

	et := time.Now().Add(w.timeout)
	for time.Now().Before(et) {
		res, err := w.client.GetResult(task.RequestID)
		r.Result = *res
		if err != nil {
			w.failedRequestCh <- FailResponse{r, err}
			return
		}

		switch res.GetResolveStatus() {
		case oracletypes.RESOLVE_STATUS_OPEN:
			// if request ID found, poll till results gotten or timeout
			break
		case oracletypes.RESOLVE_STATUS_SUCCESS:
			w.successfulRequestCh <- r
		default:
			w.failedRequestCh <- FailResponse{r, errors.New("unknown")}
			return
		}
		time.Sleep(w.pollingDelay)
	}
	w.failedRequestCh <- FailResponse{r, errors.New("timed out")}
}
