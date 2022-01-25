module github.com/costinm/krun/k8s

go 1.16

replace github.com/costinm/krun => ../

require (
	github.com/costinm/krun v0.0.0-00010101000000-000000000000
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/klog v1.0.0
)

require golang.org/x/text v0.3.7 // indirect
