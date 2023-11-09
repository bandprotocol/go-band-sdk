package sender

import (
	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/requester/middleware"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logger"
)

type Sender struct {
	client client.Clienter
	logger logger.Logger

	retryMiddlewares []middleware.Retry

	freeKeys chan keyring.Info

	// Channel
	requestQueueCh       chan types.Request
	successfulRequestsCh chan types.RequestResult
	failedRequestCh      chan types.FailedRequest
}

func NewSender(
	client client.Clienter,
	logger logger.Logger,
	RequestQueueCh chan types.Request,
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
		retryMiddlewares:     make([]middleware.Retry, 0),
		freeKeys:             freeKeys,
		requestQueueCh:       RequestQueueCh,
		successfulRequestsCh: make(chan types.RequestResult),
		failedRequestCh:      make(chan types.FailedRequest),
	}, nil
}

func (s *Sender) WithRetryMiddleware(middlewares []middleware.Retry) {
	s.retryMiddlewares = middlewares
}

func (s *Sender) SuccessRequestsCh() <-chan types.RequestResult {
	return s.successfulRequestsCh
}

func (s *Sender) FailedRequestsCh() <-chan types.FailedRequest {
	return s.failedRequestCh
}

func (s *Sender) Start() {
	for {
		req := <-s.requestQueueCh
		key := <-s.freeKeys
		go s.request(req, key)
	}
}

func (s *Sender) request(req types.Request, key keyring.Info) {
	var fr types.FailedRequest
	var r types.RequestResult
	var retry = true

	defer func() {
		s.freeKeys <- key
	}()

	defer func() {
		fr.RequestResult = r
		if retry {
			s.executeRetryMiddleware(fr)
		}
	}()

	// Mutate the msg sender to the actual sender
	req.Msg.Sender = key.GetAddress().String()

	// Attempt to send the request
	res, err := s.client.SendRequest(req.Msg, key)
	r.TxResponse = *res
	if res.Code != 0 || err != nil {
		fr.Error = err
		return
	}

	retry = false
	s.successfulRequestsCh <- types.RequestResult{TxResponse: *res}
}

func (s *Sender) executeRetryMiddleware(fr types.FailedRequest) {
	for _, mw := range s.retryMiddlewares {
		fr, next := mw.Call(fr)
		if !next {
			s.failedRequestCh <- fr
			return
		}
	}
	s.requestQueueCh <- fr.Request
}
