package log

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	logger          *logrus.Logger
	loggerOnce      sync.Once
	levelPrintNames = map[logrus.Level]string{
		logrus.PanicLevel: "Panic",
		logrus.FatalLevel: "Fatal",
		logrus.ErrorLevel: "Error",
		logrus.WarnLevel:  "Warn",
		logrus.InfoLevel:  "Info",
		logrus.DebugLevel: "Debug",
		logrus.TraceLevel: "Trace",
	}
	levelFlagNames = map[string]logrus.Level{
		"panic": logrus.PanicLevel,
		"fatal": logrus.FatalLevel,
		"error": logrus.ErrorLevel,
		"warn":  logrus.WarnLevel,
		"info":  logrus.InfoLevel,
		"debug": logrus.DebugLevel,
		"trace": logrus.TraceLevel,
	}
)

// customFormatter implements logrus.Formatter to print errors in the format: Error: message
// and other levels as usual.
type customFormatter struct{}

func (f *customFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	level, ok := levelPrintNames[entry.Level]
	if !ok {
		level = entry.Level.String()
	}
	return []byte(level + ": " + entry.Message + "\n"), nil
}

// GetLogger returns a singleton logrus.Logger instance
func GetLogger() *logrus.Logger {
	loggerOnce.Do(func() {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
		logger.SetFormatter(&customFormatter{})
	})
	return logger
}

// SetLevel sets the log level for the logger
func SetLevel(level logrus.Level) {
	GetLogger().SetLevel(level)
}

// InitLogger sets the log level based on the verbosity string and exits on error
func InitLogger(verbosity string) {
	level, ok := levelFlagNames[verbosity]
	if !ok {
		GetLogger().Error("Unknown verbosity level: " + verbosity)
		os.Exit(1)
	}
	SetLevel(level)
}
