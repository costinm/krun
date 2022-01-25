module github.com/costinm/krun/gcp

go 1.16

replace github.com/costinm/krun => ../

replace github.com/costinm/krun.k8s/k8s => ../k8s

require (
	cloud.google.com/go v0.97.0
	cloud.google.com/go/container v1.0.0
	cloud.google.com/go/gkehub v0.2.0
	github.com/costinm/krun v0.0.0-00010101000000-000000000000
	google.golang.org/api v0.59.0
	google.golang.org/genproto v0.0.0-20211021150943-2b146023228c
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
)

require github.com/costinm/krun/k8s/k8s v0.0.0-20211105170631-bc715687f216
