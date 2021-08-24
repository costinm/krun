package snigate

import (
	context2 "context"
	"log"
	"net"
	"strings"

	"github.com/costinm/hbone"
	"github.com/costinm/krun/pkg/k8s"
	"golang.org/x/net/context"
	"golang.org/x/net/http2"
)

type SNIGate struct {
	SNIListener net.Listener
	H2RListener net.Listener
	Auth        *hbone.Auth
	HBone       *hbone.HBone
}

func InitSNIGate(kr *k8s.KRun, sniPort string, h2rPort string) (*SNIGate, error) {

	err := kr.InitK8SClient(context2.Background())
	if err != nil {
		return nil, err
	}

	kr.LoadConfig()

	kr.Refresh() // create the tokens expected for Istio

	if kr.Gateway == "" {
		kr.Gateway = "hgate"
	}

	err = kr.StartIstioAgent()
	if err != nil {
		log.Fatal("Failed to start istio agent and envoy", err)
	}

	auth, err := hbone.LoadAuth(kr.BaseDir + "var/run/secrets/istio.io/")
	if err != nil {
		return nil, err
	}

	h2r := hbone.New(auth)

	h2r.TokenCallback = func(ctx context.Context, host string) (string, error) {
		log.Println("Gettoken", host)
		// TODO: P0 cache the token !!!!
		return kr.GetToken(ctx, host)
	}

	h2r.EndpointResolver = func(sni string) *hbone.Endpoint {
		// Current Istio SNI looks like:
		//
		// outbound_.9090_._.prometheus-1-prometheus.mon.svc.cluster.local
		// We need to map it to a cloudrun external address, add token based on the audience, and make the call using
		// the tunnel.
		//
		// Also supports the 'natural' form

		//
		//
		parts := strings.Split(sni, ".")
		remoteService := parts[0]
		if parts[0] == "outbound_" {
			remoteService = parts[3]
			// TODO: extract 'version' from URL, convert it to cloudrun revision ?
		}
		log.Println("Endpoint resolver, h2r not found", parts)

		base := remoteService + ".a.run.app"
		h2c := h2r.NewClient(sni)
		ep := h2c.NewEndpoint("https://" + base + "/_hbone/mtls")
		ep.SNI= base

		return ep
	}
	
	h2r.H2RCallback = func(s string, conn *http2.ClientConn) {
		log.Println("H2R connection event", s, conn)
		// TODO: save a WorkloadInstance of EndpontSlice

	}

	sniL, err := hbone.ListenAndServeTCP(sniPort, h2r.HandleSNIConn)
	if err != nil {
		return nil, err
	}

	h2rL, err := hbone.ListenAndServeTCP(h2rPort, h2r.HandlerH2RConn)
	if err != nil {
		return nil, err
	}

	return &SNIGate{
		SNIListener: sniL,
		H2RListener: h2rL,
		Auth: auth,
		HBone: h2r,
	}, nil


}
