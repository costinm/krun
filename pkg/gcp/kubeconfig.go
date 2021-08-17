package gcp

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeconfig "k8s.io/client-go/tools/clientcmd/api"
)

// Utilities around kube config

// SaveKubeConfig saves the KUBECONFIG to ./var/run/.kube/config
// The assumption is that on a read-only image, /var/run will be
// writeable and not backed up.
func SaveKubeConfig(kc *kubeconfig.Config, dir, file string) error {
	cfgjs, err := clientcmd.Write(*kc)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(dir, file), cfgjs, 0744)
	if err != nil {
		return err
	}
	return nil
}

func  restConfig(kc *kubeconfig.Config) (*rest.Config, error) {
	// TODO: set default if not set ?
	return clientcmd.NewNonInteractiveClientConfig(*kc, "", &clientcmd.ConfigOverrides{}, nil).ClientConfig()
}

func MergeKubeConfig(dst *kubeconfig.Config, src *kubeconfig.Config) {
	for k, c := range src.Clusters {
		dst.Clusters[k] = c
	}
	for k, c := range src.Contexts {
		dst.Contexts[k] = c
	}
	for k, c := range src.AuthInfos {
		dst.AuthInfos[k] = c
	}
}
