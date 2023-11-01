package sender

import "github.com/bandprotocol/band-sdk/requester/types"

type RetrySender struct {
	types.RetryHandler
}

func (r *RetrySender) Run() {
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
