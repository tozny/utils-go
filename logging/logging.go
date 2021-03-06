package logging

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

// Logger is an interface defining object that providers logging methods that are
// log level aware. This is especially useful when another interface supports
// logging with logging levels. This interface can be embedded.
type Logger interface {
	SetLevel(string)
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
	Debug(...interface{})
	Debugf(string, ...interface{})
	Debugln(...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Infoln(...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
	Errorln(...interface{})
	Critical(...interface{})
	Criticalf(string, ...interface{})
	Criticalln(...interface{})
}

type StructuredLogger interface {
	Logger
	Warn(...interface{})
	Warnf(string, ...interface{})
	Warnln(...interface{})
	Debugw(message string, r *http.Request, v ...interface{})
	Infow(message string, r *http.Request, v ...interface{})
	Warnw(message string, r *http.Request, v ...interface{})
	Errorw(message string, r *http.Request, v ...interface{})
	CriticalW(message string, r *http.Request, v ...interface{})
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
