package mock

import (
	"github.com/bandprotocol/go-band-sdk/utils/logging"
)

var _ logging.Logger = &Logger{}

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Debug(topic string, format string, args ...interface{}) {
}

func (l *Logger) Info(topic string, format string, args ...interface{}) {
}

func (l *Logger) Warning(topic string, format string, args ...interface{}) {
}

func (l *Logger) Error(topic string, format string, args ...interface{}) {
}

func (l *Logger) Critical(topic string, format string, args ...interface{}) {
}
