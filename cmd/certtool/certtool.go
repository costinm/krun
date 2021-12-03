package main

import (
	"context"
	"crypto/x509"
	"flag"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/costinm/hbone"
	"github.com/costinm/hbone/otel"
	"github.com/costinm/krun/pkg/mesh"
	"github.com/costinm/krun/pkg/sts"
	"github.com/costinm/krun/pkg/uk8s"
	"github.com/costinm/krun/third_party/istio/cas"
	"github.com/costinm/krun/third_party/istio/istioca"
	"github.com/costinm/krun/third_party/istio/meshca"
)

var (
	aud      = flag.String("audience", "", "Audience to use in the CSR request")
	provider = flag.String("addr", "", "Address. If empty will use the cluster default. meshca or cas can be used as shortcut")
)

// The tool is also using OTel, to validate the integration.
func main() {
	flag.Parse()
	startCtx := context.Background()

	kr := mesh.New("")

	initOTel(startCtx, kr)
	f := otel.FileExporter(startCtx, os.Stdout)
	defer f()

	_, err := urest.K8SClient(startCtx, kr)
	err = kr.LoadConfig(startCtx)
	if err != nil {
		panic(err)
	}

	// k8s based GSA federated access and ID token provider
	tokenProvider, err := sts.NewSTS(kr)

	if kr.MeshConnectorAddr == "" {
		log.Fatal("Failed to find in-cluster, missing 'hgate' service in mesh env")
	}

	kr.XDSAddr = kr.MeshConnectorAddr + ":15012"

	// Used to generate the CSR
	auth := hbone.NewAuth()
	priv, csr, err := auth.NewCSR("rsa", kr.TrustDomain, "spiffe://"+kr.TrustDomain+"/ns/"+kr.Namespace+"/sa/"+kr.KSA)
	if err != nil {
		log.Fatal("Failed to find mesh certificates ", err)
	}

	// TODO: fetch public keys too - possibly from all

	if *provider == "meshca" {
		InitMeshCA(kr, auth, csr, priv, tokenProvider)
	} else if *provider == "cas" {
		InitCAS(kr, auth, csr, priv, tokenProvider)
	} else {
		InitMeshCert(kr, auth, csr, priv, tokenProvider)
	}
	cert, err := x509.ParseCertificate(auth.Cert.Certificate[0])
	if err != nil {
		panic(err)
	}

	log.Println(cert.URIs, cert.Subject)
	time.Sleep(4 * time.Second)
}

func InitMeshCert(kr *mesh.KRun, auth *hbone.Auth, csr []byte, priv []byte, tokenProvider *sts.STS) {
	if kr.CitadelRoot != "" && kr.MeshConnectorAddr != "" {
		auth.AddRoots([]byte(kr.CitadelRoot))

		cca, err := istioca.NewCitadelClient(&istioca.Options{
			TokenProvider: &mesh.K8SCredentials{KRun: kr, Audience: "istio-ca"},
			CAEndpoint:    kr.MeshConnectorAddr + ":15012",
			TrustedRoots:  auth.TrustedCertPool,
			CAEndpointSAN: "istiod.istio-system.svc",
			GRPCOptions:   OTELGRPCClient(),
		})
		if err != nil {
			log.Fatal(err)
		}
		chain, err := cca.CSRSign(csr, 24*3600)

		//log.Println(chain, err)
		err = auth.SetKeysPEM(priv, chain)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		InitMeshCA(kr, auth, csr, priv, tokenProvider)
	}
}

func InitMeshCA(kr *mesh.KRun, auth *hbone.Auth, csr []byte, priv []byte, tokenProvider *sts.STS) {
	// TODO: Use MeshCA if citadel is not in cluster
	var ol []grpc.DialOption
	ol = append(ol, grpc.WithPerRPCCredentials(tokenProvider))
	ol = append(ol, OTELGRPCClient()...)

	mca, err := meshca.NewGoogleCAClient("meshca.googleapis.com:443",
		ol)
	chain, err := mca.CSRSign(csr, 24*3600)

	if err != nil {
		log.Fatal(err)
	}
	//log.Println(chain, priv)
	err = auth.SetKeysPEM(priv, chain)
	if err != nil {
		log.Fatal(err)
	}

}

// TODO
func InitCAS(kr *mesh.KRun, auth *hbone.Auth, csr []byte, priv []byte, tokenProvider *sts.STS) {
	// TODO: Use MeshCA if citadel is not in cluster

	mca, err := cas.NewGoogleCASClient("projects/"+kr.ProjectId+
		"/locations/"+kr.Region()+"/caPools/istio", tokenProvider)
	chain, err := mca.CSRSign(csr, 24*3600)

	if err != nil {
		log.Fatal(err)
	}
	//log.Println(chain, priv)
	err = auth.SetKeysPEM(priv, chain)
	if err != nil {
		log.Fatal(err)
	}

}
