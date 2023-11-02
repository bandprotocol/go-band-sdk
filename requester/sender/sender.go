package sender

import (
	"time"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bandprotocol/go-band-sdk/requester/client"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils"
)

type Sender struct {
	client client.Client
	logger utils.Logger

	GasPrices        string
	MaxTry           uint64
	RpcPollInterval  time.Duration
	BroadcastTimeout time.Duration

	FreeKeys chan keyring.Info

	// Channel
	Requestc        <-chan types.Request
	SuccessRequestc chan<- types.Response
	FailedRequestc  chan<- types.FailedRequest
}

func NewSender(
	client client.Client,
	logger utils.Logger,
	in <-chan types.Request,
	out chan<- types.Response,
	failed chan<- types.FailedRequest,
	infos []keyring.Info,
	pollInterval time.Duration,
	broadcastTimeout time.Duration,
) *Sender {
	infoc := make(chan keyring.Info, len(infos))

	for _, info := range infos {
		infoc <- info
	}
	return &Sender{
		client:           client,
		logger:           logger,
		Requestc:         in,
		SuccessRequestc:  out,
		FailedRequestc:   failed,
		FreeKeys:         infoc,
		RpcPollInterval:  pollInterval,
		BroadcastTimeout: broadcastTimeout,
	}
}

func (s *Sender) Start() {
	for {
		req := <-s.Requestc
		key := <-s.FreeKeys
		go s.handleRequest(req, key)
	}
}

func (s *Sender) handleRequest(req types.Request, key keyring.Info) {
	defer func() {
		s.FreeKeys <- key
	}()

	// Mutate msg request to current sender should be ok for this use case
	req.Msg.Sender = key.GetAddress().String()

	res, err := s.client.SendRequest(req.Msg, key)
	if err != nil {
		// TODO: Fail request event
		s.FailedRequestc <- types.NewFailedRequest(req, types.ErrBroadcast.Wrapf("with error %s", err))
	}

	// Check tx response from sync broadcast
	if res.Code != 0 {
		switch res.Codespace {
		case sdkerrors.RootCodespace:
			switch res.Code {
			case sdkerrors.ErrInsufficientFee.ABCICode():
				s.FailedRequestc <- types.NewFailedRequest(req, types.ErrInsufficientFunds)
			default:
				s.FailedRequestc <- types.NewFailedRequest(req, types.ErrUnexpected)
			}
		case oracletypes.ModuleName:
			switch res.Code {
			default:
				s.FailedRequestc <- types.NewFailedRequest(req, types.ErrUnexpected)
			}
		default:
			s.FailedRequestc <- types.NewFailedRequest(req, types.ErrUnexpected)
		}
		return
	}

	txHash := res.TxHash

	start := time.Now()
	for time.Since(start) < s.BroadcastTimeout {
		time.Sleep(s.RpcPollInterval)

		res, err := s.client.GetTx(txHash)
		if err != nil {
			s.logger.Debug("Failed to query tx with error", err.Error())
			continue
		}

		// TODO: Extract info for event
		_, _, _, err = decodeTxFromRes(res)
		if err != nil {
			// It's not expected behavior
			s.logger.Critical("Cannot decode tx", "Cannot decode Tx")
			return
		}

		if res.Code != 0 {
			switch res.Codespace {
			case sdkerrors.RootCodespace:
				switch res.Code {
				case sdkerrors.ErrInsufficientFee.ABCICode():
					s.FailedRequestc <- types.NewFailedRequest(req, types.ErrInsufficientFunds)
				default:
					s.FailedRequestc <- types.NewFailedRequest(req, types.ErrUnexpected)
				}
			case oracletypes.ModuleName:
				switch res.Code {
				default:
					s.FailedRequestc <- types.NewFailedRequest(req, types.ErrUnexpected)
				}
			default:
				s.FailedRequestc <- types.NewFailedRequest(req, types.ErrUnexpected)
			}
			return
		}

		rid, err := GetRequestID(res.Logs[0].Events)
		if err != nil {
			s.FailedRequestc <- types.NewFailedRequest(req, types.ErrUnexpected)
			return
		}

		// dsFee, err := GetDsFee(res.Logs[0].Events)
		// if err != nil {
		// 	errDetail := fmt.Sprintf("Failed to parse DS fees on %s: %s", res.TxHash, err.Error())
		// 	s.incidentService.TriggerIncident(common.ParseDSFeesFailed.With(errDetail))
		// 	return
		// }

		// TODO: Add event request successful
		s.logger.Info(
			"Tx send successful",
			"Tx Hash %s with request ID %d (Client ID: %s).",
			txHash,
			rid,
			req.Msg.ClientID,
		)

		s.SuccessRequestc <- types.Response{Id: req.Id, TxHash: txHash, RequestId: rid}
	}

	s.FailedRequestc <- types.NewFailedRequest(req, types.ErrTxNotConfirm)
}
