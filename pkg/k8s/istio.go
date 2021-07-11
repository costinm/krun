package k8s

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/creack/pty"
)


// When running as root:
// - if /var/lib/istio/resolv.conf is found, use it.
// - else, copy /etc/resolv.conf to /var/lib/istio/resolv.conf and create a new resolv.conf
func resolvConfForRoot()  {
	if _, err := os.Stat("/var/lib/istio/resolv.conf"); !os.IsNotExist(err) {
		log.Println("Alternate resolv.conf exists")
		return
	}

	os.MkdirAll("/var/lib/istio", 0755)
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		log.Println("Failed to read resolv.conf, DNS interception will fail ", err)
		return
	}
	err = os.WriteFile("/var/lib/istio/resolv.conf", data, 0755)
	if err != nil {
		log.Println("Failed to create alternate resolv.conf, DNS interception will fail ", err)
		return
	}
	err = os.WriteFile("/etc/resolv.conf", []byte(`nameserver: 127.0.0.1`), 755)
	if err != nil {
		log.Println("Failed to create resolv.conf, DNS interception will fail ", err)
		return
	}
	log.Println("Adjusted resolv.conf")
}

func (kr *KRun) agentCommand() *exec.Cmd {
	// From the template:

	//- proxy
	//- sidecar
	//- --domain
	//- $(POD_NAMESPACE).svc.{{ .Values.global.proxy.clusterDomain }}
	//- --proxyLogLevel={{ annotation .ObjectMeta `sidecar.istio.io/logLevel` .Values.global.proxy.logLevel }}
	//- --proxyComponentLogLevel={{ annotation .ObjectMeta `sidecar.istio.io/componentLogLevel` .Values.global.proxy.componentLogLevel }}
	//- --log_output_level={{ annotation .ObjectMeta `sidecar.istio.io/agentLogLevel` .Values.global.logging.level }}
	//{{- if .Values.global.sts.servicePort }}
	//- --stsPort={{ .Values.global.sts.servicePort }}
	//{{- end }}
	//{{- if .Values.global.logAsJson }}
	//- --log_as_json
	//{{- end }}
	//{{- if gt .EstimatedConcurrency 0 }}
	//- --concurrency
	//- "{{ .EstimatedConcurrency }}"
	//{{- end -}}
	//{{- if .Values.global.proxy.lifecycle }}
	args := []string{"proxy"}
	if kr.Gateway != "" {
		args = append(args,"router")
	} else {
		args = append(args,"sidecar")
	}
	args = append(args, "--domain")
	args = append(args, kr.Namespace +".svc.cluster.local")
	if kr.AgentDebug != "" {
		args = append(args,	"--log_output_level=" + kr.AgentDebug)
	}
	return exec.Command("/usr/local/bin/pilot-agent", args...)
}

// StartIstioAgent creates the env and starts istio agent.
// If running as root, will also init iptables and change UID to 1337.
func (kr *KRun) StartIstioAgent(proxyConfig string) {
	// /dev/stdout is rejected - it is a pipe.
	// https://github.com/envoyproxy/envoy/issues/8297#issuecomment-620659781
	prefix := "."
	if os.Getuid() == 0 {
		prefix = ""
		resolvConfForRoot()
	}

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

	kr.initLabelsFile()

	env = append(env, "OUTPUT_CERTS="+prefix+"/var/run/secrets/istio.io/")
	env = append(env, "PROXY_CONFIG="+proxyConfig)

	// This would be used if a audience-less JWT was present - not possible with TokenRequest
	// TODO: add support for passing a long lived 1p JWT in a file, for local run
	//env = append(env, "JWT_POLICY=first-party-jwt")

	if os.Getuid() == 0 { //&& kr.Gateway != "" {
		kr.runIptablesSetup(env)
		log.Println("iptables done ", kr.Gateway)
	} else {
		log.Println("No iptables")
	}

	// Currently broken in iptables - use whitebox interception, but still run it
	env = append(env, "ISTIO_META_DNS_CAPTURE=true")
	env = append(env, "DNS_PROXY_ADDR=localhost:53")


	if _, err := os.Stat(prefix + "/var/lib/istio/envoy/envoy_bootstrap_tmpl.json"); os.IsNotExist(err) {
		// TODO: also check real /var/lib - and possibly $ISTIO_SRC/...
		env = append(env, "BOOTSTRAP_XDS_AGENT=true")
	}

	cmd := kr.agentCommand()
	var stdout io.ReadCloser
	if os.Getuid() == 0 {
		os.MkdirAll("/etc/istio/proxy", 777)
		os.Chown("/etc/istio/proxy", 1337, 1337)

		// No longer needed, patched agent to run as root
		//if os.Getenv("K8S_DNS") == "" {
		//	cmd.SysProcAttr = &syscall.SysProcAttr{}
		//	cmd.SysProcAttr.Credential = &syscall.Credential{
		//		Uid: 1337,
		//		Gid: 1337,
		//	}
		//}
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
	os.MkdirAll(prefix+"/var/lib/istio/envoy/", 0700)
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

func (kr *KRun) initLabelsFile() {
	if kr.Gateway != "" {
		ioutil.WriteFile("/etc/istio/pod/labels", []byte(
				`version=v1-cloudrun
security.istio.io/tlsMode="istio"
istio="ingressgateway"
`), 0777)
	} else {
		ioutil.WriteFile("/etc/istio/pod/labels", []byte(fmt.Sprintf(
			`version="v1"
security.istio.io/tlsMode="istio"
app="%s"
service.istio.io/canonical-name="%s"
`, kr.Name, kr.Name)), 0777)
	}
}

func (kr *KRun) runIptablesSetup(env []string) {
	// TODO: make the args the default !
	// pilot-agent istio-iptables -p 15001 -u 1337 -m REDIRECT -i '*' -b "" -x "" -- crash

	//pilot-agent istio-iptables -p 15001 -u 1337 -m REDIRECT -i '10.8.4.0/24' -b "" -x ""
	cmd := exec.Command("/usr/local/bin/pilot-agent",
		"istio-iptables",
		"-p", "15001", // outbound capture port
		//"-z", "15006", - no inbound interception
		"-u", "1337",
		"-m", "REDIRECT",
		"-i", "10.0.0.0/8", // all outbound captured
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

