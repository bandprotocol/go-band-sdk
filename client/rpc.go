package client

import (
	"context"
	"fmt"
	"strconv"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/bandprotocol/go-band-sdk/utils/logger"
)

// RPC stores many clients to communicate with BandChain
type RPC struct {
	ctx       client.Context
	txFactory tx.Factory
	nodes     []*rpchttp.HTTP
	keyring   keyring.Keyring

	logger logger.Logger
}

func NewRPC(
	logger logger.Logger,
	endpoints []string,
	chainId string,
	timeout string,
	gasPrice string,
	keyring keyring.Keyring,
) (*RPC, error) {
	nodes := make([]*rpchttp.HTTP, 0, len(endpoints))
	for _, endpoint := range endpoints {
		node, err := newRPCClient(endpoint, timeout)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return &RPC{
		ctx:       NewClientCtx(chainId),
		txFactory: createTxFactory(chainId, gasPrice, keyring),
		nodes:     nodes,
		keyring:   keyring,
		logger:    logger,
	}, nil
}

// GetAccount find max account sequence from multiple clients
func (c RPC) GetAccount(account sdk.AccAddress) (client.Account, error) {
	var maxSeqAcc client.Account

	resultCh := make(chan client.Account)
	failCh := make(chan struct{})

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			account, err := getAccount(c.ctx.WithClient(node), account)
			if err != nil {
				failCh <- struct{}{}
			} else {
				resultCh <- account
			}
		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case acc := <-resultCh:
			if maxSeqAcc == nil || acc.GetAccountNumber() > maxSeqAcc.GetAccountNumber() {
				maxSeqAcc = acc
			}
		case <-failCh:
		}

	}

	if maxSeqAcc == nil {
		return nil, fmt.Errorf("failed to get account from all endpoints")
	}

	return maxSeqAcc, nil
}

func (c RPC) GetResult(id uint64) (*oracletypes.Result, error) {
	resultCh := make(chan *oracletypes.Result, len(c.nodes))
	failCh := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Getting request", "Try to get request from %s", node.Remote())
			res, err := getRequest(c.ctx.WithClient(node), id)
			if err != nil {
				c.logger.Warning(
					"Fail to get request from single endpoint",
					"Fail to get request from %s with error %s",
					node.Remote(), err,
				)
				failCh <- struct{}{}
				return
			}
			if res.Result != nil {
				resultCh <- res.Result
			} else {
				c.logger.Debug(
					"Request has not been resolved yet",
					"Get result from %s, but request #%d has not been resolved yet",
					node.Remote(), id,
				)
				failCh <- struct{}{}
			}

		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case res := <-resultCh:
			// Return the first result that we found
			return res, nil
		case <-failCh:
		}
	}
	// If cannot get result from all nodes return error
	return nil, fmt.Errorf("failed to get result from all endpoints")
}

func (c RPC) GetTx(txHash string) (*sdk.TxResponse, error) {
	resultCh := make(chan *sdk.TxResponse, len(c.nodes))
	failCh := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Getting txutil", "Try to find txutil(%s) from %s", txHash, node.Remote())
			res, err := getTx(c.ctx.WithClient(node), txHash)
			if err != nil {
				c.logger.Debug(
					"Fail to get transaction from single endpoint",
					"Fail to get transaction from %s, it might be wait for included in block",
					node.Remote(),
				)
				failCh <- struct{}{}
				return
			}
			resultCh <- res
		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case res := <-resultCh:
			// Return the first result that we found
			return res, nil
		case <-failCh:
		}
	}
	// If cannot get transaction from all nodes return error
	return nil, fmt.Errorf("failed to find transaction from all endpoints")
}

func (c RPC) BlockSearch(query string, page *int, perPage *int, orderBy string) (*ctypes.ResultBlockSearch, error) {
	resultCh := make(chan *ctypes.ResultBlockSearch)
	failCh := make(chan struct{})

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			blockResult, err := node.BlockSearch(context.Background(), query, page, perPage, orderBy)
			if err != nil || blockResult.TotalCount == 0 {
				failCh <- struct{}{}
			} else {
				resultCh <- blockResult
			}
		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case res := <-resultCh:
			return res, nil
		case <-failCh:
		}
	}

	return nil, fmt.Errorf("failed to block search from all endpoints")
}

func (c RPC) GetBlockResult(height int64) (*ctypes.ResultBlockResults, error) {
	resultCh := make(chan *ctypes.ResultBlockResults)
	failCh := make(chan struct{})

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			blockResult, err := node.BlockResults(context.Background(), &height)
			if err != nil {
				failCh <- struct{}{}
			} else {
				resultCh <- blockResult
			}
		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case res := <-resultCh:
			return res, nil
		case <-failCh:
		}
	}

	return nil, fmt.Errorf("failed to find block result from all endpoints")
}

func (c RPC) QueryRequestFailureReason(id uint64) (string, error) {
	query := fmt.Sprintf("resolve.resolve_status=2 AND resolve.id=%v", id)
	blockSearchResult, err := c.BlockSearch(query, nil, nil, "")
	if err != nil {
		return "", err
	}

	if blockSearchResult.TotalCount == 0 {
		return "", fmt.Errorf("no request data tx found")
	}

	height := blockSearchResult.Blocks[0].Block.Height
	blockResult, err := c.GetBlockResult(height)
	if err != nil {
		return "", err
	}

	for _, event := range blockResult.EndBlockEvents {
		if event.Type == "reason" {
			rid := uint64(0)
			resolveStatus := uint64(0)
			reason := ""

			for _, attr := range event.Attributes {
				switch string(attr.Key) {
				case "id":
					rid, _ = strconv.ParseUint(string(attr.Value), 10, 64)
				case "resolve_status":
					resolveStatus, _ = strconv.ParseUint(string(attr.Value), 10, 64)
				case "reason":
					reason = string(attr.Value)
				}
			}

			if reason != "" && rid == id && resolveStatus == uint64(oracletypes.RESOLVE_STATUS_FAILURE) {
				return reason, nil
			}
		}
	}

	return "", fmt.Errorf("no reason found")
}

func (c RPC) GetSignature(id uint64) ([]byte, error) {
	// TODO: Implement when need TSS signature
	return []byte{}, nil
}

func (c RPC) SendRequest(msg *oracletypes.MsgRequestData, gasPrice float64, key keyring.Info) (*sdk.TxResponse, error) {
	// Get account to get nonce of sender first
	acc, err := c.GetAccount(key.GetAddress())
	if err != nil {
		return nil, err
	}

	txf := c.txFactory.WithAccountNumber(acc.GetAccountNumber()).WithSequence(acc.GetSequence())
	// Estimate gas
	gasCh := make(chan uint64, len(c.nodes))
	failCh := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Estimating tx gas", "From %s", node.Remote())
			gas, err := estimateGas(c.ctx.WithClient(node), txf, msg)
			if err != nil {
				c.logger.Warning(
					"Fail to estimate tx gas from single endpoint",
					"From %s",
					node.Remote(),
				)
				failCh <- struct{}{}
			} else {
				gasCh <- gas
			}
		}(node)
	}

	gas := uint64(0)

Gas:
	for i := 0; i < len(c.nodes); i++ {
		select {
		case gas = <-gasCh:
			// Return the first result that we found
			break Gas
		case <-failCh:
		}
	}

	// If all endpoint fail to estimate gas return error
	// TODO: Find a way to extract error to meaningful response that able to handle by caller
	if gas == 0 {
		return nil, fmt.Errorf("fail to estimate gas")
	}

	// Build unsigned txutil with estimated gas
	txf = txf.WithGas(gas)
	txf.WithGasPrices(fmt.Sprintf("%v0uband", gasPrice))
	txb, err := tx.BuildUnsignedTx(txf, msg)
	if err != nil {
		return nil, err
	}

	// Sign txutil
	err = tx.Sign(txf, key.GetName(), txb, false)
	if err != nil {
		return nil, err
	}

	// Generate the transaction bytes
	txBytes, err := c.ctx.TxConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return nil, err
	}

	resultCh := make(chan *sdk.TxResponse, len(c.nodes))
	errorCh := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Broadcasting tx", "Try to broadcast tx to %s", node.Remote())
			res, err := c.ctx.WithClient(node).BroadcastTx(txBytes)
			if err != nil {
				c.logger.Warning(
					"Fail to broadcast to single endpoint",
					"to %s, with error %s",
					node.Remote(), err,
				)
				errorCh <- struct{}{}
				return
			}
			resultCh <- res
		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case res := <-resultCh:
			// Return the first tx response that we found
			return res, nil
		case <-errorCh:
		}
	}
	// If the tx cannot broadcast to all nodes, return an error
	return nil, fmt.Errorf("failed to find transaction from all endpoints")
}
