package sender

import (
	"errors"
	"strings"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	owasm "github.com/bandprotocol/go-owasm/api"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bandprotocol/go-band-sdk/requester/client"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logger"
)

type Sender struct {
	client client.Clienter
	logger logger.Logger

	retryMiddlewares []RetryMiddleware

	freeKeys chan keyring.Info

	// Channel
	requestQueueCh       chan types.Request
	successfulRequestsCh chan<- types.Response
	failedRequestCh      chan<- types.FailedRequest
}

func NewSender(
	client client.Client,
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
		retryMiddlewares:     make([]RetryMiddleware, 0),
		freeKeys:             freeKeys,
		requestQueueCh:       RequestQueueCh,
		successfulRequestsCh: make(chan<- types.Response),
		failedRequestCh:      make(chan<- types.FailedRequest),
	}, nil
}

func (s *Sender) WithRetryMiddleware(middlewares []RetryMiddleware) {
	s.retryMiddlewares = middlewares
}

func (s *Sender) SuccessRequestsCh() chan<- types.Response {
	return s.successfulRequestsCh
}

func (s *Sender) FailedRequestsCh() chan<- types.FailedRequest {
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
	var retry = true

	defer func() {
		s.freeKeys <- key
	}()

	defer func() {
		if retry {
			s.executeRetryMiddleware(&fr)
		}
	}()

	// Mutate the msg sender to the actual sender
	req.Msg.Sender = key.GetAddress().String()

	// Attempt to send the request
	res, err := s.client.SendRequest(req.Msg, key)
	if err != nil {
		fr = parseCheckTxErrorIntoFailedRequest(req, err)
		return
	}

	// Check tx response from sync broadcast
	if res.Code != 0 {
		fr = parseCheckTxResponseErrorIntoFailedRequest(req, *res)
		return
	}

	retry = false
	s.successfulRequestsCh <- types.Response{TxHash: res.TxHash}
}

func (s *Sender) executeRetryMiddleware(fr *types.FailedRequest) {
	for _, mw := range s.retryMiddlewares {
		next := mw.Call(fr)
		if !next {
			s.failedRequestCh <- *fr
			return
		}
	}
	s.requestQueueCh <- fr.Request
}

func parseCheckTxErrorIntoFailedRequest(req types.Request, err error) types.FailedRequest {
	// TODO: Log error
	var failedRequest types.FailedRequest
	switch {
	case errors.Is(err, oracletypes.ErrBadWasmExecution):
		// Out of prepare gas
		if strings.Contains(err.Error(), owasm.ErrOutOfGas.Error()) {
			failedRequest = types.NewFailedRequest(req, types.ErrOutOfPrepareGas)
		}
	default:
		failedRequest = types.NewFailedRequest(req, types.ErrUnexpected.Wrapf("with error %s", err))
	}
	return failedRequest
}

func parseCheckTxResponseErrorIntoFailedRequest(req types.Request, res sdk.TxResponse) types.FailedRequest {
	// TODO: Log error
	var failedRequest types.FailedRequest
	switch res.Codespace {
	case sdkerrors.RootCodespace:
		// Handle root errors
		switch res.Code {
		case sdkerrors.ErrInsufficientFee.ABCICode():
			// Balance not sufficient to execute transaction
			failedRequest = types.NewFailedRequest(req, types.ErrInsufficientFunds)
		default:
			failedRequest = types.NewFailedRequest(req, types.ErrUnexpected)
		}
	case oracletypes.ModuleName:
		// Handle oracle errors
		switch res.Code {
		default:
			failedRequest = types.NewFailedRequest(req, types.ErrUnexpected)
		}
	default:
		failedRequest = types.NewFailedRequest(req, types.ErrUnexpected)
	}
	return failedRequest
}
