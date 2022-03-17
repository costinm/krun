module github.com/costinm/krun

go 1.16

replace github.com/costinm/krun/pkg/urest => ./pkg/urest

require (
	github.com/GoogleCloudPlatform/cloud-run-mesh v0.0.0-20220128230121-cac57262761b
	google.golang.org/protobuf v1.27.1 // indirect
)
