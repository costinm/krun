module github.com/costinm/krun/cmd/toktool

go 1.17

replace github.com/costinm/krun => ../..

replace github.com/costinm/krun/third_party => ../../third_party

replace github.com/costinm/hbone => ../../../hbone

require github.com/costinm/krun v0.0.0-00010101000000-000000000000

require (
	cloud.google.com/go v0.97.0 // indirect
	github.com/creack/pty v1.1.13 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.0.0-20211014172544-2b766c08f1c0 // indirect
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
