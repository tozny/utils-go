package utils

import (
	"log"
	"os"
)

// ServiceLogger represents a logger with logging level prefixes for a specific service.
type ServiceLogger struct {
	ServiceName string
	Debug       *log.Logger
	Info        *log.Logger
	Fatal       *log.Logger
	Error       *log.Logger
}

// NewServiceLogger returns a logger with designated logging levels for a particular service.
func NewServiceLogger(serviceName string) ServiceLogger {
	logger := ServiceLogger{
		ServiceName: serviceName,
		Debug:       log.New(os.Stdout, "DEBUG: "+serviceName+":", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		Info:        log.New(os.Stdout, "INFO: "+serviceName+":", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		Fatal:       log.New(os.Stdout, "FATAL: "+serviceName+":", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		Error:       log.New(os.Stdout, "ERROR: "+serviceName+":", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
	}
	return logger
}
