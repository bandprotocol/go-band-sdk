package subscriber

import (
	"time"
)

type Config struct {
	ExpirationDuration time.Duration
	CleanupInterval    time.Duration
}

func DefaultConfig() Config {
	return Config{
		ExpirationDuration: 5 * time.Minute,
		CleanupInterval:    10 * time.Minute,
	}
}
