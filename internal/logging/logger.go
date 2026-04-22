package logging

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	configureOnce sync.Once
	baseLogger    *logrus.Logger
)

type Logger struct {
	entry *logrus.Entry
}

func ConfigureFromEnv() {
	logger := defaultLogger()
	if IsDebugMode() {
		logger.SetLevel(logrus.DebugLevel)
		return
	}
	logger.SetLevel(logrus.InfoLevel)
}

func IsDebugMode() bool {
	return parseBoolEnv(os.Getenv("DEBUG_MODE"))
}

func With(component string) *Logger {
	return &Logger{entry: defaultLogger().WithField("component", component)}
}

func (l *Logger) Debug(message string, keyvals ...any) {
	l.log(logrus.DebugLevel, message, keyvals...)
}

func (l *Logger) Info(message string, keyvals ...any) {
	l.log(logrus.InfoLevel, message, keyvals...)
}

func (l *Logger) Error(message string, keyvals ...any) {
	l.log(logrus.ErrorLevel, message, keyvals...)
}

func (l *Logger) log(level logrus.Level, message string, keyvals ...any) {
	entry := l.entry
	if entry == nil {
		entry = defaultLogger().WithField("component", "app")
	}
	entry.WithFields(fieldsFromKeyvals(keyvals...)).Log(level, message)
}

func defaultLogger() *logrus.Logger {
	configureOnce.Do(func() {
		logger := logrus.New()
		logger.SetOutput(os.Stderr)
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
		logger.SetLevel(logrus.InfoLevel)
		baseLogger = logger
	})
	return baseLogger
}

func fieldsFromKeyvals(keyvals ...any) logrus.Fields {
	fields := make(logrus.Fields, (len(keyvals)+1)/2)
	for index := 0; index < len(keyvals); index += 2 {
		key := fmt.Sprintf("field_%d", index)
		if named, ok := keyvals[index].(string); ok && strings.TrimSpace(named) != "" {
			key = named
		}

		value := any(nil)
		if index+1 < len(keyvals) {
			value = keyvals[index+1]
		}
		fields[key] = value
	}
	return fields
}

func parseBoolEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "debug":
		return true
	default:
		return false
	}
}
