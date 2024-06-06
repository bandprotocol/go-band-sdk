package retry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	oracletypes "github.com/bandprotocol/chain/v2/x/oracle/types"

	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/retry"
	"github.com/bandprotocol/go-band-sdk/requester/sender"
	"github.com/bandprotocol/go-band-sdk/requester/types"
	"github.com/bandprotocol/go-band-sdk/utils/logging/mock"
)

func setupFactory(maxTry uint64) *retry.HandlerFactory {
	mockLogger := mock.NewLogger()
	return retry.NewHandlerFactory(maxTry, mockLogger)
}

func TestCounterHandler(t *testing.T) {
	factory := setupFactory(3)
	handler := retry.NewCounterHandler[types.Task, types.Task](factory)

	mockTask := sender.NewTask(uint64(1), &oracletypes.MsgRequestData{})

	_, err := handler.Handle(
		mockTask, func(ctx types.Task) (types.Task, error) {
			return ctx, nil
		},
	)

	assert.NoErrorf(t, err, "expected no error, got %v", err)
}

func TestCounterHandlerWithMaxRetry(t *testing.T) {
	factory := setupFactory(1)
	handler := retry.NewCounterHandler[types.Task, types.Task](factory)

	mockTask := sender.NewTask(uint64(1), &oracletypes.MsgRequestData{})

	parser := func(ctx types.Task) (types.Task, error) {
		return ctx, nil
	}

	_, err := handler.Handle(mockTask, parser)
	assert.NoErrorf(t, err, "expected no error, got %v", err)
	// Should fail as max retry exceeded
	_, err = handler.Handle(mockTask, parser)
	assert.Errorf(t, err, "expected error, got %v", err)
}

func TestResolverHandler(t *testing.T) {
	factory := setupFactory(1)
	counter := retry.NewCounterHandler[types.Task, types.Task](factory)
	resolver := retry.NewResolverHandler[types.Task, types.Task](factory)

	mockTask := sender.NewTask(uint64(1), &oracletypes.MsgRequestData{})

	parser := func(ctx types.Task) (types.Task, error) {
		return ctx, nil
	}

	_, err := counter.Handle(mockTask, parser)
	assert.NoErrorf(t, err, "expected no error, got %v", err)
	// Resolver should clear current counter for the task
	_, err = resolver.Handle(mockTask, parser)
	assert.NoErrorf(t, err, "expected no error, got %v", err)

	// Should pass as counter is cleared
	_, err = counter.Handle(mockTask, parser)
	assert.NoErrorf(t, err, "expected no error, got %v", err)
}
