module certtool

go 1.17

replace github.com/costinm/krun => ../..

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
	go.opentelemetry.io/otel/metric v0.24.0
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
	cloud.google.com/go v0.97.0 // indirect
	cloud.google.com/go/security v1.1.0 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/creack/pty v1.1.13 // indirect
	github.com/felixge/httpsnoop v1.0.2 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/shirou/gopsutil/v3 v3.21.9 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/otel/internal/metric v0.24.0 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.1.0 // indirect
	golang.org/x/net v0.0.0-20211014172544-2b766c08f1c0 // indirect
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1 // indirect
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/api v0.59.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20211021150943-2b146023228c // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apimachinery v0.22.2 // indirect
)
