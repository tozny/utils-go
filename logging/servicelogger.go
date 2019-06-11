package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// ServiceLogger represents a logger with logging level prefixes for a specific service.
type ServiceLogger struct {
	logLevel    int
	serviceName string
	*log.Logger
}

// ServiceLog adds additional level and service name to a provided log struct.
type ServiceLog struct {
	Level   string      `json:"level"`
	Service string      `json:"service"`
	Log     interface{} `json:"log"`
}

var levelMap = map[string]int{
	"OFF":      0,
	"SERVICE":  1,
	"CRITICAL": 2,
	"ERROR":    3,
	"INFO":     4,
	"DEBUG":    5,
}

// NewServiceLogger returns a logger with designated logging levels for a particular service.
func NewServiceLogger(out io.Writer, serviceName string, level string) ServiceLogger {
	logger := ServiceLogger{
		serviceName: serviceName,
		Logger:      log.New(out, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
	}
	logger.SetLevel(level)
	return logger
}

// Raw returns the underlying log.Logger for use in methods that do not support a full
// logging.Logger with log levels.
func (sl *ServiceLogger) Raw() *log.Logger {
	return sl.Logger
}

// SetLevel allows the log level of the ServiceLogger to be updated based on
// supported log level strings. These include in order:
// 	- "OFF"
// 	- "SERVICE"
// 	- "CRITICAL"
// 	- "ERROR"
// 	- "INFO"
// 	- "DEBUG"
//
// As the level goes from OFF to DEBUG, more detailed logs will output.
func (sl *ServiceLogger) SetLevel(level string) {
	levelInt, exists := levelMap[level]
	if !exists {
		sl.Printf("Unknown logging level %s. Using ERROR instead.", level)
		levelInt = levelMap["ERROR"]
	}
	sl.logLevel = levelInt
}

// Debug is equivalent to Println with "DEBUG: SERVICENAME: " prepended. Only output
// when log level is DEBUG.
func (sl *ServiceLogger) Debug(v ...interface{}) {
	sl.doPrint("DEBUG", v...)
}

// Debugln is equivalent to Println with "DEBUG: SERVICENAME: " prepended. Only output
// when log level is DEBUG.
func (sl *ServiceLogger) Debugln(v ...interface{}) {
	sl.doPrintln("DEBUG", v...)
}

// Debugf is equivalent to Printf with "DEBUG: SERVICENAME: " prepended. Only output
// when log level is DEBUG.
func (sl *ServiceLogger) Debugf(format string, v ...interface{}) {
	sl.doPrintf("DEBUG", format, v...)
}

// Debugj is equivalent to Printj with "DEBUG" as the log level. Only output
// when log level is CRITICAL or higher.
func (sl *ServiceLogger) Debugj(v interface{}) error {
	return sl.doPrintj("DEBUG", v)
}

// Info is equivalent to Println with "INFO: SERVICENAME: " prepended. Only output
// when log level is INFO or higher.
func (sl *ServiceLogger) Info(v ...interface{}) {
	sl.doPrint("INFO", v...)
}

// Infoln is equivalent to Println with "INFO: SERVICENAME: " prepended. Only output
// when log level is INFO or higher.
func (sl *ServiceLogger) Infoln(v ...interface{}) {
	sl.doPrintln("INFO", v...)
}

// Infof is equivalent to Printf with "INFO: SERVICENAME: " prepended. Only output
// when log level is INFO or higher.
func (sl *ServiceLogger) Infof(format string, v ...interface{}) {
	sl.doPrintf("INFO", format, v...)
}

// Infoj is equivalent to Printj with "INFO" as the log level. Only output
// when log level is CRITICAL or higher.
func (sl *ServiceLogger) Infoj(v interface{}) error {
	return sl.doPrintj("INFO", v)
}

// Error is equivalent to Print with "ERROR: SERVICENAME: " prepended. Only output
// when log level is ERROR.
func (sl *ServiceLogger) Error(v ...interface{}) {
	sl.doPrint("ERROR", v...)
}

// Errorln is equivalent to Println with "ERROR: SERVICENAME: " prepended. Only output
// when log level is ERROR or higher.
func (sl *ServiceLogger) Errorln(v ...interface{}) {
	sl.doPrintln("ERROR", v...)
}

// Errorf is equivalent to Printf with "ERROR: SERVICENAME: " prepended. Only output
// when log level is ERROR or higher.
func (sl *ServiceLogger) Errorf(format string, v ...interface{}) {
	sl.doPrintf("ERROR", format, v...)
}

// Errorj is equivalent to Printj with "ERROR" as the log level. Only output
// when log level is CRITICAL or higher.
func (sl *ServiceLogger) Errorj(v interface{}) error {
	return sl.doPrintj("ERROR", v)
}

// Critical is equivalent to Print with "CRITICAL: SERVICENAME: " prepended. Only output
// when log level is CRITICAL.
func (sl *ServiceLogger) Critical(v ...interface{}) {
	sl.doPrint("CRITICAL", v...)
}

// Criticalln is equivalent to Println with "CRITICAL: SERVICENAME: " prepended. Only output
// when log level is CRITICAL or higher.
func (sl *ServiceLogger) Criticalln(v ...interface{}) {
	sl.doPrintln("CRITICAL", v...)
}

// Criticalf is equivalent to Printf with "CRITICAL: SERVICENAME: " prepended. Only output
// when log level is CRITICAL or higher.
func (sl *ServiceLogger) Criticalf(format string, v ...interface{}) {
	sl.doPrintf("CRITICAL", format, v...)
}

// Criticalj is equivalent to Printj with "CRITICAL" as the log level. Only output
// when log level is CRITICAL or higher.
func (sl *ServiceLogger) Criticalj(v interface{}) error {
	return sl.doPrintj("CRITICAL", v)
}

// Print with "SERVICENAME: " prepended. Only output when log level is SERVICE or higher.
func (sl *ServiceLogger) Print(v ...interface{}) {
	sl.doPrint("SERVICE", v...)
}

// Println with "SERVICENAME: " prepended. Only output when log level is SERVICE or higher.
func (sl *ServiceLogger) Println(v ...interface{}) {
	sl.doPrintln("SERVICE", v...)
}

// Printf with "SERVICENAME: " prepended. Only output when log level is SERVICE or higher.
func (sl *ServiceLogger) Printf(format string, v ...interface{}) {
	sl.doPrintf("SERVICE", format, v...)
}

// Printj adds service name and level to v and prints in json format.
func (sl *ServiceLogger) Printj(v interface{}) error {
	return sl.doPrintj("SERVICE", v)
}

// prefixString returns "LEVEL: SERVICENAME: " unless the level is at or below
// SERVICE in which case the prefix is just "SERVICENAME: "
func (sl *ServiceLogger) prefixString(level string) string {
	if levelMap[level] <= levelMap["SERVICE"] {
		return sl.serviceName + ": "
	}
	return level + ": " + sl.serviceName + ": "
}

// doPrint conditionally prints a message prefixed with the log level and service name.
// If log level is below level, the message is not output.
func (sl *ServiceLogger) doPrint(level string, v ...interface{}) {
	if sl.logLevel >= levelMap[level] {
		sl.Output(2, sl.prefixString(level)+fmt.Sprint(v...))
	}
}

// doPrint conditionally prints a printf formatted message prefixed with the log
// level and service name. If log level is below level, the message is not output.
func (sl *ServiceLogger) doPrintf(level string, format string, v ...interface{}) {
	if sl.logLevel >= levelMap[level] {
		sl.Output(2, sl.prefixString(level)+fmt.Sprintf(format, v...))
	}
}

// doPrintj conditionally prints a log struct with the level and service name in a json format.
// error occurs when json.Marshal fails for json.UnsupportedValueError or json.UnsupportedTypeError and will output the error instead.
// If log level is below level, the message is not output.
func (sl *ServiceLogger) doPrintj(level string, v interface{}) error {
	if sl.logLevel >= levelMap[level] {
		log := ServiceLog{
			Level:   level,
			Service: sl.serviceName,
			Log:     v,
		}
		msg, err := json.Marshal(log)
		if err != nil {
			sl.Output(2, fmt.Sprintf("Failed to write %+v, err: %s", v, err.Error()))
			return err
		}
		sl.Output(2, string(msg))
	}
	return nil
}

// doPrint conditionally prints a message prefixed with the log level and service name,
// followed by a newline. If log level is below level, the message is not output.
func (sl *ServiceLogger) doPrintln(level string, v ...interface{}) {
	if sl.logLevel >= levelMap[level] {
		sl.Output(2, sl.prefixString(level)+fmt.Sprintln(v...))
	}
}
