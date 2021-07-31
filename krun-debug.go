
package main

import (
	"context"
	"log"
	"os"

	"github.com/costinm/cert-ssh/ssh"
	"github.com/costinm/krun/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Optional debug dependency, using cert-based SSH.
func init() {
	initDebug = InitDebug
}

func InitDebug(kr *k8s.KRun) {
	sshCA := os.Getenv("SSHCA")
	extra := os.Getenv("SSH_AUTH")

	sshConfig := os.Getenv("SSH_CONFIG")
	if sshConfig != "" {
		s, err :=  kr.Client.CoreV1().ConfigMaps(kr.Namespace).Get(context.Background(),
			sshConfig, metav1.GetOptions{})
		if err != nil {
			log.Println("failed to load ssh config", sshConfig)
			return
		}
		sshCA = s.Data["SSHCA"]
		extra = s.Data["SSH_AUTH"]
		// TODO: also load private key from secret
	}

	if sshCA == "" && extra == "" {
		return
	}
	err := ssh.StartSSHDWithCA(kr.Namespace, sshCA)
	if err != nil {
		log.Println("Failed to start ssh", err)
	}
}
