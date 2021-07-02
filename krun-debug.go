
package main

import (
	"log"

	"github.com/costinm/cert-ssh/ssh"
	"github.com/costinm/krun/pkg/k8s"
)

// Optional debug dependency, using cert-based SSH.
func init() {
	initDebug = InitDebug
}

func InitDebug(kr *k8s.KRun) {
	err := ssh.StartSSHDWithCA(kr.Namespace, kr.SSHCA)
	if err != nil {
		log.Println("Failed to start ssh", err)
	}
}
