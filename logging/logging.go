package logging

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

// Logger is an interface defining object that providers logging methods that are
// log level aware. This is especially useful when another interface supports
// logging with logging levels. This interface can be embedded.
type Logger interface {
	RawLogger() *log.Logger
	SetLevel(string)
	Print(...interface{})
	Printf(string, ...interface{})
	Printj(interface{}) error
	Println(...interface{})
	Debug(...interface{})
	Debugf(string, ...interface{})
	Debugj(interface{}) error
	Debugln(...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Infoj(interface{}) error
	Infoln(...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
	Errorj(interface{}) error
	Errorln(...interface{})
	Critical(...interface{})
	Criticalf(string, ...interface{})
	Criticalln(...interface{})
	Criticalj(interface{}) error
}

// LogWriter maps string values to io.Writer interfaces intended for logging output.
//
// This function is intended to provide a standard way of mapping environment-based
// configuration with various logging output writers. An empty string will default to
// standard out. stdout, stderr will send output to standard out and standard error
// respectively. /dev/null will discard the output. Any other string will provide
// a writer to a file at that location.
//
// When calling this function, it is a good idea to type assert for an io.Closer
// or similar and if assertion is successful properly close the log file on shutdown.
func LogWriter(writerString string) (io.Writer, error) {
	switch writerString {
	case "stdout", "":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "/dev/null":
		return ioutil.Discard, nil
	default:
		return os.OpenFile(writerString, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	}
}
