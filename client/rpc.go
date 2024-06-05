package client

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	bandtsstypes "github.com/bandprotocol/chain/v2/x/bandtss/types"
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
	ctx               client.Context
	txFactory         tx.Factory
	nodes             []*rpchttp.HTTP
	keyring           keyring.Keyring
	subscriptionInfos *sync.Map
	logger            logging.Logger
}

// NewRPC creates new RPC client
func NewRPC(
	logger logging.Logger,
	endpoints []string,
	chainID string,
	timeout time.Duration,
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
		ctx:               NewClientCtx(chainID),
		txFactory:         createTxFactory(chainID, gasPrice, keyring),
		nodes:             nodes,
		keyring:           keyring,
		subscriptionInfos: &sync.Map{},
		logger:            logger,
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
func (c RPC) GetResult(id uint64) (*OracleResult, error) {
	resultCh := make(chan *OracleResult, len(c.nodes))
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

			signingID := bandtsstypes.SigningID(0)
			if res.Signing != nil {
				signingID = res.Signing.SigningID
			}
			oracleResult := OracleResult{
				Result:    res.Result,
				SigningID: signingID,
			}

			resultCh <- &oracleResult
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
					"Fail to get transaction from %s with error %s, it might be waiting for included in the block",
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

func (c RPC) GetSignature(signingID uint64) (*SigningResult, error) {
	resultCh := make(chan *SigningResult, len(c.nodes))
	failCh := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			c.logger.Debug("GetSignature", "Attempting to get signature from %s", node.Remote())

			res, err := getSigning(c.ctx.WithClient(node), signingID)
			if err != nil {
				c.logger.Warning(
					"GetSignature",
					"Failed to get signature from %s with error %s",
					node.Remote(), err,
				)
				failCh <- struct{}{}
				return
			} else if res.CurrentGroupSigningResult == nil {
				c.logger.Warning(
					"GetSignature",
					"Failed to get signature from %s, signing ID: %d, no signing result from current group",
					node.Remote(), signingID,
				)
				failCh <- struct{}{}
				return
			}

			signingResult := SigningResult{
				CurrentGroup:   convertSigningResultToSigningInfo(res.CurrentGroupSigningResult),
				ReplacingGroup: convertSigningResultToSigningInfo(res.ReplacingGroupSigningResult),
			}
			resultCh <- &signingResult
		}(node)
	}

	// check if the signature is ready to be used. If every node returns waiting status, return one.
	// If every node fails to query a result return an error.
	var res *SigningResult
	for range c.nodes {
		select {
		case res = <-resultCh:
			if res.IsReady() {
				return res, nil
			}
		case <-failCh:
		}
	}

	if res == nil {
		return nil, fmt.Errorf("failed to get result from all endpoints")
	}

	return res, nil
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
	failCh := make(chan error, len(c.nodes))

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
				failCh <- err
			} else {
				gasCh <- gas
			}
		}(node)
	}

	gas := uint64(0)

	var gasErr error
Gas:
	for range c.nodes {
		select {
		case gas = <-gasCh:
			// Return the first result that we found
			gasErr = nil
			break Gas
		case err := <-failCh:
			// Check if all endpoints return the same error.
			if gasErr == nil {
				gasErr = err
			} else if !errors.Is(gasErr, err) {
				gasErr = fmt.Errorf("fail to estimate gas")
			}
		}
	}

	// If all endpoint fail to estimate gas return error
	if gasErr != nil {
		return nil, gasErr
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

// GetRequestProofByID fetches and returns the EVM proof for the given request ID from the REST endpoint.
func (c RPC) GetRequestProofByID(reqID uint64) ([]byte, error) {
	resultCh := make(chan []byte, len(c.nodes))
	failCh := make(chan struct{}, len(c.nodes))

	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			evmProofBytes, err := getRequestProof(c.ctx.WithClient(node), reqID)
			if err != nil {
				c.logger.Warning("GetRequestProofByID", "can't get proof bytes from %s; %s", node.Remote(), err)
				failCh <- struct{}{}
				return
			}

			resultCh <- evmProofBytes
		}(node)
	}

	for range c.nodes {
		select {
		case res := <-resultCh:
			// Return the first tx response that we found
			return res, nil
		case <-failCh:
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

// Subscribe subscribes to the given event name and query from multiple clients. If it cannot
// subscribe to a node within the given timeout, it will drop that node from the result.
// If no nodes can be subscribed, it will return an error.
func (c RPC) Subscribe(name, query string) (*SubscriptionInfo, error) {
	chInfos := make([]ChannelInfo, 0, len(c.nodes))
	queue := make(chan ChannelInfo, 1)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// subscribe to all nodes within a given timeout
	wg.Add(len(c.nodes))
	for _, node := range c.nodes {
		go func(node *rpchttp.HTTP) {
			if !node.IsRunning() {
				c.logger.Debug("Subscribe", "start the node %s", node.Remote())
				if err := node.Start(); err != nil {
					c.logger.Warning("Subscribe", "Failed to start %s with error %s", node.Remote(), err)
					wg.Done()
					return
				}
			}

			eventCh, err := node.Subscribe(ctx, name, query, 1000)
			if err != nil {
				c.logger.Warning("Subscribe", "Failed to subscribe to %s with error %s", node.Remote(), err)
				wg.Done()
				return
			}

			queue <- ChannelInfo{
				RemoteAddr: node.Remote(),
				EventCh:    eventCh,
			}
		}(node)
	}

	// add result into the list.
	go func() {
		for info := range queue {
			chInfos = append(chInfos, info)
			wg.Done()
		}
	}()

	wg.Wait()
	if len(chInfos) == 0 {
		return nil, fmt.Errorf("failed to subscribe to all endpoints")
	}

	subInfo := SubscriptionInfo{
		Name:         name,
		Query:        query,
		ChannelInfos: chInfos,
	}

	c.subscriptionInfos.Store(name, subInfo)
	return &subInfo, nil
}

// Unsubscribe unsubscribes from the given event name from multiple clients.
func (c RPC) Unsubscribe(name string) error {
	v, found := c.subscriptionInfos.Load(name)
	if !found {
		return fmt.Errorf("event is not subscribed; name %s", name)
	}

	info, ok := v.(SubscriptionInfo)
	if !ok {
		return fmt.Errorf("failed to cast to SubscriptionInfo; name %s", name)
	}

	remoteAddrs := make(map[string]struct{})
	for _, chInfo := range info.ChannelInfos {
		remoteAddrs[chInfo.RemoteAddr] = struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(info.ChannelInfos))
	for _, node := range c.nodes {
		if _, found := remoteAddrs[node.Remote()]; !found {
			continue
		}

		go func(node *rpchttp.HTTP, name, query string) {
			defer wg.Done()
			// if failed to unsubscribe, it means that the node object is not running or timeout.
			// Log and continue the process.
			if err := node.Unsubscribe(ctx, name, query); err != nil {
				c.logger.Warning("Unsubscribe", "Failed to unsubscribe from %s with error %s", node.Remote(), err)
			}
		}(node, name, info.Query)
	}

	wg.Wait()
	c.subscriptionInfos.Delete(name)
	return nil
}
