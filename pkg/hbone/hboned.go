package hbone

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime/debug"
	"time"

	"golang.org/x/net/http2"
)

type HBone struct {
	Auth *Auth

	h2Server *http2.Server
	listener net.Listener
	Cert     *tls.Certificate
	rp       *httputil.ReverseProxy
}

type HBoneAcceptedConn struct {
	hb   *HBone
	conn net.Conn
}

// StartBHoneD will listen on addr as H2C (typically :15009)
//
//
// Incoming streams for /_hbone/mtls will be treated as a mTLS connection,
// using the Istio certificates and root. After handling mTLS, the clear text
// connection will be forwarded to localhost:8080 ( TODO: custom port ).
//
// TODO: setting for app protocol=h2, http, tcp - initial impl uses tcp
//
// Incoming requests for /_hbone/22 will be forwarded to localhost:22, for
// debugging with ssh.
//
//
func (hb *HBone) Init() error {
	if hb.Auth == nil {
		hb.Auth = &Auth{
		}
	}
	err := hb.Auth.InitKeys()
	if err != nil {
		return err
	}
	hb.h2Server = &http2.Server{}
	u, _ := url.Parse("http://localhost:8080")
	hb.rp = httputil.NewSingleHostReverseProxy(u)

	return nil
}

// Start the HBone server. Must be called after envoy and the app are ready.
func (hb *HBone) Start(addr string) error {
	var err error
	hb.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go hb.serve()

	return nil
}

func (hb *HBone) serve() {
	for {
		remoteConn, err := hb.listener.Accept()
		if ne, ok := err.(net.Error); ok {
			if ne.Temporary() {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}
		if err != nil {
			log.Println("Accept error, closing listener ", err)
			return
		}

		go hb.handleAcceptedH2C(remoteConn)
	}
}

func (hac *HBoneAcceptedConn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	defer func() {
		log.Println("Hbone", "", "", r, time.Since(t0))

		if r := recover(); r != nil {
			fmt.Println("Recovered in hbone", r)

			debug.PrintStack()

			// find out exactly what the error was and set err
			var err error

			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic")
			}
			if err != nil {
				fmt.Println("ERRROR: ", err)
			}
		}
	}()

	// TODO: parse Envoy / hbone headers.

	// TCP proxy for SSH ( no mTLS, SSH has its own equivalent)
	if r.RequestURI ==  "/_hbone/22" {
		err := hac.hb.HandleTCPProxy(w, r.Body, "localhost:15022")
		log.Println("hbone proxy done ", r.RequestURI, err)

		return
	}
	if r.RequestURI ==  "/_hbone/mtls" {
		// Create a stream, used for proxy with caching.
		conf := hac.hb.Auth.TLSConfig

		tls := tls.Server(&HTTPConn{r: r, w: w, acceptedConn: hac.conn}, conf)

		// TODO: replace with handshake with context
		err := tls.Handshake()
		if err != nil {
			return
		}

		// TODO: All Istio checks go here. The TLS handshake doesn't check
		// root cert or anything - this is proof of concept only, to eval
		// perf.

		// TODO: allow user to customize app port, protocol.
		// TODO: if protocol is not matching wire protocol, convert.
		hac.hb.HandleTCPProxy(tls, tls, "locahost:8080")
		//if tls.ConnectionState().NegotiatedProtocol == "h2" {
		//	// http2 and http expect a net.Listener, and do their own accept()
		//	hb.proxy.ServeConn(
		//		tls,
		//		&http2.ServeConnOpts{
		//			Handler: http.HandlerFunc(l.ug.H2Handler.httpHandleHboneCHTTP),
		//			Context: tc.Context(), // associated with the stream, with cancel
		//		})
		//} else {
		//	// HTTP/1.1
		//	// TODO. Typically we want to upgrade over the wire to H2
		//}
		return
	}

	// This is not a tunnel, but regular request. For test only - should be off once mTLS
	// works properly.
	hac.hb.rp.ServeHTTP(w, r)
}


func (hb *HBone) handleAcceptedH2C(conn net.Conn) {
	hc := &HBoneAcceptedConn{hb: hb, conn: conn}
	hb.h2Server.ServeConn(
		conn,
		&http2.ServeConnOpts{
			Handler: hc,                   // Also plain text, needs to be upgraded
			Context: context.Background(),
			//Context can be used to cancel, pass meta.
			// h2 adds http.LocalAddrContextKey(NetAddr), ServerContextKey (*Server)
		})
}

func (hb *HBone) HandleTCPProxy(w io.Writer, r io.Reader, s string) error {
	nc, err := net.Dial("tcp", s)
	if err != nil {
		log.Println("Error dialing ", s ,err)
		return err
	}

	errCh := make(chan error, 2)
	go hb.proxyFromClient(nc, r, errCh)

	_, err = CopyBuffered(w, nc)

	remoteErr := <-errCh
	if remoteErr != nil && remoteErr != io.EOF {
		return remoteErr
	}
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func (hb *HBone) proxyFromClient(w io.Writer, r io.Reader, ch chan error) {
	_, err := CopyBuffered(w, r)
	ch <- err
}



