package o11y

import (
	"context"
	"maps"
	"net/url"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Telemetry emits structured log records to an OpenTelemetry collector, enriched with a fixed set of attributes
// describing the machine and the current session.
//
// A Telemetry is safe for concurrent use: the enrichment attributes are rendered once and swapped atomically, so
// RenewSession may run concurrently with any of the Log methods.
type Telemetry struct {
	logger log.Logger

	// base holds the enrichment fields other than the session id. It is written once, during construction, and
	// only ever read afterwards.
	base map[string]any

	// attrs is base plus the current session id, pre-rendered so that the common path of emitting a record with
	// no extra fields costs no allocation at all.
	attrs atomic.Pointer[[]log.KeyValue]

	cleanup func() error
}

// NewTelemetry creates a Telemetry that ships log records to an OpenTelemetry collector.
//
// When enabled is false no exporter is installed and no record ever leaves the process. Nothing is sent over the
// network to build the enrichment attributes either: in particular the geolocation lookup, which would disclose the
// caller's public IP to a third party, is skipped entirely. The returned Telemetry is still fully usable; its Log
// methods simply discard.
//
// # Parameters:
//   - endpoint: base URL of the OTLP collector; "/v1/logs" is appended to its path
//   - serviceName: the service name reported to the collector, also used to scope the machine identifier
//   - version: the application version reported with every record
//   - headers: extra headers for the OTLP exporter, typically authentication
//   - environment: EnvDevelopment or EnvProduction
//   - enabled: whether to collect and export anything at all
//
// # Returns:
//   - A Telemetry, which is never nil and is safe to use and Close even when the error is non-nil
//   - An error if the exporter could not be initialised, in which case the returned Telemetry discards records
func NewTelemetry(
	endpoint, serviceName, version string,
	headers map[string]string,
	environment OtelEnvironment,
	enabled bool,
) (*Telemetry, error) {
	fields := make(map[string]any)
	fields["version"] = version

	// Machine info. ProtectedID is an HMAC of the host id scoped to this application, so the value cannot be
	// correlated with the telemetry of any other program running on the same machine.
	if id, err := machineid.ProtectedID(serviceName); err == nil {
		fields["machine.id"] = strings.ToLower(id)
	}
	fields["machine.os"] = runtime.GOOS
	fields["machine.arch"] = runtime.GOARCH

	// Geolocation is the one enrichment that leaves the machine to be gathered: it discloses the public IP to a
	// third party and blocks the constructor for up to a second. Everything above is read locally, so only this is
	// gated on collection actually being switched on.
	if enabled {
		if geo, err := FetchGeolocation(); err == nil {
			fields["location.country"] = geo.Country
			fields["location.region"] = geo.Region
			fields["location.city"] = geo.City
		}
	}

	cleanup, err := initLogger(endpoint, serviceName, headers, environment, enabled)

	t := &Telemetry{
		logger:  global.GetLoggerProvider().Logger(serviceName),
		base:    fields,
		cleanup: cleanup,
	}
	t.RenewSession()

	return t, err
}

// RenewSession assigns a new session id to every record emitted from now on. It is safe to call concurrently with the
// Log methods.
func (t *Telemetry) RenewSession() {
	fields := maps.Clone(t.base)
	if fields == nil {
		fields = make(map[string]any, 1)
	}
	fields["session.id"] = uuid.New().String()

	attrs := attributesFrom(fields)
	t.attrs.Store(&attrs)
}

// Close flushes any buffered records and shuts the exporter down. It is safe to call on a Telemetry whose construction
// returned an error.
func (t *Telemetry) Close() error {
	if t.cleanup == nil {
		return nil
	}

	return t.cleanup()
}

// region - Private functions

// initLogger installs a global OTLP log exporter. It always returns a usable cleanup function, including on every
// error path, so that a caller which ignores the error cannot end up with a Telemetry that panics on Close.
func initLogger(
	endpoint, serviceName string,
	headers map[string]string,
	environment OtelEnvironment,
	enabled bool,
) (func() error, error) {
	noop := func() error { return nil }

	if !enabled {
		return noop, nil
	}

	ctx := context.Background()

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return noop, err
	}

	exp, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(parsedURL.Host),
		otlploghttp.WithURLPath(parsedURL.Path+"/v1/logs"),
		otlploghttp.WithHeaders(headers),
	)
	if err != nil {
		return noop, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(string(environment)),
		),
	)
	if err != nil {
		return noop, err
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
