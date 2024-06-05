package client

import (
	bandtsstypes "github.com/bandprotocol/chain/v2/x/bandtss/types"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	tsstypes "github.com/bandprotocol/chain/v2/x/tss/types"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
)

// OracleResult stores the necessary information for an oracle query result.
type OracleResult struct {
	Result    *oracletypes.Result
	SigningID bandtsstypes.SigningID
}

// SigningResult stores the necessary information for a signing request result.
type SigningResult struct {
	CurrentGroup   SigningInfo
	ReplacingGroup SigningInfo
}

func (s SigningResult) IsReady() bool {
	return s.CurrentGroup.Status == tsstypes.SIGNING_STATUS_SUCCESS &&
		(s.ReplacingGroup.Status == tsstypes.SIGNING_STATUS_UNSPECIFIED ||
			s.ReplacingGroup.Status == tsstypes.SIGNING_STATUS_SUCCESS)
}

// SigningInfo contains signing information.
type SigningInfo struct {
	EVMSignature tsstypes.EVMSignature
	Status       tsstypes.SigningStatus
	PubKey       []byte
	PubNonce     []byte
}

// ChannelInfo stores the necessary information for a channel receiving an event from a chain.
type ChannelInfo struct {
	RemoteAddr string
	EventCh    <-chan ctypes.ResultEvent
}

// SubscriptionInfo stores the necessary information for a subscription channel for a specific events.
type SubscriptionInfo struct {
	Name         string
	Query        string
	ChannelInfos []ChannelInfo
}
