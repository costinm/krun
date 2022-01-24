module github.com/costinm/krun/third_party

go 1.17

replace github.com/costinm/krun/ => ../

replace github.com/costinm/krun/k8s/gcp => ../gcp

replace github.com/costinm/krun/k8s/k8s => ../k8s

require (
	cloud.google.com/go/security v1.1.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	google.golang.org/api v0.59.0
	google.golang.org/genproto v0.0.0-20211021150943-2b146023228c
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
	k8s.io/apimachinery v0.22.2
)

require (
	cloud.google.com/go v0.97.0 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/net v0.0.0-20210520170846-37e1c6afe023 // indirect
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1 // indirect
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/appengine v1.6.7 // indirect
)
