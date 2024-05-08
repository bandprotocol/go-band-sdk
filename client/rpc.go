package client

import (
	"context"
	"fmt"
	"strconv"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

// TODO: properly implement error channels

var _ Client = &RPC{}

// RPC implements Clients by using multiple RPC nodes
type RPC struct {
	ctx       client.Context
	txFactory tx.Factory
	nodes     []*rpchttp.HTTP
	keyring   keyring.Keyring

	logger logging.Logger
}

// NewRPC creates new RPC client
func NewRPC(
	logger logging.Logger,
	endpoints []string,
	chainID string,
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
		ctx:       NewClientCtx(chainID),
		txFactory: createTxFactory(chainID, gasPrice, keyring),
		nodes:     nodes,
		keyring:   keyring,
		logger:    logger,
	}, nil
}

// GetAccount find max account sequence from multiple clients
func (c RPC) GetAccount(account sdk.AccAddress) (client.Account, error) {
	var maxSeqAcc client.Account

	resultCh := make(chan client.Account, len(c.nodes))
	failCh := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			acc, err := getAccount(c.ctx.WithClient(node), account)
			if err != nil {
				failCh <- struct{}{}
			} else {
				resultCh <- acc
			}
		}(node)
	}

	for range c.nodes {
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

// GetResult find result from multiple clients
func (c RPC) GetResult(id uint64) (*oracletypes.Result, error) {
	resultCh := make(chan *oracletypes.Result, len(c.nodes))
	failCh := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Getting request", "Try to get request from %s", node.Remote())
			res, err := getRequest(c.ctx.WithClient(node), id)
			if err != nil {
				c.logger.Warning(
					"Fail to get request",
					"Fail to get request from %s with error %s",
					node.Remote(), err,
				)
				failCh <- struct{}{}
				return
			}

			if res.Result == nil {
				c.logger.Debug(
					"Request has not been resolved yet",
					"Get result from %s, but request #%d has not been resolved yet",
					node.Remote(), id,
				)
				failCh <- struct{}{}
				return
			}

			resultCh <- res.Result
		}(node)
	}

	for range c.nodes {
		select {
		case res := <-resultCh:
			// Return the first result that we found
			return res, nil
		case <-failCh:
		}
	}
	// If unable to get result from all nodes, return error
	return nil, fmt.Errorf("failed to get result from all endpoints")
}

func (c RPC) GetTx(txHash string) (*sdk.TxResponse, error) {
	resultCh := make(chan *sdk.TxResponse, len(c.nodes))
	failCh := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Getting tx", "Try to find tx (%s) from %s", txHash, node.Remote())
			res, err := getTx(c.ctx.WithClient(node), txHash)
			if err != nil {
				c.logger.Debug(
					"Fail to get transaction",
					"Fail to get transaction from %s, it might be wait for included in block %s",
					node.Remote(), err,
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
	// If unable to get transaction from all nodes, return error
	return nil, fmt.Errorf("failed to find transaction from all endpoints")
}

func (c RPC) GetBlockResult(height int64) (*ctypes.ResultBlockResults, error) {
	resultCh := make(chan *ctypes.ResultBlockResults)
	failCh := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			blockResult, err := node.BlockResults(ctx, &height)
			if err != nil {
				c.logger.Debug(
					"Fail to get block result",
					"Fail to get block result from %s with error %s",
					node.Remote(), err,
				)
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
	blockSearchResult, err := c.blockSearch(query, nil, nil, "")
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
		if event.Type == oracletypes.EventTypeResolve {
			rid := uint64(0)
			resolveStatus := uint64(0)
			reason := ""

			for _, attr := range event.Attributes {
				switch attr.Key {
				case oracletypes.AttributeKeyID:
					rid, err = strconv.ParseUint(attr.Value, 10, 64)
					if err != nil {
						return "", fmt.Errorf("unable to parse request id")
					}
				case oracletypes.AttributeKeyResolveStatus:
					resolveStatus, err = strconv.ParseUint(attr.Value, 10, 64)
					if err != nil {
						return "", fmt.Errorf("unable to parse resolve status")
					}
				case oracletypes.AttributeKeyReason:
					reason = attr.Value
				}
			}

			if reason != "" && rid == id && resolveStatus == uint64(oracletypes.RESOLVE_STATUS_FAILURE) {
				return reason, nil
			}
		}
	}

	return "", fmt.Errorf("no reason found")
}

func (c RPC) GetSignature(_ uint64) ([]byte, error) {
	// TODO: Implement when need TSS signature
	return []byte{}, nil
}

func (c RPC) GetBalance(_ sdk.AccAddress) (uint64, error) {
	// TODO: Implement
	return 0, nil
}

func (c RPC) SendRequest(msg *oracletypes.MsgRequestData, gasPrice float64, key keyring.Record) (*sdk.TxResponse, error) {
	// Get account to get nonce of sender first
	addr, err := key.GetAddress()
	if err != nil {
		return nil, err
	}

	acc, err := c.GetAccount(addr)
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
					"Fail to estimate tx gas",
					"Fail to estimate tx gas from %s with error %s",
					node.Remote(), err,
				)
				failCh <- struct{}{}
			} else {
				gasCh <- gas
			}
		}(node)
	}

	gas := uint64(0)

Gas:
	for range c.nodes {
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

	// Build unsigned tx with estimated gas
	txf = txf.WithGas(gas)
	txf = txf.WithGasPrices(fmt.Sprintf("%vuband", gasPrice))
	txb, err := txf.BuildUnsignedTx(msg)
	if err != nil {
		return nil, err
	}

	// Sign tx
	err = tx.Sign(txf, key.Name, txb, false)
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

	for range c.nodes {
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

func (c RPC) blockSearch(query string, page *int, perPage *int, orderBy string) (*ctypes.ResultBlockSearch, error) {
	resultCh := make(chan *ctypes.ResultBlockSearch)
	failCh := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			blockResult, err := node.BlockSearch(ctx, query, page, perPage, orderBy)
			if err != nil {
				c.logger.Warning(
					"Fail to do a blockSearch",
					"Fail to do a blockSearch from %s with error %s",
					node.Remote(), err,
				)
				failCh <- struct{}{}
			} else if blockResult.TotalCount == 0 {
				c.logger.Warning(
					"No result from a blockSearch",
					"No result from a blockSearch on %s",
					node.Remote(),
				)
				failCh <- struct{}{}
			} else {
				resultCh <- blockResult
			}
		}(node)
	}

	for range c.nodes {
		select {
		case res := <-resultCh:
			return res, nil
		case <-failCh:
		}
	}

	return nil, fmt.Errorf("failed to block search from all endpoints")
}
