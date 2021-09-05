package mesh

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Read with Secrets and ConfigMaps

func (kr *KRun) saveSecretToFile(name string, path string) {
	s, err := kr.Client.CoreV1().Secrets(kr.Namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	for k, v := range s.Data {
		err = ioutil.WriteFile(path+"/"+k, v, 0700)
		if os.Getuid() == 0 {
			_ = os.Chown(path+"/"+k, 1337, 1337)
		}
		if err != nil {
			log.Println("Failed to init secret ", name, path, k, err)
		}
	}
}

func  (kr *KRun) saveConfigMapToFile(name string, path string) {
	s, err := kr.Client.CoreV1().ConfigMaps(kr.Namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	for k, v := range s.Data {
		err = ioutil.WriteFile(path+"/"+k, []byte(v), 0755)
		if err != nil {
			log.Println("Failed to init secret ", name, path, k, err)
		}
	}
}

func (kr *KRun) GetCM(ctx context.Context, ns string, name string) (map[string]string, error) {
	s, err := kr.Client.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return map[string]string{}, err
	}

	return s.Data, nil
}

func (kr *KRun) GetSecret(ctx context.Context, ns string, name string) (map[string][]byte, error) {
	s, err := kr.Client.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return map[string][]byte{}, err
	}

	return s.Data, nil
}

// saveIstioCARoot will create a file with the root certs detected in the cluster.
// This is used for proxyless grpc or cases where envoy is not present, and to allow connection to in-cluster
// Istio.
// Normally this is handled via injection and ProxyConfig-over-XDS - but to connect to XDS we need the cert, which
// is volume mounted in Istio.
func (kr *KRun) saveIstioCARoot(ctx context.Context, prefix string) {
	// TODO: depending on error, move on or report a real error

	cm, err := kr.GetCM(ctx, "istio-system", "istio-ca-root-cert")
	if err != nil {
		log.Println("Istio root not found, citadel compat disabled", err)
	} else {
		// normally mounted to /var/run/secrets/istio
		rootCert := cm["root-cert.pem"]
		if rootCert == "" {
			log.Println("Istio root missing, citadel compat disabled")
		} else {
			kr.CARoots = append(kr.CARoots, rootCert)
			ioutil.WriteFile(prefix+"/var/run/secrets/istio/root-cert.pem", []byte(rootCert), 0755)
		}
	}
}
