package client

import (
	"fmt"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"

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

	resultc := make(chan client.Account)
	failc := make(chan struct{})

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			account, err := getAccount(c.ctx.WithClient(node), account)
			if err != nil {
				failc <- struct{}{}
			} else {
				resultc <- account
			}
		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case acc := <-resultc:
			if maxSeqAcc == nil || acc.GetAccountNumber() > maxSeqAcc.GetAccountNumber() {
				maxSeqAcc = acc
			}
		case <-failc:
		}

	}

	if maxSeqAcc == nil {
		return nil, fmt.Errorf("failed to get account from all endpoints")
	}

	return maxSeqAcc, nil
}

func (c RPC) GetResult(id uint64) (*oracletypes.Result, error) {
	resultc := make(chan *oracletypes.Result, len(c.nodes))
	failc := make(chan struct{}, len(c.nodes))

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
				failc <- struct{}{}
				return
			}
			if res.Result != nil {
				resultc <- res.Result
			} else {
				c.logger.Debug(
					"Request has not been resolved yet",
					"Get result from %s, but request #%d has not been resolved yet",
					node.Remote(), id,
				)
				failc <- struct{}{}
			}

		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case res := <-resultc:
			// Return the first result that we found
			return res, nil
		case <-failc:
		}
	}
	// If cannot get result from all nodes return error
	return nil, fmt.Errorf("failed to get result from all endpoints")
}

func (c RPC) GetTx(txHash string) (*sdk.TxResponse, error) {
	resultc := make(chan *sdk.TxResponse, len(c.nodes))
	failc := make(chan struct{}, len(c.nodes))

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
				failc <- struct{}{}
				return
			}
			resultc <- res
		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case res := <-resultc:
			// Return the first result that we found
			return res, nil
		case <-failc:
		}
	}
	// If cannot get transaction from all nodes return error
	return nil, fmt.Errorf("failed to find transaction from all endpoints")
}

func (c RPC) GetSignature(id uint64) ([]byte, error) {
	// TODO: Implement when need TSS signature
	return []byte{}, nil
}

func (c RPC) SendRequest(msg *oracletypes.MsgRequestData, key keyring.Info) (*sdk.TxResponse, error) {
	// Get account to get nonce of sender first
	acc, err := c.GetAccount(key.GetAddress())
	if err != nil {
		return nil, err
	}

	txf := c.txFactory.WithAccountNumber(acc.GetAccountNumber()).WithSequence(acc.GetSequence())
	// Estimate gas
	gasc := make(chan uint64, len(c.nodes))
	failc := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Estimating txutil gas", "From %s", node.Remote())
			gas, err := estimateGas(c.ctx.WithClient(node), txf, msg)
			if err != nil {
				c.logger.Warning(
					"Fail to estimate txutil gas from single endpoint",
					"From %s",
					node.Remote(),
				)
				failc <- struct{}{}
			} else {
				gasc <- gas
			}
		}(node)
	}

	gas := uint64(0)

Gas:
	for i := 0; i < len(c.nodes); i++ {
		select {
		case gas = <-gasc:
			// Return the first result that we found
			break Gas
		case <-failc:
		}
	}

	// If all endpoint fail to estimate gas return error
	// TODO: Find a way to extract error to meaningful response that able to handle by caller
	if gas == 0 {
		return nil, fmt.Errorf("fail to estimate gas")
	}

	// Build unsigned txutil with estimated gas
	txf = txf.WithGas(gas)
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

	resultc := make(chan *sdk.TxResponse, len(c.nodes))
	errorc := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Broadcasting txutil", "Try to broadcast txutil to %s", node.Remote())
			res, err := c.ctx.WithClient(node).BroadcastTx(txBytes)
			if err != nil {
				c.logger.Warning(
					"Fail to broadcast to single endpoint",
					"to %s, with error %s",
					node.Remote(), err,
				)
				errorc <- struct{}{}
				return
			}
			resultc <- res
		}(node)
	}

	for i := 0; i < len(c.nodes); i++ {
		select {
		case res := <-resultc:
			// Return the first txutil response that we found
			return res, nil
		case <-errorc:
		}
	}
	// If the txutil cannot broadcast to all nodes, return an error
	return nil, fmt.Errorf("failed to find transaction from all endpoints")
}
