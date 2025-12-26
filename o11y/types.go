package o11y

// LogDestination specifies where logs should be sent.
type LogDestination string

const (
	// LogToNone disables logging output.
	LogToNone LogDestination = "none"
	// LogToTerminal sends logs to the terminal/console.
	LogToTerminal LogDestination = "terminal"
	// LogToOTel sends logs to OpenTelemetry.
	LogToOTel LogDestination = "otel"
	// LogToBoth sends logs to both terminal and OpenTelemetry.
	LogToBoth LogDestination = "both"
)

// OtelEnvironment specifies the OpenTelemetry environment configuration.
type OtelEnvironment string

const (
	// EnvDevelopment configures OpenTelemetry for development environment.
	EnvDevelopment OtelEnvironment = "development"
	// EnvProduction configures OpenTelemetry for production environment.
	EnvProduction OtelEnvironment = "production"
)
