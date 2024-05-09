package sender

import (
	"encoding/json"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

type Sender struct {
	client client.Client
	logger logging.Logger

	freeKeys chan keyring.Record
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
	logger logging.Logger,
	kr keyring.Keyring,
	gasPrice float64,
	timeout time.Duration,
	pollingDelay time.Duration,
	requestQueueCh chan Task,
	successChBufferSize int,
	failureChBufferSize int,
) (*Sender, error) {
	infos, err := kr.List()
	if err != nil {
		return nil, err
	}

	freeKeys := make(chan keyring.Record, len(infos))
	for _, info := range infos {
		freeKeys <- *info
	}

	return &Sender{
		client:               client,
		logger:               logger,
		freeKeys:             freeKeys,
		gasPrice:             gasPrice,
		timeout:              timeout,
		pollingDelay:         pollingDelay,
		requestQueueCh:       requestQueueCh,
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
		// Assume all tasks can be marshalled
		b, _ := json.Marshal(req.Msg)

		s.logger.Info("Sender", "querying request with ID(%d) with payload: %s", req.ID(), string(b))

		go s.request(req, key)
	}
}

func (s *Sender) request(task Task, key keyring.Record) {
	defer func() {
		s.freeKeys <- key
	}()

	// Mutate the msg's sender to the actual sender
	addr, err := key.GetAddress()
	if err != nil {
		s.logger.Error("Sender", "failed to get address from key: %s", err.Error())
		return
	}
	task.Msg.Sender = addr.String()

	// Attempt to send the request
	resp, err := s.client.SendRequest(&task.Msg, s.gasPrice, key)
	// Handle error
	if err != nil {
		s.logger.Error("Sender", "failed to broadcast request ID(%d) with error: %s", task.ID(), err.Error())
		s.failedRequestCh <- FailResponse{task, sdk.TxResponse{}, types.ErrBroadcastFailed.Wrapf(err.Error())}
		return
	} else if resp != nil && resp.Code != 0 {
		s.logger.Error("Sender", "failed to broadcast request ID(%d) with code %d", task.ID(), resp.Code)
		s.failedRequestCh <- FailResponse{task, *resp, types.ErrBroadcastFailed}
		return
	} else if resp == nil {
		s.logger.Error("Sender", "failed to broadcast request ID(%d) no response", task.ID())
		s.failedRequestCh <- FailResponse{task, sdk.TxResponse{}, types.ErrUnknown}
		return
	}

	txHash := resp.TxHash
	s.logger.Info("Sender", "successfully broadcasted request ID(%d) with tx_hash: %s", task.ID(), txHash)

	// Poll for tx confirmation
	et := time.Now().Add(s.timeout)
	for time.Now().Before(et) {
		resp, err = s.client.GetTx(txHash)
		if err != nil {
			time.Sleep(s.pollingDelay)
			continue
		}

		if resp.Code != 0 {
			s.logger.Warning("Sender", "request ID(%d) failed with code %d", task.ID(), resp.Code)
			s.failedRequestCh <- FailResponse{task, *resp, types.ErrBroadcastFailed.Wrapf(resp.RawLog)}
			return
		}

		s.logger.Info("Sender", "request ID(%d) has been confirmed", task.ID())
		s.successfulRequestsCh <- SuccessResponse{task, *resp}
		return
	}
	s.logger.Error("Sender", "request ID(%d) has timed out", task.ID())
	s.failedRequestCh <- FailResponse{task, *resp, types.ErrBroadcastFailed.Wrapf("timed out")}
}
