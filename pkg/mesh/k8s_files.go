package mesh

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Read with Secrets and ConfigMaps

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

