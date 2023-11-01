package client

import (
	"fmt"

	"github.com/bandprotocol/band-sdk/utils"
	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
)

type Client interface {
	// Query functions
	GetAccount(account sdk.AccAddress) (client.Account, error)
	GetTx(txHash string) (*sdk.TxResponse, error)
	GetResult(id uint64) (*oracletypes.Result, error)
	GetSignature(id uint64) ([]byte, error)
	// TODO: Implement get balance for monitoring
	// GetBalance(account sdk.AccAddress) (uint64, error)

	// Build, sign, and broadcast transaction
	SendRequest(msg *oracletypes.MsgRequestData, key keyring.Info) (*sdk.TxResponse, error)
}

var (
	_ Client = &MultipleClient{}
)

// MultipleClient stores many clients to communicate with BandChain
type MultipleClient struct {
	ClientCtx client.Context
	TxFactory tx.Factory
	Nodes     []*rpchttp.HTTP
	Keyring   keyring.Keyring

	logger utils.Logger
}

func NewMultipleClient(
	logger utils.Logger,
	endpoints []string,
	chainId string,
	timeout string,
	gasPrice string,
	keyring keyring.Keyring,
) (*MultipleClient, error) {
	nodes := make([]*rpchttp.HTTP, 0, len(endpoints))
	for _, endpoint := range endpoints {
		node, err := newRPCClient(endpoint, timeout)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return &MultipleClient{
		ClientCtx: NewClientCtx(chainId),
		TxFactory: createTxFactory(chainId, gasPrice, keyring),
		Nodes:     nodes,
		Keyring:   keyring,
		logger:    logger,
	}, nil
}

// find max account sequence from multiple clients
func (c MultipleClient) GetAccount(account sdk.AccAddress) (client.Account, error) {
	var maxSeqAcc client.Account

	resultc := make(chan client.Account)
	failc := make(chan struct{})

	for _, node := range c.Nodes {
		go func(node *rpchttp.HTTP) {
			account, err := getAccount(c.ClientCtx.WithClient(node), account)
			if err != nil {
				failc <- struct{}{}
			} else {
				resultc <- account
			}
		}(node)
	}

	for i := 0; i < len(c.Nodes); i++ {
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

func (c MultipleClient) GetResult(id uint64) (*oracletypes.Result, error) {
	resultc := make(chan *oracletypes.Result, len(c.Nodes))
	failc := make(chan struct{}, len(c.Nodes))

	for _, node := range c.Nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Getting request", "Try to get request from %s", node.Remote())
			res, err := getRequest(c.ClientCtx.WithClient(node), id)
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

	for i := 0; i < len(c.Nodes); i++ {
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

func (c MultipleClient) GetTx(txHash string) (*sdk.TxResponse, error) {
	resultc := make(chan *sdk.TxResponse, len(c.Nodes))
	failc := make(chan struct{}, len(c.Nodes))

	for _, node := range c.Nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Getting tx", "Try to find tx(%s) from %s", txHash, node.Remote())
			res, err := getTx(c.ClientCtx.WithClient(node), txHash)
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

	for i := 0; i < len(c.Nodes); i++ {
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

func (c MultipleClient) GetSignature(id uint64) ([]byte, error) {
	// TODO: Implement when need TSS signature
	return []byte{}, nil
}

func (c MultipleClient) SendRequest(msg *oracletypes.MsgRequestData, key keyring.Info) (*sdk.TxResponse, error) {
	// Get account to get nonce of sender first
	acc, err := c.GetAccount(key.GetAddress())
	if err != nil {
		return nil, err
	}

	txf := c.TxFactory.WithAccountNumber(acc.GetAccountNumber()).WithSequence(acc.GetSequence())
	// Estimate gas
	gasc := make(chan uint64, len(c.Nodes))
	failc := make(chan struct{}, len(c.Nodes))

	for _, node := range c.Nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Estimating tx gas", "From %s", node.Remote())
			gas, err := estimateGas(c.ClientCtx.WithClient(node), txf, msg)
			if err != nil {
				c.logger.Warning(
					"Fail to estimate tx gas from single endpoint",
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
	for i := 0; i < len(c.Nodes); i++ {
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

	// Build unsigned tx with estimated gas
	txf = txf.WithGas(gas)
	txb, err := tx.BuildUnsignedTx(txf, msg)
	if err != nil {
		return nil, err
	}

	// Sign tx
	err = tx.Sign(txf, key.GetName(), txb, false)
	if err != nil {
		return nil, err
	}

	// Generate the transaction bytes
	txBytes, err := c.ClientCtx.TxConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return nil, err
	}

	resultc := make(chan *sdk.TxResponse, len(c.Nodes))
	errorc := make(chan struct{}, len(c.Nodes))

	for _, node := range c.Nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("Broadcasting tx", "Try to broadcast tx to %s", node.Remote())
			res, err := c.ClientCtx.WithClient(node).BroadcastTx(txBytes)
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

	for i := 0; i < len(c.Nodes); i++ {
		select {
		case res := <-resultc:
			// Return the first tx response that we found
			return res, nil
		case <-errorc:
		}
	}
	// If cannot broadcast to all nodes return error
	return nil, fmt.Errorf("failed to find transaction from all endpoints")
}
