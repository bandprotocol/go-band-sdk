package client

import (
	bandtsstypes "github.com/bandprotocol/chain/v2/x/bandtss/types"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	tsstypes "github.com/bandprotocol/chain/v2/x/tss/types"
)

// OracleResult stores necessary information for an oracle query result.
type OracleResult struct {
	Result    *oracletypes.Result
	SigningID bandtsstypes.SigningID
}

// SigningResult stores necessary information for a signing request result.
type SigningResult struct {
	CurrentGroup   SigningInfo
	ReplacingGroup SigningInfo
}

// SigningInfo contains signing information.
type SigningInfo struct {
	Signing []byte
	Status  tsstypes.SigningStatus
	PubKey  []byte
}
