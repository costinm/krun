package main

import (
	"context"
	"log"

	"github.com/costinm/cert-ssh/ssh"
	"github.com/costinm/cloud-run-mesh/pkg/mesh"
)

// Optional debug dependency, using cert-based SSH or loaded from a secret.
// TODO: add conditional compilation, or move it to a separate binary that can be forked

func init() {
	initDebug = InitDebug
}

func InitDebug(kr *mesh.KRun) {
	sshCM, err := kr.GetSecret(context.Background(), kr.Namespace, "sshdebug")
	if err != nil {
		log.Println("SSH config error", err)
	}
	ssh.InitFromSecret(sshCM, kr.Namespace)
}
