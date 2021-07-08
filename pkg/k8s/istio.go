package k8s

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

// StartIstioAgent creates the env and starts istio agent.
// If running as root, will also init iptables and change UID to 1337.
func (kr *KRun) StartIstioAgent(ns string, proxyConfig string, prefix string) {
	// /dev/stdout is rejected - it is a pipe.
	// https://github.com/envoyproxy/envoy/issues/8297#issuecomment-620659781

	env := os.Environ()
	// XDS and CA servers are using system certificates ( recommended ).
	// If using a private CA - add it's root to the docker images, everything will be consistent
	// and simpler !
	env = append(env, "XDS_ROOT_CA=SYSTEM")
	env = append(env, "CA_ROOT_CA=SYSTEM")

	os.MkdirAll(prefix + "/etc/istio/proxy", 0755)

	// Save the istio certificates - for proxyless or app use.
	os.MkdirAll(prefix + "/var/run/secrets/istio.io", 0755)
	os.MkdirAll(prefix + "/etc/istio/pod", 0755)
	if os.Getuid() == 0 {
		os.Chown(prefix + "/var/run/secrets/istio.io", 1337, 1337)
		os.Chown(prefix + "/etc/istio/pod", 1337, 1337)
		os.Chown(prefix + "/etc/istio/proxy", 1337, 1337)
	}
	ioutil.WriteFile("/etc/istio/pod/labels", []byte(fmt.Sprintf(`version="v1"
security.istio.io/tlsMode="istio"
app="%s"
service.istio.io/canonical-name="%s"
`, kr.Name, kr.Name)), 0777)

	env = append(env, "OUTPUT_CERTS=" + prefix + "/var/run/secrets/istio.io/")

	// This would be used if a audience-less JWT was present - not possible with TokenRequest
	// TODO: add support for passing a long lived 1p JWT in a file, for local run
	//env = append(env, "JWT_POLICY=first-party-jwt")

	if os.Getuid() == 0 { // && kr.Gateway != "" {
		// TODO: make the args the default !
		// pilot-agent istio-iptables -p 15001 -u 1337 -m REDIRECT -i '*' -b "" -x "" -- crash

		//pilot-agent istio-iptables -p 15001 -u 1337 -m REDIRECT -i '10.8.4.0/24' -b "" -x ""
		cmd := exec.Command("/usr/local/bin/pilot-agent",
			"istio-iptables",
			"-p", "15001", // outbound capture port
			//"-z", "15006", - no inbound interception
		  "-u", "1337",
		  "-m", "REDIRECT",
		  "-i", "10.8.4.0/24", // all outbound captured
		  "-b", "", // disable all inbound redirection
		  // "-d", "15090,15021,15020", // exclude specific ports
		  "-x", "")
		cmd.Env = env
		cmd.Dir = "/"
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			log.Println("Error starting iptables", err)
		} else {
			err = cmd.Wait()
			if err != nil {
				log.Println("Error starting iptables", err)
			}
		}
		log.Println("Iptables start done")
	}

	env = append(env, "ISTIO_META_DNS_CAPTURE=true")


	env = append(env, "PROXY_CONFIG=" + proxyConfig)

	if _, err := os.Stat(prefix + "/var/lib/istio/envoy/envoy_bootstrap_tmpl.json"); os.IsNotExist(err) {
		// TODO: also check real /var/lib - and possibly $ISTIO_SRC/...
		env = append(env, "BOOTSTRAP_XDS_AGENT=true")
	}


	var cmd *exec.Cmd
	if kr.Gateway != "" {
		ioutil.WriteFile("/etc/istio/pod/labels", []byte(`version=v1-cloudrun
security.istio.io/tlsMode="istio"
istio="ingressgateway"
`), 0777)
		cmd = exec.Command("/usr/local/bin/pilot-agent", "proxy", "router", "--domain", ns+".svc.cluster.local")
	} else {
		cmd = exec.Command("/usr/local/bin/pilot-agent", "proxy", "sidecar", "--domain", ns+".svc.cluster.local")
	}
	var stdout io.ReadCloser
	if os.Getuid() == 0 {
		os.MkdirAll("/etc/istio/proxy", 777)
		os.Chown("/etc/istio/proxy", 1337, 1337)

		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Credential = &syscall.Credential{
			Uid: 1337,
			Gid: 1337,
		}
		//cmd.SysProcAttr.Setsid = true
		//cmd.SysProcAttr.Setctty = true
		pty, tty, err := pty.Open()
		if err != nil {
			log.Println("Error opening pty ", err)
			stdout, _ = cmd.StdoutPipe()
			os.Stdout.Chown(1337, 1337)
		} else {
			cmd.Stdout = tty
			err = tty.Chown(1337, 1337)
			if err != nil {
				log.Println("Error chown ", tty.Name(), err)
			} else {
				log.Println("Opened pyy", tty.Name(), pty.Name())
			}
			stdout = pty
		}
		cmd.Dir = "/"
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Env = env

	cmd.Stderr = os.Stderr
	os.MkdirAll(prefix + "/var/lib/istio/envoy/", 0700)
	go func() {
		log.Println("Starting istio agent with ", cmd.Env)
		err := cmd.Start()
		if err != nil {
			log.Println("Failed to start ", cmd, err)
		}
		if stdout != nil {
			go func() {
				io.Copy(os.Stdout, stdout)
			}()
		}
		err = cmd.Wait()
		if err != nil {
			log.Println("Wait err ", err)
		}

		os.Exit(0)
	}()


	// TODO: wait for agent to be ready
}

