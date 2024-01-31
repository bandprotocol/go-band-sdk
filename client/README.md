# BandChain Go Client Module

This module is a part of the BandChain Go Client SDK, a Go library for interacting with the BandChain blockchain. It provides a set of tools to easily communicate with BandChain to query data and send transactions.

## Installation

To start using the BandChain Go Client SDK, install the library and its dependencies using the following command:

```bash
go get github.com/bandprotocol/go-band-sdk/client
```

## Features

The client module provides the following functionalities:

- Get account details
- Retrieve transaction details
- Get result of a request
- Get signature of a request
- Get block result
- Query request failure reason
- Get balance of an account
- Send a request

## Usage

Here is an example of how to use the client to retrieve data from BandChain:

```go
package main

import (
	"fmt"

	"github.com/bandprotocol/go-band-sdk/client"
	"github.com/bandprotocol/go-band-sdk/utils/logger"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

func main() {
	// Initialize a new keyring
	kr := keyring.NewInMemory()

	// Define the RPC endpoints
	endpoints := []string{"https://rpc.laozi-testnet6.bandchain.org:443"}

	// Create a new RPC client
	rpcClient, err := client.NewRPC(
		logger.NewLogrus("info"),
		endpoints,
		"band-laozi-testnet6",
		"30s",
		"0.0025uband",
		kr,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Use the RPC client to get account details
	account, err := rpcClient.GetAccount(kr.GetAddress())
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Account details:", account)
}
```

In this example, we initialize a new client and use it to query the failure reason of a request with ID 1234.

## Documentation

For more detailed information about the BandChain Go Client SDK and its modules, please refer to the official documentation.