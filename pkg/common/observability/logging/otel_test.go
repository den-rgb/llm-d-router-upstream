package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/go-logr/zapr"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestJSONRecordUsesOTelFields(t *testing.T) {
	var buf bytes.Buffer
	core := WrapCore(zapcore.NewCore(zapcore.NewJSONEncoder(EncoderConfig()), zapcore.AddSync(&buf), zapcore.InfoLevel))
	zl := zap.New(core)
	zl.Info("request assembled")
	if err := zl.Sync(); err != nil {
		t.Fatal(err)
	}

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	if rec["body"] != "request assembled" {
		t.Errorf("body = %v, want request assembled", rec["body"])
	}
	if rec["severity_text"] != "INFO" {
		t.Errorf("severity_text = %v, want INFO", rec["severity_text"])
	}
	if rec["severity_number"] != float64(9) {
		t.Errorf("severity_number = %v, want 9", rec["severity_number"])
	}
	if rec["timestamp"] == nil || rec["timestamp"] == "" {
		t.Error("timestamp missing")
	}
}

func TestWithTraceInjectsIDs(t *testing.T) {
	var buf bytes.Buffer
	zl := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(EncoderConfig()), zapcore.AddSync(&buf), zapcore.InfoLevel))

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
	ctx, span := tp.Tracer("test").Start(t.Context(), "request_orchestration")
	defer span.End()

	WithTrace(ctx, zapr.NewLogger(zl)).Info("request assembled")
	if err := zl.Sync(); err != nil {
		t.Fatal(err)
	}

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	sc := span.SpanContext()
	if rec["trace_id"] != sc.TraceID().String() {
		t.Errorf("trace_id = %v, want %s", rec["trace_id"], sc.TraceID().String())
	}
	if rec["span_id"] != sc.SpanID().String() {
		t.Errorf("span_id = %v, want %s", rec["span_id"], sc.SpanID().String())
	}
}
