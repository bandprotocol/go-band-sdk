package sender_test

import (
	"fmt"
	"testing"
	"time"

	band "github.com/bandprotocol/chain/v2/app"
	"github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/bandprotocol/go-band-sdk/client"
	mockclient "github.com/bandprotocol/go-band-sdk/client/mock"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/utils/logging"
	mocklogging "github.com/bandprotocol/go-band-sdk/utils/logging/mock"
)

func SetupSender(cl client.Client, l logging.Logger, reqCh chan sender.Task) (*sender.Sender, error) {
	kr := keyring.NewInMemory()
	mnemonic := "child across insect stone enter jacket bitter citizen inch wear breeze adapt come attend vehicle caught wealth junk cloth velvet wheat curious prize panther"
	hdPath := hd.CreateHDPath(band.Bip44CoinType, 0, 0)
	_, err := kr.NewAccount("sender1", mnemonic, "", hdPath.String(), hd.Secp256k1)
	if err != nil {
		return nil, err
	}

	return sender.NewSender(cl, l, kr, 1.0, 5*time.Second, 1*time.Second, reqCh, 1, 1)
}

func TestSenderWithSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Mock dependencies
	mockClient := mockclient.NewMockClient(ctrl)
	mockResult := sdk.TxResponse{
		Height:    0,
		TxHash:    "abc",
		Codespace: "band",
		Code:      0,
		Data:      "",
		RawLog:    "",
		Logs:      nil,
		Info:      "",
		GasWanted: 4,
		GasUsed:   10,
		Tx:        nil,
		Timestamp: "",
		Events:    nil,
	}

	mockClient.EXPECT().SendRequest(gomock.Any(), 1.0, gomock.Any()).Return(&mockResult, nil).Times(1)
	mockClient.EXPECT().GetTx("abc").Return(&mockResult, nil).Times(1)

	mockLogger := mocklogging.NewLogger()
	mockTask := sender.NewTask(1, types.MsgRequestData{})

	// Create channels
	requestQueueCh := make(chan sender.Task, 1)

	// Create a new sender
	s, err := SetupSender(mockClient, mockLogger, requestQueueCh)
	assert.NoErrorf(t, err, "expected no error, got %v", err)

	// Test Start method
	go s.Start()

	// Send a task to the requestQueueCh
	requestQueueCh <- mockTask
	// Check if the task was processed successfully
	// set time limit to 10 seconds
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-s.SuccessRequestsCh():
			return
		case <-s.FailedRequestsCh():
			t.Errorf("expected a successful response")
		case <-timeout:
			t.Errorf("timed out")
		default:
			continue
		}
	}
}

func TestSenderWithFailure(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Mock dependencies
	mockClient := mockclient.NewMockClient(ctrl)
	mockResult := sdk.TxResponse{
		Height:    0,
		TxHash:    "abc",
		Codespace: "band",
		Code:      1,
		Data:      "",
		RawLog:    "",
		Logs:      nil,
		Info:      "",
		GasWanted: 4,
		GasUsed:   10,
		Tx:        nil,
		Timestamp: "",
		Events:    nil,
	}

	mockClient.EXPECT().SendRequest(gomock.Any(), 1.0, gomock.Any()).Return(&mockResult, nil).Times(1)

	mockLogger := mocklogging.NewLogger()
	mockTask := sender.NewTask(1, types.MsgRequestData{})

	// Create channels
	requestQueueCh := make(chan sender.Task, 1)

	// Create a new sender
	s, err := SetupSender(mockClient, mockLogger, requestQueueCh)
	assert.NoErrorf(t, err, "expected no error, got %v", err)

	// Test Start method
	go s.Start()

	// Send a task to the requestQueueCh
	requestQueueCh <- mockTask
	// Check if the task was processed successfully
	// set time limit to 10 seconds
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-s.SuccessRequestsCh():
			t.Errorf("expected a failed response")
		case <-s.FailedRequestsCh():
			return
		case <-timeout:
			t.Errorf("timed out")
		default:
			continue
		}
	}
}

func TestSenderWithClientError(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Mock dependencies
	mockClient := mockclient.NewMockClient(ctrl)

	mockClient.EXPECT().SendRequest(gomock.Any(), 1.0, gomock.Any()).Return(nil, fmt.Errorf("error")).Times(1)

	mockLogger := mocklogging.NewLogger()
	mockTask := sender.NewTask(1, types.MsgRequestData{})

	// Create channels
	requestQueueCh := make(chan sender.Task, 1)

	// Create a new sender
	s, err := SetupSender(mockClient, mockLogger, requestQueueCh)
	assert.NoErrorf(t, err, "expected no error, got %v", err)

	// Test Start method
	go s.Start()

	// Send a task to the requestQueueCh
	requestQueueCh <- mockTask
	// Check if the task was processed successfully
	// set time limit to 10 seconds
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-s.SuccessRequestsCh():
			t.Errorf("expected a failed response")
		case <-s.FailedRequestsCh():
			return
		case <-timeout:
			t.Errorf("timed out")
		default:
			continue
		}
	}
}
