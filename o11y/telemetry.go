package o11y

import (
	"context"
	"net/url"
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
	cleanup   func() error
}

func NewTelemetry(
	endpoint, serviceName, version string,
	headers map[string]string,
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

	cleanup, _ := initLogger(endpoint, serviceName, headers, environment, enabled)
	logger := global.GetLoggerProvider().Logger(serviceName)

	return &Telemetry{
		logger:    logger,
		prefilled: fields,
		cleanup:   cleanup,
	}
}

func (t *Telemetry) RenewSession() {
	t.prefilled["session.id"] = uuid.New().String()
}

func (t *Telemetry) Close() error {
	return t.cleanup()
}

// region - Private functions

func initLogger(
	endpoint, serviceName string,
	headers map[string]string,
	environment OtelEnvironment,
	enabled bool,
) (func() error, error) {
	if !enabled {
		return func() error { return nil }, nil
	}

	ctx := context.Background()

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return func() error { return nil }, err
	}

	exp, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(parsedURL.Host),
		otlploghttp.WithURLPath(parsedURL.Path+"/v1/logs"),
		otlploghttp.WithHeaders(headers),
	)
	if err != nil {
		return func() error { return nil }, err
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

	return func() error {
		return lp.Shutdown(context.Background())
	}, nil
}

// endregion
