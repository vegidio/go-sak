package o11y

import (
	"context"
	"fmt"
	"strings"
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
	record.SetSeverityText(strings.ToLower(severity.String()))
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
		case []bool, []string, []int, []int64, []float64:
			attrs = append(attrs, handleSlice(k, val))
		case map[string]bool, map[string]string, map[string]int, map[string]int64, map[string]float64:
			attrs = append(attrs, handleMap(k, val))
		default:
			// Fallback to string representation
			attrs = append(attrs, log.String(k, fmt.Sprintf("%v", val)))
		}
	}

	return attrs
}

// endregion

// region - Private functions

func handleSlice(key string, slice any) log.KeyValue {
	var values []log.Value

	switch val := slice.(type) {
	case []bool:
		values = lo.Map(val, func(item bool, _ int) log.Value {
			return log.BoolValue(item)
		})
	case []string:
		values = lo.Map(val, func(item string, _ int) log.Value {
			return log.StringValue(item)
		})
	case []int:
		values = lo.Map(val, func(item int, _ int) log.Value {
			return log.IntValue(item)
		})
	case []int64:
		values = lo.Map(val, func(item int64, _ int) log.Value {
			return log.Int64Value(item)
		})
	case []float64:
		values = lo.Map(val, func(item float64, _ int) log.Value {
			return log.Float64Value(item)
		})
	default:
		// Fallback to string representation for unsupported slice types
		values = []log.Value{log.StringValue(fmt.Sprintf("%v", val))}
	}

	return log.Slice(key, values...)
}

func handleMap(key string, m any) log.KeyValue {
	var values []log.KeyValue

	switch val := m.(type) {
	case map[string]bool:
		values = lo.MapToSlice(val, func(k string, v bool) log.KeyValue {
			return log.Bool(k, v)
		})
	case map[string]string:
		values = lo.MapToSlice(val, func(k string, v string) log.KeyValue {
			return log.String(k, v)
		})
	case map[string]int:
		values = lo.MapToSlice(val, func(k string, v int) log.KeyValue {
			return log.Int(k, v)
		})
	case map[string]int64:
		values = lo.MapToSlice(val, func(k string, v int64) log.KeyValue {
			return log.Int64(k, v)
		})
	case map[string]float64:
		values = lo.MapToSlice(val, func(k string, v float64) log.KeyValue {
			return log.Float64(k, v)
		})
	default:
		// Fallback to string representation for unsupported map types
		values = []log.KeyValue{log.String("value", fmt.Sprintf("%v", val))}
	}

	return log.Map(key, values...)
}

// endregion
