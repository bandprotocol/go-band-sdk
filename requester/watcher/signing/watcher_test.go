package signing_test

import (
	"fmt"
	"testing"
	"time"

	bandtsstypes "github.com/bandprotocol/chain/v2/x/bandtss/types"
	feedstypes "github.com/bandprotocol/chain/v2/x/feeds/types"
	tsstypes "github.com/bandprotocol/chain/v2/x/tss/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	"github.com/bandprotocol/go-band-sdk/client"
	mockclient "github.com/bandprotocol/go-band-sdk/client/mock"
	"github.com/bandprotocol/go-band-sdk/requester/watcher/signing"
	logging "github.com/bandprotocol/go-band-sdk/utils/logging/mock"
)

func TestWatcherSuccess(t *testing.T) {
	testCases := []struct {
		name string
		res  client.SigningResult
	}{
		{
			name: "only currentGroup",
			res: client.SigningResult{
				CurrentGroup: client.SigningInfo{
					EVMSignature: tsstypes.EVMSignature{
						RAddress:  []byte("rAddress1"),
						Signature: []byte("signature1"),
					},
					Status:   tsstypes.SIGNING_STATUS_SUCCESS,
					PubKey:   []byte("pubkey1"),
					PubNonce: []byte("pubnonce1"),
				},
				ReplacingGroup: client.SigningInfo{},
			},
		},
		{
			name: "both current and replacing group",
			res: client.SigningResult{
				CurrentGroup: client.SigningInfo{
					EVMSignature: tsstypes.EVMSignature{
						RAddress:  []byte("rAddress1"),
						Signature: []byte("signature1"),
					},
					Status:   tsstypes.SIGNING_STATUS_SUCCESS,
					PubKey:   []byte("pubkey1"),
					PubNonce: []byte("pubnonce1"),
				},
				ReplacingGroup: client.SigningInfo{
					EVMSignature: tsstypes.EVMSignature{
						RAddress:  []byte("rAddress2"),
						Signature: []byte("signature2"),
					},
					Status:   tsstypes.SIGNING_STATUS_SUCCESS,
					PubKey:   []byte("pubkey2"),
					PubNonce: []byte("pubnonce2"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mockclient.NewMockClient(ctrl)
			mockClient.EXPECT().GetSignature(gomock.Any()).Return(&tc.res, nil).Times(1) // #nosec G601

			mockLogger := logging.NewLogger()

			watcherCh := make(chan signing.Task, 100)

			w := signing.NewWatcher(mockClient, mockLogger, 5*time.Second, 1*time.Second, watcherCh)
			go w.Start()

			msg := &bandtsstypes.MsgRequestSignature{
				FeeLimit: sdk.NewCoins(sdk.NewCoin("uband", sdk.NewInt(1000000))),
				Sender:   "",
			}
			content := feedstypes.NewFeedSignatureOrder(
				[]string{"crypto_price.ethusd", "crypto_price.usdtusd"},
				feedstypes.FEEDS_TYPE_DEFAULT,
			)
			msg.SetContent(content)

			task := signing.NewTask(1, 1, msg)
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
		})
	}
}

func TestWatcherWithResolveFailure(t *testing.T) {
	testCases := []struct {
		name string
		res  client.SigningResult
	}{
		{
			name: "fail due to current group status fallen",
			res: client.SigningResult{
				CurrentGroup: client.SigningInfo{
					EVMSignature: tsstypes.EVMSignature{
						RAddress:  []byte("rAddress1"),
						Signature: []byte("signature1"),
					},
					Status:   tsstypes.SIGNING_STATUS_FALLEN,
					PubKey:   []byte("pubkey1"),
					PubNonce: []byte("pubnonce1"),
				},
				ReplacingGroup: client.SigningInfo{},
			},
		},
		{
			name: "fail due to current group status expired",
			res: client.SigningResult{
				CurrentGroup: client.SigningInfo{
					EVMSignature: tsstypes.EVMSignature{
						RAddress:  []byte("rAddress1"),
						Signature: []byte("signature1"),
					},
					Status:   tsstypes.SIGNING_STATUS_EXPIRED,
					PubKey:   []byte("pubKey1"),
					PubNonce: []byte("pubNonce1"),
				},
				ReplacingGroup: client.SigningInfo{},
			},
		},
		{
			name: "fail due to replacing group status fallen",
			res: client.SigningResult{
				CurrentGroup: client.SigningInfo{
					EVMSignature: tsstypes.EVMSignature{
						RAddress:  []byte("rAddress1"),
						Signature: []byte("signature1"),
					},
					Status:   tsstypes.SIGNING_STATUS_SUCCESS,
					PubKey:   []byte("pubKey1"),
					PubNonce: []byte("pubNonce1"),
				},
				ReplacingGroup: client.SigningInfo{
					EVMSignature: tsstypes.EVMSignature{
						RAddress:  []byte("rAddress2"),
						Signature: []byte("signature2"),
					},
					Status:   tsstypes.SIGNING_STATUS_FALLEN,
					PubKey:   []byte("pubKey2"),
					PubNonce: []byte("pubNonce2"),
				},
			},
		},
		{
			name: "fail due to replacing group status expired",
			res: client.SigningResult{
				CurrentGroup: client.SigningInfo{
					EVMSignature: tsstypes.EVMSignature{
						RAddress:  []byte("rAddress1"),
						Signature: []byte("signature1"),
					},
					Status:   tsstypes.SIGNING_STATUS_SUCCESS,
					PubKey:   []byte("pubKey1"),
					PubNonce: []byte("pubNonce1"),
				},
				ReplacingGroup: client.SigningInfo{
					EVMSignature: tsstypes.EVMSignature{
						RAddress:  []byte("rAddress2"),
						Signature: []byte("signature2"),
					},
					Status:   tsstypes.SIGNING_STATUS_EXPIRED,
					PubKey:   []byte("pubKey2"),
					PubNonce: []byte("pubNonce2"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mockclient.NewMockClient(ctrl)
			mockClient.EXPECT().GetSignature(gomock.Any()).Return(&tc.res, nil).Times(1) // #nosec G601

			mockLogger := logging.NewLogger()

			watcherCh := make(chan signing.Task, 100)

			w := signing.NewWatcher(mockClient, mockLogger, 5*time.Second, 1*time.Second, watcherCh)
			go w.Start()

			msg := &bandtsstypes.MsgRequestSignature{
				FeeLimit: sdk.NewCoins(sdk.NewCoin("uband", sdk.NewInt(1000000))),
				Sender:   "",
			}
			content := feedstypes.NewFeedSignatureOrder(
				[]string{"crypto_price.ethusd", "crypto_price.usdtusd"},
				feedstypes.FEEDS_TYPE_DEFAULT,
			)
			msg.SetContent(content)

			task := signing.NewTask(1, 1, msg)
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
		})
	}
}

func TestWatcherWithTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockClient := mockclient.NewMockClient(ctrl)
	mockClient.EXPECT().GetSignature(gomock.Any()).Return(nil, fmt.Errorf("error")).AnyTimes()

	mockLogger := logging.NewLogger()

	watcherCh := make(chan signing.Task, 100)

	w := signing.NewWatcher(mockClient, mockLogger, 5*time.Second, 1*time.Second, watcherCh)
	go w.Start()

	msg := &bandtsstypes.MsgRequestSignature{
		FeeLimit: sdk.NewCoins(sdk.NewCoin("uband", sdk.NewInt(1000000))),
		Sender:   "",
	}
	content := feedstypes.NewFeedSignatureOrder(
		[]string{"crypto_price.ethusd", "crypto_price.usdtusd"},
		feedstypes.FEEDS_TYPE_DEFAULT,
	)
	msg.SetContent(content)

	task := signing.NewTask(1, 1, msg)
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
			t.Errorf("client not being stopped before timed out")
			return
		default:
			continue
		}
	}
}
