package hbone

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Auth struct {
	CertDir   string
	Cert      *tls.Certificate
	TLSConfig *tls.Config

	// Namespace and SA are extracted from the certificate.
	Namespace       string
	SA              string
	TrustDomain     string

	// Trusted roots
	// TODO: copy Istiod multiple trust domains code. This will be a map[trustDomain]roots and a
	// list of TrustDomains. XDS will return the info via ProxyConfig.
	// This can also be done by krun - loading a config map with same info.
	TrustedCertPool *x509.CertPool
}

// TODO: ./etc/certs support: krun should copy the files, for consistency (simper code for frameworks).
// TODO: periodic reload

func (hb *Auth) InitKeys() error {
	if hb.CertDir == "" {
		hb.CertDir = "./var/run/secrets/istio.io/"
	}
	keyFile := filepath.Join(hb.CertDir, "key.pem")
	err := WaitFile(keyFile, 5 * time.Second)
	if err != nil {
		return err
	}

	keyBytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return err
	}
	certBytes, err := ioutil.ReadFile(filepath.Join(hb.CertDir, "cert-chain.pem"))
	if err != nil {
		return err
	}
	tlsCert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return err
	}
	hb.Cert = &tlsCert
	if tlsCert.Certificate == nil || len(tlsCert.Certificate) == 0 {
		return errors.New("missing certificate")
	}

	hb.TrustedCertPool = x509.NewCertPool()
	// TODO: multiple roots
	rootCert, _ := ioutil.ReadFile(filepath.Join(hb.CertDir, "root-cert.pem"))
	if rootCert != nil {
		block, rest := pem.Decode(rootCert)
		var blockBytes []byte
		for block != nil {
			blockBytes = append(blockBytes, block.Bytes...)
			block, rest = pem.Decode(rest)
		}

		rootCAs, err := x509.ParseCertificates(blockBytes)
		if err != nil {
			return err
		}
		for _, c := range rootCAs {
			log.Println("Adding root CA: ", c.Subject)
			hb.TrustedCertPool.AddCert(c)
		}
	}

	cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return err
	}
	if len(cert.URIs) > 0 {
		c0 := cert.URIs[0]
		pathComponetns := strings.Split(c0.Path, "/")
		if c0.Scheme == "spiffe" && pathComponetns[1] == "ns" && pathComponetns[3] == "sa" {
			hb.Namespace = pathComponetns[2]
			hb.SA = pathComponetns[4]
			hb.TrustDomain = cert.URIs[0].Host
		} else {
			//log.Println("Cert: ", cert)
			// TODO: extract domain, ns, name
			log.Println("Unexpected ID ", c0, cert.Issuer, cert.NotAfter)
		}
		//log.Println("Cert: ", cert)
		// TODO: extract domain, ns, name
		log.Println("ID ", c0, cert.Issuer, cert.NotAfter)
	} else {
		// org and name are set
		log.Println("Cert: ", cert.Subject.Organization, cert.NotAfter)
	}

	hb.TLSConfig = &tls.Config{
		//MinVersion: tls.VersionTLS13,
		//PreferServerCipherSuites: ugate.preferServerCipherSuites(),
		InsecureSkipVerify: true,                  // This is not insecure here. We will verify the cert chain ourselves.
		ClientAuth:         tls.RequestClientCert, // not require - we'll fallback to JWT

		Certificates:       []tls.Certificate{*hb.Cert}, // a.TlsCerts,

		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("client certificate required")
			}
			var peerCert *x509.Certificate
			intCertPool := x509.NewCertPool()
			for id, rawCert := range rawCerts {
				cert, err := x509.ParseCertificate(rawCert)
				if err != nil {
					return err
				}
				if id == 0 {
					peerCert = cert
				} else {
					intCertPool.AddCert(cert)
				}
			}
			if peerCert == nil || len(peerCert.URIs) == 0 {
				return errors.New("peer certificate does not contain URI type SAN")
			}
			trustDomain := peerCert.URIs[0].Host
			if trustDomain != hb.TrustDomain {
				return errors.New("invalid trust domain")
			}

			_, err = peerCert.Verify(x509.VerifyOptions{
				Roots:         hb.TrustedCertPool,
				Intermediates: intCertPool,
			})
			return err
		},
		NextProtos: []string{"istio", "h2"},
		GetCertificate: func(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return hb.Cert, nil
		},
	}

	return nil
}

// WaitFile will check for the file to show up - the agent is running in a separate process.
func WaitFile(keyFile string, d time.Duration) error {
	t0 := time.Now()
	var err error
	for {
		// Wait for key file to show up - pilot agent creates it.
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			if time.Since(t0) > d {
				return err
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		return nil
	}

	return err
}
