package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds telemetry configuration.
type Config struct {
	// ServiceName is the name of the service for telemetry identification.
	ServiceName string
	// ServiceVersion is the version/tag of the service.
	ServiceVersion string
	// Environment is the deployment environment (development, staging, production).
	Environment string
	// ExporterType is the type of exporter to use ("otlp" or "stdout").
	ExporterType string
	// OTLPEndpoint is the OTLP collector endpoint (e.g., "localhost:4318").
	OTLPEndpoint string
	// Enabled controls whether telemetry is active.
	Enabled bool
}

// Provider holds initialized telemetry providers and offers a clean shutdown.
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

// Setup initializes OpenTelemetry tracing and metrics providers.
// Returns a Provider that must be shut down via Shutdown() on application exit.
func Setup(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{}, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	tp, err := initTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("initializing tracer provider: %w", err)
	}
	otel.SetTracerProvider(tp)

	mp, err := initMeterProvider(ctx, cfg, res)
	if err != nil {
		// Clean up tracer provider if meter provider fails.
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("initializing meter provider: %w", err)
	}
	otel.SetMeterProvider(mp)

	return &Provider{
		tracerProvider: tp,
		meterProvider:  mp,
	}, nil
}

// Shutdown gracefully shuts down all telemetry providers, flushing any pending data.
func (p *Provider) Shutdown(ctx context.Context) error {
	var firstErr error

	if p.tracerProvider != nil {
		if err := p.tracerProvider.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("shutting down tracer provider: %w", err)
		}
	}

	if p.meterProvider != nil {
		if err := p.meterProvider.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("shutting down meter provider: %w", err)
		}
	}

	return firstErr
}

// Tracer returns a named tracer from the global TracerProvider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// Meter returns a named meter from the global MeterProvider.
func Meter(name string) metric.Meter {
	return otel.Meter(name)
}

func initTracerProvider(ctx context.Context, cfg Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error

	switch cfg.ExporterType {
	case "otlp":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.Environment == "development" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
	default:
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	if err != nil {
		return nil, fmt.Errorf("creating trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithResource(res),
	)

	return tp, nil
}

func initMeterProvider(ctx context.Context, cfg Config, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	var exporter sdkmetric.Exporter
	var err error

	switch cfg.ExporterType {
	case "otlp":
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.Environment == "development" {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
	default:
		exporter, err = stdoutmetric.New()
	}

	if err != nil {
		return nil, fmt.Errorf("creating metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(30*time.Second))),
		sdkmetric.WithResource(res),
	)

	return mp, nil
}