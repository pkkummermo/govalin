package logging

import (
	"log"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func GetLogger() *zap.SugaredLogger {
	if logger == nil {
		config := zap.NewDevelopmentConfig()
		config.DisableStacktrace = true
		config.Level.SetLevel(zap.InfoLevel)
		prodLogger, _ := config.Build()
		defer func() {
			err := logger.Sync()
			if err != nil {
				log.Printf("Error when trying to sync logger. %v", err)
			}
		}() // flushes buffer, if any
		logger = prodLogger.Sugar()
	}

	return logger
}
