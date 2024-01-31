package logging

import (
	"os"

	"github.com/kyokomi/emoji"
	"github.com/sirupsen/logrus"
)

var (
	_ Logger = &Logrus{}
)

type Logrus struct {
	logger *logrus.Logger
}

func NewLogrus(level string) *Logrus {
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

	return &Logrus{logger: log}
}

func (l *Logrus) Debug(topic string, format string, args ...interface{}) {
	l.logger.Debug(topic + ": " + emoji.Sprintf(format, args...))
}

func (l *Logrus) Info(topic string, format string, args ...interface{}) {
	l.logger.Info(topic + ": " + emoji.Sprintf(format, args...))
}

func (l *Logrus) Warning(topic string, format string, args ...interface{}) {
	l.logger.Warning(topic + ": " + emoji.Sprintf(format, args...))
}

func (l *Logrus) Error(topic string, format string, args ...interface{}) {
	l.logger.Error(topic + ": " + emoji.Sprintf(format, args...))
}

func (l *Logrus) Critical(topic string, format string, args ...interface{}) {
	l.logger.Error(topic + ": " + emoji.Sprintf(format, args...))
}
