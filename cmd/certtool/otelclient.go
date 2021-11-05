package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/costinm/krun/pkg/mesh"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"google.golang.org/grpc"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

// TODO: use otelhttptrace to get httptrace (low level client traces)

func initOTel(ctx context.Context, kr *mesh.KRun) func() {
	kr.TransportWrapper = func(transport http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(transport)
	}
	exp, err := newExporter(os.Stderr)
	if err != nil {
		log.Fatal(err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String("ExampleService"))),
	)
	cleanup := InstallExportPipeline(ctx)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Host telemetry -
	host.Start()

	// End telemetry magic
	return func() {
		cleanup()
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}
}

func OTELGRPCClient() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor())}
}

func newExporter(w io.Writer) (trace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human readable output.
		stdouttrace.WithPrettyPrint(),
		// Do not print timestamps for the demo.
		//stdouttrace.WithoutTimestamps(),
	)
}

func InstallExportPipeline(ctx context.Context) func() {
	exporter, err := stdoutmetric.New(
		//stdoutmetric.WithPrettyPrint(),
		stdoutmetric.WithWriter(os.Stderr))
	if err != nil {
		log.Fatalf("creating stdoutmetric exporter: %v", err)
	}

	pusher := controller.New(
		processor.NewFactory(
			simple.NewWithInexpensiveDistribution(),
			exporter,
			processor.WithMemory(true),
		),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(3*time.Second),
		// WithResource, WithCollectPeriod, WithPushTimeout
	)

	if err = pusher.Start(ctx); err != nil {
		log.Fatalf("starting push controller: %v", err)
	}

	global.SetMeterProvider(pusher)

	if err := runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	); err != nil {
		log.Fatalln("failed to start runtime instrumentation:", err)
	}

	return func() {
		if err := pusher.Stop(ctx); err != nil {
			log.Fatalf("stopping push controller: %v", err)
		}
	}
}
