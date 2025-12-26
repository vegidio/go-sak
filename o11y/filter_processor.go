package o11y

import (
	"context"

	log "github.com/sirupsen/logrus"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// filterProcessor wraps another processor and filters by log level
type filterProcessor struct {
	next     sdklog.Processor
	minLevel log.Level
}

func newFilterProcessor(next sdklog.Processor, minLevel log.Level) *filterProcessor {
	return &filterProcessor{next: next, minLevel: minLevel}
}

func (f *filterProcessor) OnEmit(ctx context.Context, record *sdklog.Record) error {
	// Map OTel severity to Logrus level
	severity := record.Severity()

	// Skip if below minimum level
	// OTel severities: Debug=5, Info=9, Warn=13, Error=17
	var shouldExport bool
	switch f.minLevel {
	case log.InfoLevel:
		shouldExport = severity >= 9 // Info and above
	case log.WarnLevel:
		shouldExport = severity >= 13 // Warn and above
	case log.ErrorLevel:
		shouldExport = severity >= 17 // Error and above
	default:
		shouldExport = true
	}

	if !shouldExport {
		return nil
	}

	return f.next.OnEmit(ctx, record)
}

func (f *filterProcessor) Enabled(ctx context.Context, param sdklog.EnabledParameters) bool {
	return f.next.Enabled(ctx, param)
}

func (f *filterProcessor) Shutdown(ctx context.Context) error {
	return f.next.Shutdown(ctx)
}

func (f *filterProcessor) ForceFlush(ctx context.Context) error {
	return f.next.ForceFlush(ctx)
}
