module certtool

go 1.16

replace github.com/costinm/krun => ../..

replace github.com/costinm/krun/k8s/k8s => ../../k8s

replace github.com/costinm/krun/third_party => ../../third_party

replace github.com/costinm/hbone => ../../../hbone

require (
	github.com/costinm/hbone v0.0.0-20211028162624-73e290a5b331
	github.com/costinm/krun v0.0.0-00010101000000-000000000000
	github.com/costinm/krun/third_party v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.26.1
	go.opentelemetry.io/otel v1.1.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.24.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.1.0
	go.opentelemetry.io/otel/sdk v1.1.0
	go.opentelemetry.io/otel/sdk/metric v0.24.0
)

require (
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.26.1
	go.opentelemetry.io/contrib/instrumentation/host v0.26.1
	go.opentelemetry.io/contrib/instrumentation/runtime v0.26.1
	google.golang.org/grpc v1.41.0
)

require (
	cloud.google.com/go/trace v1.0.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.24.0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.0.0
	github.com/costinm/krun/k8s/gcp v0.0.0-20211105170631-bc715687f216
	github.com/costinm/krun/k8s/k8s v0.0.0-20211105170631-bc715687f216
	go.opentelemetry.io/otel/metric v0.24.0
	go.opentelemetry.io/otel/sdk/export/metric v0.24.0
	google.golang.org/api v0.59.0
)
