package logging

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// EncoderConfig returns a zap encoder config that emits OTel Logs Data Model
// field names on stdout JSON records.
func EncoderConfig() zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "timestamp"
	config.LevelKey = "severity_text"
	config.NameKey = "logger"
	config.CallerKey = "caller"
	config.MessageKey = "body"
	config.StacktraceKey = "stacktrace"
	config.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	config.EncodeLevel = LevelEncoder
	return config
}

// SeverityText maps zap / logr verbosity levels to OTel severity_text.
func SeverityText(l zapcore.Level) string {
	if l >= 0 {
		switch {
		case l >= zapcore.DPanicLevel:
			return "FATAL"
		case l >= zapcore.ErrorLevel:
			return "ERROR"
		case l >= zapcore.WarnLevel:
			return "WARN"
		default:
			return "INFO"
		}
	}

	switch l {
	case zapcore.Level(-1 * DEBUG):
		return "DEBUG"
	case zapcore.Level(-1 * TRACE):
		return "TRACE"
	default:
		if l >= zapcore.Level(-1*VERBOSE) {
			return "INFO"
		}
		return "TRACE"
	}
}

// SeverityNumber maps zap / logr verbosity levels to OTel severity_number.
func SeverityNumber(l zapcore.Level) int {
	switch SeverityText(l) {
	case "FATAL":
		return 21
	case "ERROR":
		return 17
	case "WARN":
		return 13
	case "DEBUG":
		return 5
	case "TRACE":
		return 1
	default:
		return 9
	}
}

// ServiceName returns OTEL_SERVICE_NAME or fallback.
func ServiceName(fallback string) string {
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		return name
	}
	return fallback
}

// WrapCore adds severity_number to every log record.
func WrapCore(c zapcore.Core) zapcore.Core {
	return &otelCore{Core: c}
}

type otelCore struct {
	zapcore.Core
}

func (c *otelCore) With(fields []zapcore.Field) zapcore.Core {
	return &otelCore{Core: c.Core.With(fields)}
}

func (c *otelCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *otelCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	fields = append(fields, zap.Int("severity_number", SeverityNumber(ent.Level)))
	return c.Core.Write(ent, fields)
}

// WithTrace returns logger with trace_id/span_id when a span is active on ctx.
func WithTrace(ctx context.Context, logger logr.Logger) logr.Logger {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return logger
	}
	return logger.WithValues("trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
}
