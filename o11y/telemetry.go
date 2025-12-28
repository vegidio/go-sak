package o11y

import (
	"context"
	"runtime"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Telemetry struct {
	logger    log.Logger
	prefilled map[string]any
	cleanup   func()
}

func NewTelemetry(
	endpoint, serviceName, version string,
	environment OtelEnvironment,
	enabled bool,
) *Telemetry {
	fields := make(map[string]any)

	id, _ := machineid.ID()
	fields["version"] = version
	fields["session.id"] = uuid.New().String()

	// Machine info
	fields["machine.id"] = strings.ToLower(id)
	fields["machine.os"] = runtime.GOOS
	fields["machine.arch"] = runtime.GOARCH

	// Geolocation
	if geo, err := FetchGeolocation(); err == nil {
		fields["location.country"] = geo.Country
		fields["location.region"] = geo.Region
		fields["location.city"] = geo.City
	}

	logger := global.GetLoggerProvider().Logger(serviceName)
	cleanup, _ := initLogger(endpoint, serviceName, environment, enabled)

	return &Telemetry{
		logger:    logger,
		prefilled: fields,
		cleanup:   cleanup,
	}
}

func (t *Telemetry) RenewSession() {
	t.prefilled["session.id"] = uuid.New().String()
}

func (t *Telemetry) Close() {
	t.cleanup()
}

// region - Private functions

func initLogger(
	endpoint, serviceName string,
	environment OtelEnvironment,
	enabled bool,
) (func(), error) {
	if !enabled {
		return func() {}, nil
	}

	ctx := context.Background()

	exp, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return func() {}, err
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
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
	)

	global.SetLoggerProvider(lp)

	return func() { _ = lp.Shutdown(context.Background()) }, nil
}

// endregion
