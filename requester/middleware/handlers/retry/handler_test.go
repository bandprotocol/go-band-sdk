package retry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.uber.org/mock/gomock"

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
	ctrl := gomock.NewController(t)
	factory := setupFactory(3)
	handler := retry.NewCounterHandler[types.Task, types.Task](factory)

	mockTask := types.NewMockTask(ctrl)
	mockTask := sender.NewTask(uint64(1), nil)
	mockTask.EXPECT().ID().Return(uint64(1)).AnyTimes()

	_, err := handler.Handle(
		mockTask, func(ctx types.Task) (types.Task, error) {
			return ctx, nil
		},
	)

	assert.NoErrorf(t, err, "expected no error, got %v", err)
}

func TestCounterHandlerWithMaxRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := setupFactory(1, ctrl)
	handler := retry.NewCounterHandler[types.Task, types.Task](factory)

	mockTask := types.NewMockTask(ctrl)
	mockTask.EXPECT().ID().Return(uint64(1)).AnyTimes()

	parser := func(ctx types.Task) (types.Task, error) {
		return ctx, nil
	}

	handler.Handle(mockTask, parser)
	_, err := handler.Handle(mockTask, parser)

	assert.Errorf(t, err, "expected error, got %v", err)
}

func TestResolverHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := setupFactory(1, ctrl)
	counter := retry.NewCounterHandler[types.Task, types.Task](factory)
	resolver := retry.NewResolverHandler[types.Task, types.Task](factory)

	mockTask := types.
		mockTask.EXPECT().ID().Return(uint64(1)).AnyTimes()

	parser := func(ctx types.Task) (types.Task, error) {
		return ctx, nil
	}

	counter.Handle(mockTask, parser)
	resolver.Handle(mockTask, parser)
	_, err := counter.Handle(mockTask, parser)

	assert.NoErrorf(t, err, "expected no error, got %v", err)
}
