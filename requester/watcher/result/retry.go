package result

import "github.com/bandprotocol/go-band-sdk/requester/types"

type RetryResult struct {
	types.RetryHandler
}

func (r *RetryResult) Run() {
	for {
		fail := <-r.FailedRequestc
		attempt := r.SeenRequest[fail.Id]
		if attempt < r.MaxTry {
			r.SeenRequest[fail.Id] = attempt + 1
			// TODO: Process retry logic
			r.Requestc <- fail.Request
		} else {
			r.Abortc <- fail.Id
		}
	}
}
