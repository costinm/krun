
package main

import (
	"log"
	"os"

	"github.com/costinm/cert-ssh/ssh"
	"github.com/costinm/krun/pkg/k8s"
)

// Optional debug dependency, using cert-based SSH.
func init() {
	initDebug = InitDebug
}

func InitDebug(kr *k8s.KRun) {
	sshCA := os.Getenv("SSHCA")
	extra := os.Getenv("SSH_AUTH")

	if sshCA == "" && extra == "" {
		return
	}
	err := ssh.StartSSHDWithCA(kr.Namespace, sshCA)
	if err != nil {
		log.Println("Failed to start ssh", err)
	}
}
