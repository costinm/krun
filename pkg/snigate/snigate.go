package snigate

import (
	context2 "context"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/costinm/hbone"
	"github.com/costinm/krun/pkg/k8s"
	"golang.org/x/net/http2"
)

type SNIGate struct {
	SNIListener net.Listener
	H2RListener net.Listener
	Auth        *hbone.Auth
	HBone       *hbone.HBone
}

type cachedToken struct {
	token string
	expiration time.Time
}

type TokenCache struct {
	cache sync.Map
	kr    *k8s.KRun
}

func (c TokenCache) Token(ctx context2.Context, host string) (string, error) {

	if got, f := c.cache.Load(host); f {
		t := got.(cachedToken)
		if !t.expiration.After(time.Now().Add(-time.Minute)) {
			return t.token, nil
		}
	}

	t, err := c.kr.GetToken(ctx, host)
	// TODO: cache error state or throttle
	if err != nil {
		return "", err
	}
	log.Println("XXX debug Gettoken", host, t)

	c.cache.Store(host, cachedToken{t, time.Now().Add(45 * time.Minute)})
	return t, nil
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

	tcache := &TokenCache{kr: kr}
	h2r.TokenCallback = tcache.Token

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
			// TODO: watcher on Service or ServiceEntry ( k8s or XDS ) to get annotation, allowing service name to be different
		}
		log.Println("Endpoint resolver, h2r not found", parts)

		base := remoteService + ".a.run.app"
		h2c := h2r.NewClient(sni)
		ep := h2c.NewEndpoint("https://" + base + "/_hbone/mtls")
		ep.SNI= base

		return ep
	}
	
	h2r.H2RCallback = func(s string, conn *http2.ClientConn) {
		if s == "" {
			return
		}
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
