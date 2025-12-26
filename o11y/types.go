package o11y

// OtelEnvironment specifies the OpenTelemetry environment configuration.
type OtelEnvironment string

const (
	EnvDevelopment OtelEnvironment = "development"
	EnvProduction  OtelEnvironment = "production"
)
