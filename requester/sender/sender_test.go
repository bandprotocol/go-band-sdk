package sender_test

import (
	"testing"
	"time"

	band "github.com/bandprotocol/chain/v2/app"
	"github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/bandprotocol/go-band-sdk/client/mock"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	mock2 "github.com/bandprotocol/go-band-sdk/utils/logging/mock"
)

func TestSender(t *testing.T) {
	ctrl := gomock.NewController(t)

	kr := keyring.NewInMemory()
	mnemonic := "child across insect stone enter jacket bitter citizen inch wear breeze adapt come attend vehicle caught wealth junk cloth velvet wheat curious prize panther"
	hdPath := hd.CreateHDPath(band.Bip44CoinType, 0, 0)
	kr.NewAccount("sender1", mnemonic, "", hdPath.String(), hd.Secp256k1)

	// Mock dependencies
	mockClient := mock.NewMockClient(ctrl)
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
	mockLogger := mock2.NewLogger()

	mockTask := sender.NewTask(1, types.MsgRequestData{})

	// Create channels
	requestQueueCh := make(chan sender.Task, 1)

	// Create a new sender
	s, err := sender.NewSender(
		mockClient,
		mockLogger,
		requestQueueCh,
		1,
		1,
		1.0,
		kr,
	)
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
		case _ = <-s.SuccessRequestsCh():
			return
		case _ = <-s.FailedRequestsCh():
			t.Errorf("expected a successful response")
		case <-timeout:
			t.Errorf("timed out")
		default:
			continue
		}
	}
}
