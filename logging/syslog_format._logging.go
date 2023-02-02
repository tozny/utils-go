package logging

import (
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/tozny/utils-go"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var (
	_pool = buffer.NewPool()
	// Get retrieves a buffer from the pool, creating one if necessary.
	Get = _pool.Get

	_ zapcore.Encoder = &syslogEncoder{}
	_                 = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()).(genericEncoder)
)

// Priority maps to the syslog priority levels
type Priority int

const (
	severityMask    = 0x07
	facilityMask    = 0xf8
	nilValue        = "-"
	timestampFormat = "2006-01-02T15:04:05.000Z"
	maxHostnameLen  = 255
	maxAppNameLen   = 48

	NonTransparentFraming Framing = iota
	OctetCountingFraming
	DefaultFraming = NonTransparentFraming

	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)
const (
	// Facility.

	// From /usr/include/sys/syslog.h.
	// These are the same up to LOG_FTP on Linux, BSD, and OS X.

	LOG_KERN Priority = iota << 3
	LOG_USER
	LOG_MAIL
	LOG_DAEMON
	LOG_AUTH
	LOG_SYSLOG
	LOG_LPR
	LOG_NEWS
	LOG_UUCP
	LOG_CRON
	LOG_AUTHPRIV
	LOG_FTP
	_ // unused
	_ // unused
	_ // unused
	_ // unused
	LOG_LOCAL0
	LOG_LOCAL1
	LOG_LOCAL2
	LOG_LOCAL3
	LOG_LOCAL4
	LOG_LOCAL5
	LOG_LOCAL6
	LOG_LOCAL7
)

var (
	facilityMap = map[string]Priority{
		"KERN":     LOG_KERN,
		"USER":     LOG_USER,
		"MAIL":     LOG_MAIL,
		"DAEMON":   LOG_DAEMON,
		"AUTH":     LOG_AUTH,
		"SYSLOG":   LOG_SYSLOG,
		"LPR":      LOG_LPR,
		"NEWS":     LOG_NEWS,
		"UUCP":     LOG_UUCP,
		"CRON":     LOG_CRON,
		"AUTHPRIV": LOG_AUTHPRIV,
		"FTP":      LOG_FTP,
		"LOCAL0":   LOG_LOCAL0,
		"LOCAL1":   LOG_LOCAL1,
		"LOCAL2":   LOG_LOCAL2,
		"LOCAL3":   LOG_LOCAL3,
		"LOCAL4":   LOG_LOCAL4,
		"LOCAL5":   LOG_LOCAL5,
		"LOCAL6":   LOG_LOCAL6,
		"LOCAL7":   LOG_LOCAL7,
	}
)
var Sversion = utils.EnvOrDefault("SYSLOG_VERSION", "1")
var version, err = strconv.ParseInt(Sversion, 10, 64)
var loggingFormat = utils.EnvOrDefault("LOGGING_FORMAT", "DEFAULT")
var Facility = FacilityPriority(utils.EnvOrDefault("FACILITY_VALUE", "LOCAL0"))
var hostName = utils.EnvOrDefault("HOSTNAME", "")

type Framing int

type genericEncoder interface {
	zapcore.Encoder
	zapcore.ArrayEncoder
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.String() + "] :")
}

func SyslogTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(timestampFormat)) //"2006-01-02T15:04:05.000Z"))
}

// SyslogEncoderConfig allows users to configure the concrete encoders for zap syslog.
type SyslogEncoderConfig struct {
	zapcore.EncoderConfig

	Framing   Framing  `json:"framing" yaml:"framing"`
	Facility  Priority `json:"facility" yaml:"facility"`
	Hostname  string   `json:"hostname" yaml:"hostname"`
	PID       int      `json:"pid" yaml:"pid"`
	App       string   `json:"app" yaml:"app"`
	Formatter string   `json:"formatter" yaml:"formatter"`
}

type syslogEncoder struct {
	*SyslogEncoderConfig
	je genericEncoder
}

// FacilityPriority converts a facility string into
// an appropriate priority level or returns an error
func FacilityPriority(facility string) Priority {
	facility = strings.ToUpper(facility)
	if prio, ok := facilityMap[facility]; ok {
		return prio
	}
	return 0
}

func rfc5424CompliantASCIIMapper(r rune) rune {
	// PRINTUSASCII    = %d33-126
	if r < 33 || r > 126 {
		return '_'
	}
	return r
}

func toRFC5424CompliantASCIIString(s string) string {
	return strings.Map(rfc5424CompliantASCIIMapper, s)
}
func BytesToString(b []byte) string {
	return string(b)
}

// StringToBytes converts string to byte slice.
func StringToBytes(s string) []byte {
	return []byte(s)
}
func NewSyslogEncoder(cfg SyslogEncoderConfig) zapcore.Encoder {
	if cfg.Hostname == "" {
		hostname, _ := os.Hostname()
		cfg.Hostname = hostname
	}
	if cfg.Hostname == "" {
		cfg.Hostname = nilValue
	} else {
		hostname := toRFC5424CompliantASCIIString(cfg.Hostname)
		if len(hostname) > maxHostnameLen {
			hostname = hostname[:maxHostnameLen]
		}
		cfg.Hostname = hostname
	}

	if cfg.PID == 0 {
		cfg.PID = os.Getpid()
	}
	if cfg.App == "" {
		cfg.App = nilValue
	} else {
		app := cfg.App
		if len(app) > maxAppNameLen {
			app = path.Base(app)
		}
		if len(app) > maxAppNameLen {
			app = app[:maxAppNameLen]
		}
		app = toRFC5424CompliantASCIIString(app)
	}

	cfg.EncoderConfig.LineEnding = "\n"

	var ge genericEncoder
	switch cfg.Formatter {
	case "stdout":
		ge = zapcore.NewConsoleEncoder(cfg.EncoderConfig).(genericEncoder)
	case "json":
		fallthrough
	default:
		ge = zapcore.NewJSONEncoder(cfg.EncoderConfig).(genericEncoder)
	}
	return &syslogEncoder{
		SyslogEncoderConfig: &cfg,
		je:                  ge,
	}
}

func (enc *syslogEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	return enc.je.AddArray(key, arr)
}

func (enc *syslogEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	return enc.je.AddObject(key, obj)
}

func (enc *syslogEncoder) AddBinary(key string, val []byte)          { enc.je.AddBinary(key, val) }
func (enc *syslogEncoder) AddByteString(key string, val []byte)      { enc.je.AddByteString(key, val) }
func (enc *syslogEncoder) AddBool(key string, val bool)              { enc.je.AddBool(key, val) }
func (enc *syslogEncoder) AddComplex128(key string, val complex128)  { enc.je.AddComplex128(key, val) }
func (enc *syslogEncoder) AddDuration(key string, val time.Duration) { enc.je.AddDuration(key, val) }
func (enc *syslogEncoder) AddFloat64(key string, val float64)        { enc.je.AddFloat64(key, val) }
func (enc *syslogEncoder) AddInt64(key string, val int64)            { enc.je.AddInt64(key, val) }

func (enc *syslogEncoder) AddReflected(key string, obj interface{}) error {
	return enc.je.AddReflected(key, obj)
}

func (enc *syslogEncoder) OpenNamespace(key string)          { enc.je.OpenNamespace(key) }
func (enc *syslogEncoder) AddString(key, val string)         { enc.je.AddString(key, val) }
func (enc *syslogEncoder) AddTime(key string, val time.Time) { enc.je.AddTime(key, val) }
func (enc *syslogEncoder) AddUint64(key string, val uint64)  { enc.je.AddUint64(key, val) }

func (enc *syslogEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	return enc.je.AppendArray(arr)
}

func (enc *syslogEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	return enc.je.AppendObject(obj)
}

func (enc *syslogEncoder) AppendBool(val bool)              { enc.je.AppendBool(val) }
func (enc *syslogEncoder) AppendByteString(val []byte)      { enc.je.AppendByteString(val) }
func (enc *syslogEncoder) AppendComplex128(val complex128)  { enc.je.AppendComplex128(val) }
func (enc *syslogEncoder) AppendDuration(val time.Duration) { enc.je.AppendDuration(val) }
func (enc *syslogEncoder) AppendInt64(val int64)            { enc.je.AppendInt64(val) }

func (enc *syslogEncoder) AppendReflected(val interface{}) error {
	return enc.je.AppendReflected(val)
}

func (enc *syslogEncoder) AppendString(val string)            { enc.je.AppendString(val) }
func (enc *syslogEncoder) AppendTime(val time.Time)           { enc.je.AppendTime(val) }
func (enc *syslogEncoder) AppendUint64(val uint64)            { enc.je.AppendUint64(val) }
func (enc *syslogEncoder) AddComplex64(k string, v complex64) { enc.je.AddComplex64(k, v) }
func (enc *syslogEncoder) AddFloat32(k string, v float32)     { enc.je.AddFloat32(k, v) }
func (enc *syslogEncoder) AddInt(k string, v int)             { enc.je.AddInt(k, v) }
func (enc *syslogEncoder) AddInt32(k string, v int32)         { enc.je.AddInt32(k, v) }
func (enc *syslogEncoder) AddInt16(k string, v int16)         { enc.je.AddInt16(k, v) }
func (enc *syslogEncoder) AddInt8(k string, v int8)           { enc.je.AddInt8(k, v) }
func (enc *syslogEncoder) AddUint(k string, v uint)           { enc.je.AddUint(k, v) }
func (enc *syslogEncoder) AddUint32(k string, v uint32)       { enc.je.AddUint32(k, v) }
func (enc *syslogEncoder) AddUint16(k string, v uint16)       { enc.je.AddUint16(k, v) }
func (enc *syslogEncoder) AddUint8(k string, v uint8)         { enc.je.AddUint8(k, v) }
func (enc *syslogEncoder) AddUintptr(k string, v uintptr)     { enc.je.AddUintptr(k, v) }
func (enc *syslogEncoder) AppendComplex64(v complex64)        { enc.je.AppendComplex64(v) }
func (enc *syslogEncoder) AppendFloat64(v float64)            { enc.je.AppendFloat64(v) }
func (enc *syslogEncoder) AppendFloat32(v float32)            { enc.je.AppendFloat32(v) }
func (enc *syslogEncoder) AppendInt(v int)                    { enc.je.AppendInt(v) }
func (enc *syslogEncoder) AppendInt32(v int32)                { enc.je.AppendInt32(v) }
func (enc *syslogEncoder) AppendInt16(v int16)                { enc.je.AppendInt16(v) }
func (enc *syslogEncoder) AppendInt8(v int8)                  { enc.je.AppendInt8(v) }
func (enc *syslogEncoder) AppendUint(v uint)                  { enc.je.AppendUint(v) }
func (enc *syslogEncoder) AppendUint32(v uint32)              { enc.je.AppendUint32(v) }
func (enc *syslogEncoder) AppendUint16(v uint16)              { enc.je.AppendUint16(v) }
func (enc *syslogEncoder) AppendUint8(v uint8)                { enc.je.AppendUint8(v) }
func (enc *syslogEncoder) AppendUintptr(v uintptr)            { enc.je.AppendUintptr(v) }

func (enc *syslogEncoder) Clone() zapcore.Encoder {
	return enc.clone()
}

func (enc *syslogEncoder) clone() *syslogEncoder {
	clone := &syslogEncoder{
		SyslogEncoderConfig: enc.SyslogEncoderConfig,
		je:                  enc.je.Clone().(genericEncoder),
	}
	return clone
}

func (enc *syslogEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	msg := buffer.NewPool().Get()

	var p Priority
	switch ent.Level {
	case zapcore.FatalLevel:
		p = LOG_EMERG
	case zapcore.PanicLevel:
		p = LOG_CRIT
	case zapcore.DPanicLevel:
		p = LOG_CRIT
	case zapcore.ErrorLevel:
		p = LOG_ERR
	case zapcore.WarnLevel:
		p = LOG_WARNING
	case zapcore.InfoLevel:
		p = LOG_INFO
	case zapcore.DebugLevel:
		p = LOG_DEBUG
	}
	pr := int64((enc.Facility & facilityMask) | (p & severityMask))

	// <PRI>
	msg.AppendByte('<')
	msg.AppendInt(pr)
	msg.AppendByte('>')

	msg.AppendInt(version)
	msg.AppendByte(' ')

	// SP TIMESTAMP
	if ent.Time.IsZero() {
		msg.AppendString(nilValue)
	} else {
		msg.AppendString(ent.Time.Format(timestampFormat))
	}

	// SP HOSTNAME
	msg.AppendByte(' ')
	msg.AppendString(enc.Hostname)

	// SP APP-NAME
	msg.AppendByte(' ')
	msg.AppendString(enc.App)

	// SP PROCID
	//msg.AppendByte('[')]
	msg.AppendByte(' ')
	msg.AppendInt(int64(enc.PID))
	//msg.AppendString("] ")

	// SP MSGID SP STRUCTURED-DATA (just ignore)
	msg.AppendByte(' ')
	msg.AppendString("-- ")

	// SP UTF8 MSG
	json, err := enc.je.EncodeEntry(ent, fields)
	if json.Len() > 0 {

		bs := json.Bytes()
		if enc.Framing == OctetCountingFraming {
			// Strip trailing line feed
			bs = bs[:len(bs)-1]
		}
		msg.AppendString(BytesToString(bs))
	}

	if enc.Framing != OctetCountingFraming {
		return msg, err
	}

	// SYSLOG-FRAME = MSG-LEN SP SYSLOG-MSG
	out := buffer.NewPool().Get()
	out.AppendInt(int64(msg.Len()))
	out.AppendByte(' ')
	out.AppendString(BytesToString(msg.Bytes()))
	msg.Free()
	return out, err
}
