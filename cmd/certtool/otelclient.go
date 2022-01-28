package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-mesh/pkg/mesh"
	"github.com/GoogleCloudPlatform/cloud-run-mesh/pkg/sts"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"

	//mexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// TODO: use otelhttptrace to get httptrace (low level client traces)

func initOTel(ctx context.Context, kr *mesh.KRun) (func(), error) {
	kr.TransportWrapper = func(transport http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(transport)
	}

	r := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.ServiceNameKey.String(kr.Name))

	var exp trace.SpanExporter
	var err error
	if true {
		// k8s based GSA federated access and ID token provider
		tokenProvider, _ := sts.NewSTS(kr)
		tokenProvider.MDPSA = true
		tokenProvider.UseAccessToken = true

		exp, err = cloudtrace.New(cloudtrace.WithProjectID(kr.ProjectId),
			cloudtrace.WithTraceClientOptions([]option.ClientOption{
				option.WithGRPCDialOption(grpc.WithPerRPCCredentials(tokenProvider)),
				option.WithQuotaProject(kr.ProjectId),
			}))
		//_, shutdown, err := texporter.InstallNewPipeline(
		//	[]texporter.Option {
		//		// optional exporter options
		//	},
		//	// This example code uses sdktrace.AlwaysSample sampler to sample all traces.
		//	// In a production environment or high QPS setup please use ProbabilitySampler
		//	// set at the desired probability.
		//	// Example:
		//	// sdktrace.WithConfig(sdktrace.Config {
		//	//     DefaultSampler: sdktrace.ProbabilitySampler(0.0001),
		//	// })
		//	trace.WithConfig(trace.Config{
		//		DefaultSampler: trace.AlwaysSample(),
		//	}),
		//	// other optional provider options
		//)
	} else {
		exp, err = stdouttrace.New(
			stdouttrace.WithWriter(os.Stderr),
			// Use human readable output.
			stdouttrace.WithPrettyPrint(),
			// Do not print timestamps for the demo.
			//stdouttrace.WithoutTimestamps(),
		)
	}
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(r),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// =========== Metrics
	var exporter metric.Exporter
	if true {
		// k8s based GSA federated access and ID token provider
		tokenProvider, _ := sts.NewSTS(kr)
		tokenProvider.MDPSA = true
		tokenProvider.UseAccessToken = true

		//exporter, err = mexporter.NewRawExporter(mexporter.WithProjectID(kr.ProjectId),
		//	mexporter.WithMonitoringClientOptions(
		//		option.WithGRPCDialOption(grpc.WithPerRPCCredentials(tokenProvider)),
		//		option.WithQuotaProject(kr.ProjectId)))

	} else {
		exporter, err = stdoutmetric.New(
			//stdoutmetric.WithPrettyPrint(),
			stdoutmetric.WithWriter(os.Stderr))
	}
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
		controller.WithResource(r),
		// WithResource, WithCollectPeriod, WithPushTimeout
	)

	if err = pusher.Start(ctx); err != nil {
		log.Fatalf("starting push controller: %v", err)
	}

	global.SetMeterProvider(pusher)

	// Global instrumentations
	if err := runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	); err != nil {
		log.Fatalln("failed to start runtime instrumentation:", err)
	}
	// Host telemetry -
	host.Start()

	// End telemetry magic
	return func() {
		if err := pusher.Stop(ctx); err != nil {
			log.Fatalf("stopping push controller: %v", err)
		}
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}, nil
	/*
		kr.TransportWrapper = func(transport http.RoundTripper) http.RoundTripper {
			return otelhttp.NewTransport(transport)
		}
		// Host telemetry -
		host.Start()
	*/
}

func OTELGRPCClient() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor())}
}
