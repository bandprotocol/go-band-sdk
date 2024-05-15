package client

import (
	bandtsstypes "github.com/bandprotocol/chain/v2/x/bandtss/types"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
)

type OracleResult struct {
	Result    *oracletypes.Result
	SigningID bandtsstypes.SigningID
}

type SigningResult struct {
	CurrentGroupSigning   []byte
	CurrentGroupPubKey    []byte
	ReplacingGroupSigning []byte
	ReplacingGroupPubKey  []byte
}
