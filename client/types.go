package client

import (
	bandtsstypes "github.com/bandprotocol/chain/v2/x/bandtss/types"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	tsstypes "github.com/bandprotocol/chain/v2/x/tss/types"
	tmbytes "github.com/cometbft/cometbft/libs/bytes"
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

// SigningInfo contains signing information.
type SigningInfo struct {
	Signing []byte
	Status  tsstypes.SigningStatus
	PubKey  []byte
}

// SingleProof represents a single proof in a response.
type SingleProof struct {
	BlockHeight string `json:"block_height"`
}

// SingleProofResponse represents the response containing a SingleProof and the EVM proof bytes.
type SingleProofResponse struct {
	Proof         SingleProof      `json:"proof"`
	EVMProofBytes tmbytes.HexBytes `json:"evm_proof_bytes"`
}

// Proof represents a full proof in a response, containing a height and a SingleProofResponse.
type Proof struct {
	Height string              `json:"height"`
	Result SingleProofResponse `json:"result"`
}
