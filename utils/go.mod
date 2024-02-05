module github.com/bandprotocol/go-band-sdk/utils

go 1.21

toolchain go1.21.6

require (
	github.com/kyokomi/emoji v2.2.4+incompatible
	github.com/sirupsen/logrus v1.9.3
)

require (
	github.com/stretchr/testify v1.8.4 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	// use cometbft
	github.com/tendermint/tendermint => github.com/cometbft/cometbft v0.34.29
)
