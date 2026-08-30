package o11y

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
)

func newTestTelemetry() *Telemetry {
	tel, _ := NewTelemetry(
		"localhost:4318",
		"test-service",
		"1.0.0",
		nil,
		EnvDevelopment,
		false,
	)

	return tel
}

func TestLogInfo(t *testing.T) {
	tel := newTestTelemetry()
	defer tel.Close()

	assert.NotPanics(t, func() {
		tel.LogInfo("startup", map[string]any{"port": 8080})
	})
	assert.NotPanics(t, func() {
		tel.LogInfo("nil-fields", nil)
	})
}

func TestLogWarn(t *testing.T) {
	tel := newTestTelemetry()
	defer tel.Close()

	assert.NotPanics(t, func() {
		tel.LogWarn("slow-request", map[string]any{"duration_ms": int64(1200)})
	})
}

func TestLogError(t *testing.T) {
	tel := newTestTelemetry()
	defer tel.Close()

	t.Run("does not mutate the caller's fields", func(t *testing.T) {
		// The error belongs on the record, not in the map the caller passed in and may well reuse.
		fields := map[string]any{"op": "connect"}
		tel.LogError("db failure", fields, errors.New("boom"))

		assert.Equal(t, map[string]any{"op": "connect"}, fields)
		assert.NotContains(t, fields, "error")
	})

	t.Run("renders the error onto the record", func(t *testing.T) {
		attrs := tel.mapToAttributes(map[string]any{"error": errors.New("boom")})

		var found bool
		for _, a := range attrs {
			if a.Key == "error" {
				found = true
				assert.Contains(t, a.Value.AsString(), "boom")
			}
		}
		assert.True(t, found, "the error must reach the emitted record")
	})

	t.Run("handles nil fields", func(t *testing.T) {
		assert.NotPanics(t, func() {
			tel.LogError("nil-fields", nil, errors.New("boom"))
		})
	})
}

func TestMapToAttributes(t *testing.T) {
	tel := newTestTelemetry()
	defer tel.Close()
	tel.setPrefilled(map[string]any{}) // isolate from auto-prefilled fields

	t.Run("covers primitive types", func(t *testing.T) {
		attrs := tel.mapToAttributes(map[string]any{
			"s": "hello",
			"i": 42,
			"l": int64(99),
			"f": 3.14,
			"b": true,
		})
		require.Len(t, attrs, 5)

		byKey := map[string]log.KeyValue{}
		for _, a := range attrs {
			byKey[a.Key] = a
		}
		assert.Equal(t, log.KindString, byKey["s"].Value.Kind())
		assert.Equal(t, log.KindInt64, byKey["i"].Value.Kind())
		assert.Equal(t, log.KindInt64, byKey["l"].Value.Kind())
		assert.Equal(t, log.KindFloat64, byKey["f"].Value.Kind())
		assert.Equal(t, log.KindBool, byKey["b"].Value.Kind())
	})

	t.Run("covers slice and map types", func(t *testing.T) {
		attrs := tel.mapToAttributes(map[string]any{
			"tags":  []string{"a", "b"},
			"stats": map[string]int{"x": 1},
		})
		require.Len(t, attrs, 2)
		byKey := map[string]log.KeyValue{}
		for _, a := range attrs {
			byKey[a.Key] = a
		}
		assert.Equal(t, log.KindSlice, byKey["tags"].Value.Kind())
		assert.Equal(t, log.KindMap, byKey["stats"].Value.Kind())
	})

	t.Run("falls back to string for unknown types", func(t *testing.T) {
		type custom struct{ Name string }
		attrs := tel.mapToAttributes(map[string]any{"c": custom{Name: "x"}})
		require.Len(t, attrs, 1)
		assert.Equal(t, log.KindString, attrs[0].Value.Kind())
	})
}
