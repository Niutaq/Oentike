package telemetry

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitProvider initializes an OTLP exporter, and configures the corresponding trace provider.
func InitProvider(ctx context.Context, endpoint string, serviceName string) (*sdktrace.TracerProvider, error) {
	if endpoint == "" {
		return nil, nil // Local development does not require a collector.
	}
	options := []otlptracegrpc.Option{}
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, fmt.Errorf("invalid OTLP endpoint %q", endpoint)
		}
		options = append(options, otlptracegrpc.WithEndpoint(parsed.Host))
		if parsed.Scheme == "http" {
			options = append(options, otlptracegrpc.WithInsecure())
		}
	} else {
		options = append(options, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	}

	exp, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	log.Printf("OTEL Trace provider initialized with endpoint: %s", endpoint)
	return tp, nil
}
