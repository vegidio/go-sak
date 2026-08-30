package o11y

// Test-only accessors for the enrichment attributes. Production code renders them once into an atomic.Pointer, so the
// tests need a way to look at them as the plain map they were built from.

// prefilled reconstructs the enrichment fields from the rendered attributes.
func (t *Telemetry) prefilled() map[string]any {
	attrs := t.attrs.Load()
	if attrs == nil {
		return nil
	}

	fields := make(map[string]any, len(*attrs))
	for _, kv := range *attrs {
		fields[kv.Key] = kv.Value.AsString()
	}

	return fields
}

// setPrefilled replaces the enrichment attributes, so a test can isolate a record from them.
func (t *Telemetry) setPrefilled(fields map[string]any) {
	t.base = fields

	attrs := attributesFrom(fields)
	t.attrs.Store(&attrs)
}
