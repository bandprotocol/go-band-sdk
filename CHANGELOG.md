# CHANGELOG

## Unreleased

- (bump) use cosmos-sdk package v0.47.11
- (bump) replace github.com/tendermint/tendermint by github.com/cometbft/cometbft v0.37.5
- (example) update example folder
- (requester) remove channel size for initializing middleware, watcher and sender objects
- (requester) expose error channel on middleware object
- (requester) add logic for converting errors when receiving oracle request result on request watcher
- (requester) rearrange folder structure
- (requester) add a new type of retryHandler (retryHandler with condition)
- (client) add GetSigning and GetRequestProofByID

## [v1.0.1]

- Move example folder

## [v1.0.0]

- Implemented the first version of the client package
- Implemented the first version of the requester package
