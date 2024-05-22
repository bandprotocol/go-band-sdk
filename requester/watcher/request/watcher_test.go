package request_test

import (
	"fmt"
	"testing"
	"time"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	"github.com/bandprotocol/go-band-sdk/client"
	mockclient "github.com/bandprotocol/go-band-sdk/client/mock"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/request"
	logging "github.com/bandprotocol/go-band-sdk/utils/logging/mock"
)

func TestWatcher(t *testing.T) {
	ctrl := gomock.NewController(t)

	res := client.OracleResult{
		Result: &oracletypes.Result{
			ClientID:       "",
			OracleScriptID: 0,
			Calldata:       nil,
			AskCount:       0,
			MinCount:       0,
			RequestID:      0,
			AnsCount:       0,
			RequestTime:    0,
			ResolveTime:    0,
			ResolveStatus:  1,
			Result:         nil,
		},
		SigningID: 0,
	}
	mockClient := mockclient.NewMockClient(ctrl)
	mockClient.EXPECT().GetResult(gomock.Any()).Return(&res, nil).Times(1)

	mockLogger := logging.NewLogger()

	watcherCh := make(chan request.Task, 100)

	w := request.NewWatcher(mockClient, mockLogger, 5*time.Second, 1*time.Second, watcherCh)
	go w.Start()

	task := request.Task{
		RequestID: 1,
		Msg: oracletypes.MsgRequestData{
			OracleScriptID: 1,
			Calldata:       []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
			AskCount:       16,
			MinCount:       10,
			ClientID:       "test",
			FeeLimit:       sdk.NewCoins(sdk.NewCoin("uband", sdk.NewInt(1000000))),
			PrepareGas:     0,
			ExecuteGas:     0,
			Sender:         "",
		},
	}
	timeout := time.After(5 * time.Second)

	watcherCh <- task

	for {
		select {
		case <-w.SuccessfulRequestsCh():
			return
		case <-w.FailedRequestsCh():
			t.Errorf("expected success, not failure")
			return
		case <-timeout:
			t.Errorf("timed out")
			return
		default:
			continue
		}
	}
}

func TestWatcherWithResolveFailure(t *testing.T) {
	ctrl := gomock.NewController(t)

	res := client.OracleResult{
		Result: &oracletypes.Result{
			ClientID:       "",
			OracleScriptID: 1,
			Calldata:       nil,
			AskCount:       10,
			MinCount:       16,
			RequestID:      1,
			AnsCount:       16,
			RequestTime:    1,
			ResolveTime:    2,
			ResolveStatus:  2,
			Result:         nil,
		},
		SigningID: 0,
	}

	mockClient := mockclient.NewMockClient(ctrl)
	mockClient.EXPECT().GetResult(gomock.Any()).Return(&res, nil).Times(1)
	mockClient.EXPECT().QueryRequestFailureReason(gomock.Any()).Return("unknown", nil).Times(1)

	mockLogger := logging.NewLogger()

	watcherCh := make(chan request.Task, 100)

	w := request.NewWatcher(mockClient, mockLogger, 5*time.Second, 1*time.Second, watcherCh)
	go w.Start()

	msg := oracletypes.MsgRequestData{
		OracleScriptID: 1,
		Calldata:       []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
		AskCount:       16,
		MinCount:       10,
		ClientID:       "test",
		FeeLimit:       sdk.NewCoins(sdk.NewCoin("uband", sdk.NewInt(1000000))),
		PrepareGas:     0,
		ExecuteGas:     0,
		Sender:         "",
	}
	task := request.NewTask(1, 1, msg)
	timeout := time.After(10 * time.Second)

	watcherCh <- task

	for {
		select {
		case <-w.SuccessfulRequestsCh():
			t.Errorf("expected failure, not success")
			return
		case <-w.FailedRequestsCh():
			return
		case <-timeout:
			t.Errorf("timed out")
			return
		default:
			continue
		}
	}
}

func TestWatcherWithTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockClient := mockclient.NewMockClient(ctrl)
	mockClient.EXPECT().GetResult(gomock.Any()).Return(nil, fmt.Errorf("error")).AnyTimes()

	mockLogger := logging.NewLogger()

	watcherCh := make(chan request.Task, 100)

	w := request.NewWatcher(mockClient, mockLogger, 5*time.Second, 1*time.Second, watcherCh)
	go w.Start()

	msg := oracletypes.MsgRequestData{
		OracleScriptID: 1,
		Calldata:       []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
		AskCount:       16,
		MinCount:       10,
		ClientID:       "test",
		FeeLimit:       sdk.NewCoins(sdk.NewCoin("uband", sdk.NewInt(1000000))),
		PrepareGas:     0,
		ExecuteGas:     0,
		Sender:         "",
	}
	task := request.NewTask(1, 1, msg)
	timeout := time.After(10 * time.Second)

	watcherCh <- task

	for {
		select {
		case <-w.SuccessfulRequestsCh():
			t.Errorf("expected failure due to timeout")
			return
		case <-w.FailedRequestsCh():
			return
		case <-timeout:
			t.Errorf("timed out")
			return
		default:
			continue
		}
	}
}
