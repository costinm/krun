module certtool

go 1.17

replace github.com/costinm/krun => ../..
replace github.com/costinm/krun/gcp => ../../gcp

replace github.com/costinm/krun/k8s => ../../k8s

replace github.com/costinm/krun/third_party => ../../third_party

replace github.com/costinm/hbone => ../../../hbone

replace github.com/costinm/hbone/otel => ../../../hbone/otel

require (
	github.com/costinm/hbone v0.0.0-20211028162624-73e290a5b331
	github.com/costinm/krun v0.0.0-00010101000000-000000000000
	github.com/costinm/krun/third_party v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.26.1
	go.opentelemetry.io/otel v1.1.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v0.24.0
	go.opentelemetry.io/otel/sdk v1.1.0
	go.opentelemetry.io/otel/sdk/metric v0.24.0
)

require (
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.26.1
	go.opentelemetry.io/contrib/instrumentation/host v0.26.1
	google.golang.org/grpc v1.41.0
)

require (
	cloud.google.com/go/trace v1.0.0 // indirect
	github.com/GoogleCloudPlatform/cloud-run-mesh v0.0.0-20211221010907-547059a93d07 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.24.0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.0.0
	github.com/costinm/krun/gcp v0.0.0-20220124172154-1a9be088ad1e
	github.com/costinm/krun/k8s/k8s v0.0.0-20211105170631-bc715687f216
	go.opentelemetry.io/otel/metric v0.24.0
	go.opentelemetry.io/otel/sdk/export/metric v0.24.0
	google.golang.org/api v0.59.0
)

require (
	cloud.google.com/go v0.97.0 // indirect
	cloud.google.com/go/container v1.0.0 // indirect
	cloud.google.com/go/gkehub v0.2.0 // indirect
	cloud.google.com/go/monitoring v0.1.0 // indirect
	cloud.google.com/go/security v1.1.0 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/creack/pty v1.1.13 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.2 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/shirou/gopsutil/v3 v3.21.9 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/otel/internal/metric v0.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.1.0 // indirect
	golang.org/x/net v0.0.0-20211014172544-2b766c08f1c0 // indirect
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1 // indirect
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac // indirect
	golang.org/x/term v0.0.0-20210220032956-6a3ed077a48d // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20211021150943-2b146023228c // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.22.2 // indirect
	k8s.io/apimachinery v0.22.2 // indirect
	k8s.io/client-go v0.22.2 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.9.0 // indirect
	k8s.io/utils v0.0.0-20210819203725-bdf08cb9a70a // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)
