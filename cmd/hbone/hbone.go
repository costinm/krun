package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"github.com/costinm/cloud-run-mesh/pkg/gcp"
	"github.com/costinm/cloud-run-mesh/pkg/k8s"
	"github.com/costinm/hbone"
)

var (
	port = flag.String("l", "", "local port")
	tls  = flag.String("tls", "", "Cert dir for mTLS over hbone")
)

// Create a HBONE tunnel to a service.
//
// Will attempt to discover an east-west gateway and get credentials using KUBE_CONFIG or google credentials.
//
// For example:
//
// ssh -o ProxyCommand='hbone %h:22' root@fortio-cr.fortio
//
// If the server doesn't have persistent SSH key, add to the ssh parameters:
//      -F /dev/null -o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null"
//
func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("Expecting service.namespace:port")
	}

	kr := k8s.New()

	kr.VendorInit = gcp.InitGCP

	// Use kubeconfig or gcp to find the cluster
	err := kr.InitK8SClient(context.Background())
	if err != nil {
		log.Fatal("Failed to connect to K8S ", time.Since(kr.StartTime), kr, os.Environ(), err)
	}

	kr.LoadConfig()

	// Not calling RefreshAndSaveFiles - hbone is not creating files, jwts and certs in memory only.
	// Also not initializing pilot-agent or envoy - this is just using k8s to configure the hbone tunnel

	auth, err := hbone.LoadAuth("")
	if err != nil {
		log.Fatal("Failed to find mesh certificates ", err)
	}
	auth.AddRoots([]byte(gcp.MeshCA))


	url := flag.Arg(0)

	// TODO: k8s discovery for hgate
	// TODO: -R to register to the gate, reverse proxy
	// TODO: get certs

	hb := &hbone.HBone{}

	hc := hb.NewEndpoint(url)

	// Initialization done - starting the proxy either on a listener or stdin.

	if *port != "" {
		l, err := net.Listen("tcp", *port)
		if err != nil {
			panic(err)
		}
		for {
			a, err := l.Accept()
			if err != nil {
				panic(err)
			}
			go func() {
				err := hc.Proxy(context.Background(), a, a)
				//err := hbone.HboneCat(http.DefaultClient, url, a, a)
				if err != nil {
					log.Println(err)
				}
			}()
		}
	}

	err = hc.Proxy(context.Background(), os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
