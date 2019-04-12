package utils

import (
	"io/ioutil"
	"log"
	"os"
)

const (
	OFF      = iota // 0
	SERVICE  = iota
	CRITICAL = iota
	ERROR    = iota
	INFO     = iota
	DEBUG    = iota // 5
)

// ServiceLogger represents a logger with logging level prefixes for a specific service.
type ServiceLogger struct {
	ServiceName string
	*log.Logger
	Critical *log.Logger
	Error    *log.Logger
	Info     *log.Logger
	Debug    *log.Logger
}

// NewServiceLogger returns a logger with designated logging levels for a particular service.
func NewServiceLogger(serviceName string, level int) ServiceLogger {
	logger := ServiceLogger{
		serviceName,
		log.New(os.Stdout, serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		log.New(os.Stdout, "CRITICAL: "+serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		log.New(os.Stdout, "ERROR: "+serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		log.New(os.Stdout, "INFO: "+serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		log.New(os.Stdout, "DEBUG: "+serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
	}
	if level < DEBUG {
		logger.Debug.SetOutput(ioutil.Discard)
	}
	if level < INFO {
		logger.Info.SetOutput(ioutil.Discard)
	}
	if level < ERROR {
		logger.Error.SetOutput(ioutil.Discard)
	}
	if level < CRITICAL {
		logger.Critical.SetOutput(ioutil.Discard)
	}
	if level < SERVICE {
		logger.SetOutput(ioutil.Discard)
	}
	return logger
}
