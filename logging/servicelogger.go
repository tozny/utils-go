package logging

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ServiceLogger struct {
	logConfig   *zap.Config
	serviceName string
	*zap.SugaredLogger
}

// NewZapSugaredServiceLogger returns a logger with designated logging levels for a particular service.
func NewServiceLogger(out io.Writer, serviceName string, level string) ServiceLogger {
	var sugaredZapLogger *zap.SugaredLogger
	// Get a default configuration
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	// Set logging level
	config.Level.SetLevel(zapLevelMap[level])
	config.EncoderConfig.NameKey = "service"
	config.EncoderConfig.MessageKey = "message"

	initialFields := make(map[string]interface{}, 1)

	initialFields["facility"] = "user-level"
	config.InitialFields = initialFields
	config.EncoderConfig.EncodeLevel = CustomLevelEncoder
	config.EncoderConfig.EncodeTime = SyslogTimeEncoder

	zapLogger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(fmt.Errorf("Logger could not be built. This is not an expected outcome. ERR: %+v", err))
	}

	var encoder zapcore.Encoder
	if strings.EqualFold("Syslog", loggingFormat) {
		encoder = NewSyslogEncoder(SyslogEncoderConfig{
			EncoderConfig: zapcore.EncoderConfig{
				NameKey:        "logger",
				CallerKey:      "caller",
				MessageKey:     "msg",
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.EpochTimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			},

			Facility:  Facility,
			Hostname:  hostName, //if no value passed it set from os.getHostName
			PID:       os.Getpid(),
			App:       serviceName,
			Formatter: "stdout",
		})
	} else {
		config.EncoderConfig.StacktraceKey = ""
		encoder = zapcore.NewConsoleEncoder(config.EncoderConfig)
	}

	//if lc.ConsoleLog {
	zapLogger = zapLogger.WithOptions(
		zap.WrapCore(
			func(zapcore.Core) zapcore.Core {
				return zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), config.Level)
			}))

	if strings.EqualFold("Default", loggingFormat) {
		sugaredZapLogger = zapLogger.Sugar().Named(serviceName)
	} else if strings.EqualFold("Pretty", loggingFormat) {
		sugaredZapLogger = zapLogger.Sugar()
	} else {
		sugaredZapLogger = zapLogger.Sugar()
	}
	sugaredZapLogger.Desugar().With()
	logger := ServiceLogger{
		logConfig:     &config,
		serviceName:   serviceName,
		SugaredLogger: sugaredZapLogger,
	}
	return logger
}

// SetLevel allows the log level of the ServiceLogger to be updated based on
// supported log level strings. These include in order:
//   - "CRITICAL"
//   - "ERROR"
//   - "WARN"
//   - "INFO"
//   - "DEBUG"
func (sl *ServiceLogger) SetLevel(level string) {
	zapLevel, exists := zapLevelMap[level]
	if !exists {
		sl.Printf("Unknown logging level %s. Using ERROR instead.", level)
		zapLevel = zapLevelMap["ERROR"]
	}
	sl.logConfig.Level.SetLevel(zapLevel)
}

// Debug sends a debug level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators.
func (sl *ServiceLogger) Debug(v ...interface{}) {
	sl.SugaredLogger.Debug(v...)
}

// Debugln is equivalent to Debug and is included for interface compatibility reasons.
func (sl *ServiceLogger) Debugln(v ...interface{}) {
	sl.Debug(v...)
}

// Debugf sends a debug level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
// when log level is DEBUG.
func (sl *ServiceLogger) Debugf(format string, v ...interface{}) {
	sl.SugaredLogger.Debugf(format, v...)
}

// Debugw, debug "with" structured data, sends a debug level log to the configured log output.
// Severity and severity-code are set to debug level as per RFC 5424 for User-level facility
// Where message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *ServiceLogger) Debugw(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "DEBUG", "severity-code", "15", "requester-ip", ip)
	sl.SugaredLogger.Debugw(message, v...)
}

// Info sends an info level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators.
func (sl *ServiceLogger) Info(v ...interface{}) {
	sl.SugaredLogger.Info(v...)
}

// Infoln is equivalent to Info and is included for interface compatibility reasons.
func (sl *ServiceLogger) Infoln(v ...interface{}) {
	sl.SugaredLogger.Info(v...)
}

// Infof sends an info level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
// when log level is INFO or above.
func (sl *ServiceLogger) Infof(format string, v ...interface{}) {
	sl.SugaredLogger.Infof(format, v...)
}

// Infow, info "with" structured data, sends an info level log to the configured log output.
// Severity and severity-code are set to info level as per RFC 5424 for User-level facility
// Message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *ServiceLogger) Infow(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "INFO", "severity-code", "14", "requester-ip", ip)
	sl.SugaredLogger.Infow(message, v...)
}

// Warn sends a warn level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators.
func (sl *ServiceLogger) Warn(v ...interface{}) {
	sl.SugaredLogger.Warn(v...)
}

// Warnln is equivalent to Warn and is included for interface compatibility reasons.
func (sl *ServiceLogger) Warnln(v ...interface{}) {
	sl.SugaredLogger.Warn(v...)
}

// Warnf sends a warn level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
// when log level is WARN or above
func (sl *ServiceLogger) Warnf(format string, v ...interface{}) {
	sl.SugaredLogger.Warnf(format, v...)
}

// Warnw, warn "with" structured data, sends a warn level log to the configured log output.
// Severity and severity-code are set to warn level as per RFC 5424 for User-level facility
// Message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *ServiceLogger) Warnw(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "WARN", "severity-code", "12", "requester-ip", ip)
	sl.SugaredLogger.Warnw(message, v...)
}

// Error sends an error level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators
func (sl *ServiceLogger) Error(v ...interface{}) {
	sl.SugaredLogger.Error(v...)
}

// Errorln is equivalent to Error and is included for interface compatibility reasons..
func (sl *ServiceLogger) Errorln(v ...interface{}) {
	sl.SugaredLogger.Error(v...)
}

// Errorf sends an error level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
func (sl *ServiceLogger) Errorf(format string, v ...interface{}) {
	sl.SugaredLogger.Errorf(format, v...)
}

// Errorw, error "with" structured data, sends an error level log to the configured log output.
// Severity and severity-code are set to Error level as per RFC 5424 for User-level facility
// Message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *ServiceLogger) Errorw(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "ERROR", "severity-code", "11", "requester-ip", ip)
	sl.SugaredLogger.Errorw(message, v...)
}

// Critical sends an error level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators
func (sl *ServiceLogger) Critical(v ...interface{}) {
	sl.SugaredLogger.Error(v...)
}

// Criticalln is equivalent to Critical and is included for interface compatibility reasons..
func (sl *ServiceLogger) Criticalln(v ...interface{}) {
	sl.Critical(v...)
}

// Criticalf sends an error level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
func (sl *ServiceLogger) Criticalf(format string, v ...interface{}) {
	sl.SugaredLogger.Errorf(format, v...)
}

// Criticalw, critical "with" structured data, sends an error level log to the configured log output.
// Severity and severity-code are set to Critical level as per RFC 5424 for User-level facility
// Message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *ServiceLogger) CriticalW(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "CRITICAL", "severity-code", "10", "requester-ip", ip)
	sl.SugaredLogger.Errorw(message, v...)
}

// Print with "SERVICENAME: " prepended. Only output when log level is SERVICE or higher.
func (sl *ServiceLogger) Print(v ...interface{}) {
	sl.Info(v...)
}

// Println with "SERVICENAME: " prepended. Only output when log level is SERVICE or higher.
func (sl *ServiceLogger) Println(v ...interface{}) {
	sl.Info(v...)
}

// Printf with "SERVICENAME: " prepended. Only output when log level is SERVICE or higher.
func (sl *ServiceLogger) Printf(format string, v ...interface{}) {
	sl.Infof(format, v...)
}
