package utils

import (
	"io"
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

// Debugf is equivalent to Printf with "DEBUG: SERVICENAME: " prepended.
func (sl *ServiceLogger) Debugf(format string, v ...interface{}) {
	sl.Debug.Printf(format, v...)
}

// Infof is equivalent to Printf with "INFO: SERVICENAME: " prepended.
func (sl *ServiceLogger) Infof(format string, v ...interface{}) {
	sl.Info.Printf(format, v...)
}

// Errorf is equivalent to Printf with "ERROR: SERVICENAME: " prepended.
func (sl *ServiceLogger) Errorf(format string, v ...interface{}) {
	sl.Error.Printf(format, v...)
}

// Criticalf is equivalent to Printf with "CRITICAL: SERVICENAME: " prepended.
func (sl *ServiceLogger) Criticalf(format string, v ...interface{}) {
	sl.Critical.Printf(format, v...)
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

// GetLogger initializes a logging object writing to the requested log file.
func GetLogger(logFile string, serviceName string) *log.Logger {
	var output io.Writer
	switch logFile {
	case "stdout":
		output = os.Stdout
	case "/dev/null":
		output = ioutil.Discard
	default:
		// TODO: make file case work, writing output to the specified location...
		output = os.Stdout
	}
	return log.New(output, serviceName+": ", log.LstdFlags|log.Lshortfile|log.Lmicroseconds)
}
