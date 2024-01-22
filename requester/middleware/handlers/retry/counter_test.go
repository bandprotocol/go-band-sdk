package retry_test

import (
	"testing"

	"github.com/bandprotocol/go-band-sdk/requester/middleware/handlers/retry"
)

func TestCounterInc(t *testing.T) {
	counter := retry.NewCounter()

	counter.Inc(1)

	if val, ok := counter.Peek(1); !ok || val != 1 {
		t.Errorf("expected 1, got %v", val)
	}

	counter.Inc(1)
	counter.Inc(1)
	if val, _ := counter.Peek(1); val != 3 {
		t.Errorf("expected 3, got %v", val)
	}
}

func TestCounterClear(t *testing.T) {
	counter := retry.NewCounter()

	counter.Inc(1)
	counter.Clear(1)
	if _, ok := counter.Peek(1); ok {
		t.Errorf("expected false, got %v", ok)
	}
}

func TestCounterPeek(t *testing.T) {
	counter := retry.NewCounter()

	if _, ok := counter.Peek(1); ok {
		t.Errorf("expected false, got %v", ok)
	}

	counter.Inc(1)
	if val, ok := counter.Peek(1); !ok || val != 1 {
		t.Errorf("expected 1, got %v", val)
	}
}
