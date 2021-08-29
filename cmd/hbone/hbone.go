package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/costinm/hbone"
)

var (
	port = flag.String("l", "", "local port")
	tls  = flag.String("tls", "", "Cert dir for mTLS over hbone")
)

// Create a HBONE tunnel to a given URL.
//
// Current client is authenticated for HBONE using local credentials.
//
//
// ssh -o ProxyCommand='hbone https://%h:443/hbone/:22' root@fortio.app.run
// If the server doesn't have persistent SSH key, use
// -F /dev/null -o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null"
// Note the server is still authenticated using the external TLS connection and hostname.
//
func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("Expecting URL or host:port")
	}
	url := flag.Arg(0)

	// TODO: k8s discovery for hgate
	// TODO: -R to register to the gate, reverse proxy
	// TODO: get certs

	hb := &hbone.HBone{}
	hc := hb.NewEndpoint(url)

	if *port != "" {
		fmt.Println("Listening on ", *port, " for ", url)
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

	err := hc.Proxy(context.Background(), os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
