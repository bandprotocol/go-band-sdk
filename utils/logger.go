package utils

import (
	"os"

	"github.com/kyokomi/emoji"
	"github.com/sirupsen/logrus"
)

type Logger interface {
	Debug(topic string, format string, args ...interface{})
	Info(topic string, format string, args ...interface{})
	Warning(topic string, format string, args ...interface{})
	Error(topic string, format string, args ...interface{})
	Critical(topic string, format string, args ...interface{})
}

var (
	_ Logger = &LogrusLogger{}
)

type LogrusLogger struct {
	logger *logrus.Logger
}

func NewLogrusLogger(level string) *LogrusLogger {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true

	log := logrus.New()
	log.Out = os.Stdout
	log.SetFormatter(customFormatter)

	lv, err := logrus.ParseLevel(level)
	if err == nil {
		log.SetLevel(lv)
	}

	return &LogrusLogger{logger: log}
}

func (l *LogrusLogger) Debug(topic string, format string, args ...interface{}) {
	l.logger.Debug(topic + ": " + emoji.Sprintf(format, args...))
}

func (l *LogrusLogger) Info(topic string, format string, args ...interface{}) {
	l.logger.Info(topic + ": " + emoji.Sprintf(format, args...))
}

func (l *LogrusLogger) Warning(topic string, format string, args ...interface{}) {
	l.logger.Warning(topic + ": " + emoji.Sprintf(format, args...))
}

func (l *LogrusLogger) Error(topic string, format string, args ...interface{}) {
	l.logger.Error(topic + ": " + emoji.Sprintf(format, args...))
}

func (l *LogrusLogger) Critical(topic string, format string, args ...interface{}) {
	l.logger.Error(topic + ": " + emoji.Sprintf(format, args...))
}
