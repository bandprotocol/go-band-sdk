package subscriber

import ctypes "github.com/cometbft/cometbft/rpc/core/types"

const (
	EventInfoPrefixKey = "eventInfo:"
	EventIDPrefixKey   = "eventID:"
)

type EventInfo struct {
	Name    string
	Query   string
	StopChs []chan struct{}
	OutCh   chan ctypes.ResultEvent
}
