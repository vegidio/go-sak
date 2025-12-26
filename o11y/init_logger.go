package o11y

import (
	"context"
	"io"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitLogger initializes the logging system with OpenTelemetry integration.
//
// It configures Logrus as the logging framework and optionally bridges logs to an OpenTelemetry collector via
// OTLP/HTTP. The function supports four logging destinations: none, terminal only, OTel only, or both simultaneously.
//
// # Parameters:
//   - endpoint: The OTLP collector endpoint (e.g., "localhost:4318"). Only used when destination includes LogToOTel.
//   - serviceName: The name of the service for identifying logs in the observability backend.
//   - environment: The deployment environment (e.g., development, production).
//   - destination: Specifies where logs should be sent (LogToNone, LogToNone, LogToOTel, LogToBoth).
//
// # Returns:
//   - A cleanup function that should be deferred to properly shutdown the logger provider.
//   - An error if the OTLP exporter or resource creation fails.
//
// Note: When logging to OTel, only logs at Info level and above are exported to the collector, while Debug logs remain
// available in terminal output (if LogToBoth is used).
func InitLogger(
	endpoint, serviceName string,
	environment OtelEnvironment,
	destination LogDestination,
) (func(), error) {
	// If no logging, discard everything and return early
	if destination == LogToNone {
		log.SetOutput(io.Discard)
		return func() {}, nil
	}

	// Set Logrus to Debug level for terminal output
	log.SetLevel(log.DebugLevel)

	// Suppress terminal output if logging exclusively to OTel
	if destination == LogToOTel {
		log.SetOutput(io.Discard)
	}

	// Only set up OTLP if needed
	if destination == LogToOTel || destination == LogToBoth {
		ctx := context.Background()

		exp, err := otlploghttp.New(ctx,
			otlploghttp.WithEndpoint(endpoint),
			otlploghttp.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}

		res, err := resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceName(serviceName),
				semconv.DeploymentEnvironment(string(environment)),
			),
		)
		if err != nil {
			return nil, err
		}

		lp := sdklog.NewLoggerProvider(
			sdklog.WithResource(res),

			// Filter processor: only Info and above go to OTel
			sdklog.WithProcessor(newFilterProcessor(
				sdklog.NewBatchProcessor(exp),
				log.InfoLevel,
			)),
		)

		global.SetLoggerProvider(lp)

		// Bridge Logrus -> OpenTelemetry Logs
		log.AddHook(otellogrus.NewHook(serviceName, otellogrus.WithLoggerProvider(lp)))

		// Return cleanup function
		return func() { _ = lp.Shutdown(context.Background()) }, nil
	}

	// No cleanup needed for terminal-only logging
	return func() {}, nil
}
