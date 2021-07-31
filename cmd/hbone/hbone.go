package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/costinm/krun/pkg/hbone"
)


var (
	port = flag.String("l", "", "local port")
	tls = flag.String("tls", "", "Cert dir for mTLS over hbone")
)


// Create a HBONE tunnel to a given URL.
//
// Current client is authenticated for HBONE using local credentials,
// or a kube.json file. If no certs or kube.json is found, one will be generated.
//
// Example:
// ssh -v -o ProxyCommand='hbone https://c1.webinf.info:443/dm/PZ5LWHIYFLSUZB7VHNAMGJICH7YVRU2CNFRT4TXFFQSXEITCJUCQ:22'  root@PZ5LWHIYFLSUZB7VHNAMGJICH7YVRU2CNFRT4TXFFQSXEITCJUCQ
// ssh -v -o ProxyCommand='hbone https://%h:443/hbone/:22' root@fortio.app.run
//
// Note that SSH is converting %h to lowercase - the ID must be in this form
//
func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("Expecting URL or host:port")
	}
	url := flag.Arg(0)

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
				err := hbone.HboneCat(http.DefaultClient, url, a, a)
				if err != nil {
					log.Println(err)
				}
			}()
		}
	}

	fmt.Println("Connecting to ", url)
	err := hbone.HboneCat(http.DefaultClient, url, os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}


