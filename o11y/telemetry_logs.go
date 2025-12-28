package o11y

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/log"
)

func (t *Telemetry) LogInfo(event string, fields map[string]any) {
	t.log(event, fields, log.SeverityInfo)
}

func (t *Telemetry) LogWarn(event string, fields map[string]any) {
	t.log(event, fields, log.SeverityWarn)
}

func (t *Telemetry) LogError(event string, fields map[string]any, err error) {
	if fields == nil {
		fields = make(map[string]any)
	}

	fields["error"] = err
	t.log(event, fields, log.SeverityError)
}

// region - Private methods

func (t *Telemetry) log(event string, fields map[string]any, severity log.Severity) {
	var record log.Record

	record.SetTimestamp(time.Now())
	record.SetSeverity(severity)
	record.SetBody(log.StringValue(event))
	record.AddAttributes(t.mapToAttributes(fields)...)

	t.logger.Emit(context.Background(), record)
}

func (t *Telemetry) mapToAttributes(fields map[string]any) []log.KeyValue {
	m := lo.Assign(t.prefilled, fields)
	attrs := make([]log.KeyValue, 0, len(m))

	for k, v := range m {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, log.String(k, val))
		case int:
			attrs = append(attrs, log.Int(k, val))
		case int64:
			attrs = append(attrs, log.Int64(k, val))
		case float64:
			attrs = append(attrs, log.Float64(k, val))
		case bool:
			attrs = append(attrs, log.Bool(k, val))
		default:
			// Fallback to string representation
			attrs = append(attrs, log.String(k, fmt.Sprintf("%v", val)))
		}
	}

	return attrs
}

// endregion
