package logging

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ServiceLogger represents a logger with logging level prefixes for a specific service.
type AltServiceLogger struct {
	logConfig   *zap.Config
	serviceName string
	*zap.SugaredLogger
}

var zapLevelMap = map[string]zapcore.Level{
	// Zap does not have a critical level, this just adjusts severity
	"CRITICAL": zapcore.ErrorLevel,
	"ERROR":    zapcore.ErrorLevel,
	"WARN":     zapcore.WarnLevel,
	"INFO":     zapcore.InfoLevel,
	"DEBUG":    zapcore.DebugLevel,
}

type AltServiceLoggerConfig struct {
	Output        string                 //out is the location for logs to be output such as "stdout"
	ServiceName   string                 // service is the value for the "service" key.
	Level         string                 // level is the minimum logging level that will be output
	InitialFields map[string]interface{} // initialFields is a map of key value pairs that will be logged with all log message produced by this logger.
	ConsoleLog    bool                   // consoleLog if set to false outputs in a json like format (Though can have duplicate keys which downstream processors may handle in undefined ways). Formats the log in a more traditional one line fashion with a leading timestamp and log level.
	SkipLevels    int                    // level is used for configuring the caller line number. Services usually want 1, db loggers usually want 2
}

// NewZapSugaredServiceLogger returns a logger with designated logging levels for a particular service.
func NewZapSugaredServiceLogger(lc AltServiceLoggerConfig) AltServiceLogger {
	var sugaredZapLogger *zap.SugaredLogger
	// Get a default configuration
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{lc.Output}
	// Set logging level
	config.Level.SetLevel(zapLevelMap[lc.Level])
	config.EncoderConfig.NameKey = "service"
	config.EncoderConfig.MessageKey = "message"
	if lc.InitialFields == nil {
		lc.InitialFields = make(map[string]interface{}, 1)
	}
	lc.InitialFields["facility"] = "user-level"
	config.InitialFields = lc.InitialFields
	config.EncoderConfig.EncodeLevel = CustomLevelEncoder
	config.EncoderConfig.EncodeTime = SyslogTimeEncoder

	var withCaller bool = true
	if lc.SkipLevels == 1 {
		withCaller = false
	}
	zapLogger, err := config.Build(zap.WithCaller(withCaller))
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
				StacktraceKey:  "stacktrace",
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.EpochTimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				//EncodeCaller:   zapcore.ShortCallerEncoder,
			},

			Facility:  Facility,
			Hostname:  hostName,
			PID:       os.Getpid(),
			App:       lc.ServiceName,
			Formatter: lc.Output,
		})
	} else {
		encoder = zapcore.NewConsoleEncoder(config.EncoderConfig)
	}

	if lc.ConsoleLog {
		config.EncoderConfig.StacktraceKey = ""
		zapLogger = zapLogger.WithOptions(
			zap.WrapCore(
				func(zapcore.Core) zapcore.Core {
					return zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), config.Level)
				}))
	}
	if strings.EqualFold("Default", loggingFormat) {
		sugaredZapLogger = zapLogger.Sugar().Named(lc.ServiceName) // timestamp level servicename message
	} else if strings.EqualFold("Pretty", loggingFormat) {
		sugaredZapLogger = zapLogger.Sugar() // timestamp level message
	} else {
		sugaredZapLogger = zapLogger.Sugar() //syslog format
	}
	sugaredZapLogger.Desugar().With()
	logger := AltServiceLogger{
		logConfig:     &config,
		serviceName:   lc.ServiceName,
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
func (sl *AltServiceLogger) SetLevel(level string) {
	zapLevel, exists := zapLevelMap[level]
	if !exists {
		sl.Printf("Unknown logging level %s. Using ERROR instead.", level)
		zapLevel = zapLevelMap["ERROR"]
	}
	sl.logConfig.Level.SetLevel(zapLevel)
}

// Debug sends a debug level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators.
func (sl *AltServiceLogger) Debug(v ...interface{}) {
	sl.SugaredLogger.Debug(v...)
}

// Debugln is equivalent to Debug and is included for interface compatibility reasons.
func (sl *AltServiceLogger) Debugln(v ...interface{}) {
	sl.Debug(v...)
}

// Debugf sends a debug level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
// when log level is DEBUG.
func (sl *AltServiceLogger) Debugf(format string, v ...interface{}) {
	sl.SugaredLogger.Debugf(format, v...)
}

// Debugw, debug "with" structured data, sends a debug level log to the configured log output.
// Severity and severity-code are set to debug level as per RFC 5424 for User-level facility
// Where message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *AltServiceLogger) Debugw(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "DEBUG", "severity-code", "15", "requester-ip", ip)
	sl.SugaredLogger.Debugw(message, v...)
}

// Info sends an info level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators.
func (sl *AltServiceLogger) Info(v ...interface{}) {
	sl.SugaredLogger.Info(v...)
}

// Infoln is equivalent to Info and is included for interface compatibility reasons.
func (sl *AltServiceLogger) Infoln(v ...interface{}) {
	sl.SugaredLogger.Info(v...)
}

// Infof sends an info level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
// when log level is INFO or above.
func (sl *AltServiceLogger) Infof(format string, v ...interface{}) {
	sl.SugaredLogger.Infof(format, v...)
}

// Infow, info "with" structured data, sends an info level log to the configured log output.
// Severity and severity-code are set to info level as per RFC 5424 for User-level facility
// Message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *AltServiceLogger) Infow(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "INFO", "severity-code", "14", "requester-ip", ip)
	sl.SugaredLogger.Infow(message, v...)
}

// Warn sends a warn level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators.
func (sl *AltServiceLogger) Warn(v ...interface{}) {
	sl.SugaredLogger.Warn(v...)
}

// Warnln is equivalent to Warn and is included for interface compatibility reasons.
func (sl *AltServiceLogger) Warnln(v ...interface{}) {
	sl.SugaredLogger.Warn(v...)
}

// Warnf sends a warn level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
// when log level is WARN or above
func (sl *AltServiceLogger) Warnf(format string, v ...interface{}) {
	sl.SugaredLogger.Warnf(format, v...)
}

// Warnw, warn "with" structured data, sends a warn level log to the configured log output.
// Severity and severity-code are set to warn level as per RFC 5424 for User-level facility
// Message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *AltServiceLogger) Warnw(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "WARN", "severity-code", "12", "requester-ip", ip)
	sl.SugaredLogger.Warnw(message, v...)
}

// Error sends an error level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators
func (sl *AltServiceLogger) Error(v ...interface{}) {
	sl.SugaredLogger.Error(v...)
}

// Errorln is equivalent to Error and is included for interface compatibility reasons..
func (sl *AltServiceLogger) Errorln(v ...interface{}) {
	sl.SugaredLogger.Error(v...)
}

// Errorf sends an error level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
func (sl *AltServiceLogger) Errorf(format string, v ...interface{}) {
	sl.SugaredLogger.Errorf(format, v...)
}

// Errorw, error "with" structured data, sends an error level log to the configured log output.
// Severity and severity-code are set to Error level as per RFC 5424 for User-level facility
// Message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *AltServiceLogger) Errorw(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "ERROR", "severity-code", "11", "requester-ip", ip)
	sl.SugaredLogger.Errorw(message, v...)
}

// Critical sends an error level log to the configured log output.
// Each variadic value is concatanated to the previous one with no additional separators
func (sl *AltServiceLogger) Critical(v ...interface{}) {
	sl.SugaredLogger.Error(v...)
}

// Criticalln is equivalent to Critical and is included for interface compatibility reasons..
func (sl *AltServiceLogger) Criticalln(v ...interface{}) {
	sl.Critical(v...)
}

// Criticalf sends an error level log to the configured log output.
// Where format is a Printf style string and the variadic values are the values in the formatting string
func (sl *AltServiceLogger) Criticalf(format string, v ...interface{}) {
	sl.SugaredLogger.Errorf(format, v...)
}

// Criticalw, critical "with" structured data, sends an error level log to the configured log output.
// Severity and severity-code are set to Critical level as per RFC 5424 for User-level facility
// Message is a string that is the values associated with the `message` key.
// If r is not nil, the IP address of caller will be added to key `requester-ip`
// The variadic values are key-value pairs that must be string: interface{}.
// If duplicate keys are provided the logger will output all sets though downstream log processors that
func (sl *AltServiceLogger) CriticalW(message string, r *http.Request, v ...interface{}) {
	var ip string
	if r != nil {
		ip = getIP(r)
	}
	v = append(v, "severity", "CRITICAL", "severity-code", "10", "requester-ip", ip)
	sl.SugaredLogger.Errorw(message, v...)
}

// Print with "SERVICENAME: " prepended. Only output when log level is SERVICE or higher.
func (sl *AltServiceLogger) Print(v ...interface{}) {
	sl.Info(v...)
}

// Println with "SERVICENAME: " prepended. Only output when log level is SERVICE or higher.
func (sl *AltServiceLogger) Println(v ...interface{}) {
	sl.Info(v...)
}

// Printf with "SERVICENAME: " prepended. Only output when log level is SERVICE or higher.
func (sl *AltServiceLogger) Printf(format string, v ...interface{}) {
	sl.Infof(format, v...)
}

// getIP gets a requests IP address by reading off the forwarded-for
// header (for proxies) and falls back to use the remote address.
func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}
