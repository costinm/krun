package hbone

import (
	"log"
	"net"
	"net/http"
	"time"
)

// - H2R Server accepts mTLS connection from client, using h2r ALPN
// - Client opens a H2 _server_ handler on the stream, H2R server acts as
// a H2 client.
// - Endpoint is registered in k8s, using IP of the server holding the connection
// - SNI requests on the H2R server are routed to existing connection
// - if a connection is not found on local server, forward based on endpoint.

type H2RServer struct {
	Auth *Auth
	// Non-local endpoints. Key is the 'pod id' of a H2R client
	Endpoints map[string]string

	Local map[string]http.RoundTripper


	h2rListener net.Listener
	sniListener net.Listener
}

func (h2r *H2RServer) Start() error {
	var err error
	h2r.h2rListener, err = net.Listen("tcp", ":14001")
	if err != nil {
		return err
	}
	h2r.sniListener, err = net.Listen("tcp", ":14002")
	if err != nil {
		return err
	}

	go h2r.handleSNI()
	go h2r.handleH2R()

	return nil
}

func (h2r *H2RServer) handleSNI() {
		for {
			remoteConn, err := h2r.sniListener.Accept()
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

			go h2r.handleAcceptedSNI(remoteConn)
		}

	}

func (h2r *H2RServer) handleAcceptedSNI(conn net.Conn) {
	s := &Stream{in: conn}
	sni, err := ParseTLS(s)
	if err != nil {
		conn.Close()
		return
	}
	
	rt := h2r.Local[sni]
	if rt != nil {
		
	}
}

func (h2r *H2RServer) handleH2R() {
	for {
		remoteConn, err := h2r.h2rListener.Accept()
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

		go h2r.handleAcceptedH2R(remoteConn)
	}
	
}

func (h2r *H2RServer) handleAcceptedH2R(conn net.Conn) {
	
}
