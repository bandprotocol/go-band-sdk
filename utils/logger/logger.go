package logger

type Logger interface {
	Debug(topic string, format string, args ...interface{})
	Info(topic string, format string, args ...interface{})
	Warning(topic string, format string, args ...interface{})
	Error(topic string, format string, args ...interface{})
	Critical(topic string, format string, args ...interface{})
}
