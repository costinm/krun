module github.com/costinm/krun/cmd/krun

go 1.16

replace github.com/costinm/krun => ../..

//replace github.com/costinm/cert-ssh => ../../../cert-ssh

require (
	github.com/costinm/hbone v0.0.0-20211014182100-e32b869e6c4b
	github.com/costinm/krun v0.0.0-00010101000000-000000000000
)

require github.com/GoogleCloudPlatform/cloud-run-mesh v0.0.0-20211221010907-547059a93d07
