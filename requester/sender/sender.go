package sender

import (
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logger"
)

type Sender struct {
	client client.Client
	logger logger.Logger

	freeKeys chan keyring.Info
	gasPrice float64

	timeout      time.Duration
	pollingDelay time.Duration

	// Channel
	requestQueueCh       chan Task
	successfulRequestsCh chan SuccessResponse
	failedRequestCh      chan FailResponse
}

func NewSender(
	client client.Client,
	logger logger.Logger,
	RequestQueueCh chan Task,
	successChBufferSize int,
	failureChBufferSize int,
	gasPrice float64,
	kr keyring.Keyring,
) (*Sender, error) {
	infos, err := kr.List()
	if err != nil {
		return nil, err
	}

	freeKeys := make(chan keyring.Info, len(infos))
	for _, info := range infos {
		freeKeys <- info
	}

	return &Sender{
		client:               client,
		logger:               logger,
		freeKeys:             freeKeys,
		gasPrice:             gasPrice,
		requestQueueCh:       RequestQueueCh,
		successfulRequestsCh: make(chan SuccessResponse, successChBufferSize),
		failedRequestCh:      make(chan FailResponse, failureChBufferSize),
	}, nil
}

func (s *Sender) SuccessRequestsCh() <-chan SuccessResponse {
	return s.successfulRequestsCh
}

func (s *Sender) FailedRequestsCh() <-chan FailResponse {
	return s.failedRequestCh
}

func (s *Sender) Start() {
	for {
		req := <-s.requestQueueCh
		key := <-s.freeKeys
		go s.request(req, key)
	}
}

func (s *Sender) request(task Task, key keyring.Info) {
	defer func() {
		s.freeKeys <- key
	}()

	// Mutate the msg's sender to the actual sender
	task.Msg.Sender = key.GetAddress().String()

	// Attempt to send the request
	res, err := s.client.SendRequest(&task.Msg, s.gasPrice, key)
	if res.Code != 0 || err != nil {
		s.failedRequestCh <- FailResponse{task, *res, types.ErrBroadcastFailed.Wrapf(err.Error())}
		return
	}
	txHash := res.TxHash

	// Poll for tx confirmation
	et := time.Now().Add(s.timeout)
	for time.Now().Before(et) {

		resp, err := s.client.GetTx(txHash)
		if err != nil {
			time.Sleep(s.pollingDelay)
			continue
		}

		if resp.Code != 0 {
			s.failedRequestCh <- FailResponse{task, *res, types.ErrBroadcastFailed.Wrapf(resp.RawLog)}
			return
		} else {
			s.successfulRequestsCh <- SuccessResponse{task, *res}
			return
		}
	}
}
