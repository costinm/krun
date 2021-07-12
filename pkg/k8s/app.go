package k8s

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// StartApp uses the reminder of the command line to exec an app, using K8S_UID as UID, if present.
func (kr *KRun) StartApp() {
	var cmd *exec.Cmd
	if len(os.Args) == 1 {
		return
	} else if len(os.Args) == 2 {
		cmd = exec.Command(os.Args[1])
	} else {
		cmd = exec.Command(os.Args[1], os.Args[2:]...)
	}
	if os.Getuid() == 0 {
		uid := os.Getenv("K8S_UID")
		if uid != "" {
			uidi, err := strconv.Atoi(uid)
			if err == nil {
				cmd.SysProcAttr = &syscall.SysProcAttr{}
				cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uidi)}
			}
		}
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set port to 8080 - some apps use the PORT from knative to start.a
	cmd.Env = []string{"PORT=8080"}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e,"PORT=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	go func() {
		err := cmd.Start()
		if err != nil {
			log.Println("Failed to start ", cmd, err)
		}
		kr.appCmd = cmd
		err = cmd.Wait()
		if err != nil {
			log.Println("Failed to wait ", cmd, err)
		}
		kr.Exit(0)
	}()
}


