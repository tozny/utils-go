package utils

import (
	"io/ioutil"
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
	*log.Logger
}

// NewServiceLogger returns a logger with designated logging levels for a particular service.
func NewServiceLogger(serviceName string, debug bool) ServiceLogger {
	logger := ServiceLogger{
		serviceName,
		log.New(os.Stdout, "DEBUG: "+serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		log.New(os.Stdout, "INFO: "+serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		log.New(os.Stdout, "FATAL: "+serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		log.New(os.Stdout, "ERROR: "+serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		log.New(os.Stdout, serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
	}
	if debug == false {
		logger.Debug.SetOutput(ioutil.Discard)
	}
	return logger
}
