package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


// MeshConfig is a minimal mesh config.
type MeshConfig struct {
	TrustDomain string `yaml:"trustDomain,omitempty"`
	DefaultConfig ProxyConfig `yaml:"defaultConfig,omitempty"`
}
type ProxyConfig struct {
	DiscoveryAddress string `yaml:"discoveryAddress,omitempty"`
	MeshId string `yaml:"meshId,omitempty"`
	ProxyMetadata map[string]string `yaml:"proxyMetadata,omitempty"`
}

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

// FindXDSAddr will try to find the XDSAddr using in-cluster info.
// This is called after K8S client has been initialized.
func (kr *KRun) FindXDSAddr() error {
	// TODO: find default tag, label, etc.
	// Current code is written for MCP, use XDS_ADDR explicitly
	// otherwise.
	s, err :=  kr.Client.CoreV1().ConfigMaps("istio-system").Get(context.Background(),
		"istio-asm-managed", metav1.GetOptions{})
	if err != nil {
		return err
	}
	meshCfg := s.Data["mesh"]
	mc := MeshConfig{}
	err = yaml.Unmarshal([]byte(meshCfg), &mc)
	if err != nil {
		return err
	}

	kr.TrustDomain = mc.TrustDomain
	if kr.ProjectId == "" {
		td := strings.Split(kr.TrustDomain, ".")
		if len(td) > 1 {
			kr.ProjectId = td[0]
		}
	}
	kr.XDSAddr = mc.DefaultConfig.DiscoveryAddress
	kr.MCPAddr = mc.DefaultConfig.ProxyMetadata[ "ISTIO_META_CLOUDRUN_ADDR"]
	meshId := mc.DefaultConfig.MeshId
	mid := strings.Split(meshId, "-")
	if len(mid) > 1 {
		kr.ProjectNumber = mid[1]
	}

	return nil
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
	args = append(args, "--serviceCluster")
	args = append(args, kr.Name + "." + kr.Namespace)

	if kr.AgentDebug != "" {
		args = append(args,	"--log_output_level=" + kr.AgentDebug)
	}
	args = append(args, "--stsPort=15463")
	return exec.Command("/usr/local/bin/pilot-agent", args...)
}

// StartIstioAgent creates the env and starts istio agent.
// If running as root, will also init iptables and change UID to 1337.
func (kr *KRun) StartIstioAgent() error {
	if kr.XDSAddr == "-" {
		return nil
	}
	if kr.XDSAddr == "" {
		err := kr.FindXDSAddr()
		if err != nil {
			return err
		}
	}
	proxyConfig := fmt.Sprintf(`{"discoveryAddress": "%s"}`, kr.XDSAddr)
	// /dev/stdout is rejected - it is a pipe.
	// https://github.com/envoyproxy/envoy/issues/8297#issuecomment-620659781
	prefix := "."
	if os.Getuid() == 0 {
		prefix = ""
	}

	if kr.Name == "" && kr.Gateway != "" {
		kr.Name = kr.Gateway
	}

	env := os.Environ()
	// XDS and CA servers are using system certificates ( recommended ).
	// If using a private CA - add it's root to the docker images, everything will be consistent
	// and simpler !
	env = append(env, "XDS_ROOT_CA=SYSTEM")
	env = append(env, "PILOT_CERT_PROVIDER=system")
	env = append(env, "CA_ROOT_CA=SYSTEM")
	env = append(env, "POD_NAMESPACE=" + kr.Namespace)

	if kr.ExtraEnv != nil {
		env = append(env, kr.ExtraEnv...)
	}

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

	kr.WhiteboxMode = os.Getenv("ISTIO_META_INTERCEPTION_MODE") == "NONE"
	if os.Getuid() != 0 {
		kr.WhiteboxMode = true
	}
	if kr.Gateway != "" {
		kr.WhiteboxMode = true
	}

	if !kr.WhiteboxMode { //&& kr.Gateway != "" {
		err := kr.runIptablesSetup(env)
		if err != nil {
			log.Println("iptables disabled ", err)
			kr.WhiteboxMode = true
		} else {
			log.Println("iptables done ")
		}
	} else {
		log.Println("No iptables")
	}

	// Currently broken in iptables - use whitebox interception, but still run it
	if !kr.WhiteboxMode {
		resolvConfForRoot()
		env = append(env, "ISTIO_META_DNS_CAPTURE=true")
		env = append(env, "DNS_PROXY_ADDR=localhost:53")
	}

	// MCP config
	// The following 2 are required for MeshCA.
	env = append(env, fmt.Sprintf("GKE_CLUSTER_URL=https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s",
		kr.ProjectId, kr.ClusterLocation, kr.ClusterName))
	env = append(env, fmt.Sprintf("GCP_METADATA=%s|%s|%s|%s",
		kr.ProjectId, kr.ProjectNumber, kr.ClusterName, kr.ClusterLocation ))

	env = append(env, "XDS_ADDR=" + kr.XDSAddr)
	//env = append(env, "CA_ROOT_CA=/etc/ssl/certs/ca-certificates.crt")
	//env = append(env, "XDS_ROOT_CA=/etc/ssl/certs/ca-certificates.crt")
	env = append(env, "JWT_POLICY=third-party-jwt")

	env = append(env, "TRUST_DOMAIN=" + kr.TrustDomain)

	if kr.MCPAddr != "" {
		env = append(env, "CA_ADDR=meshca.googleapis.com:443")
		env = append(env, "XDS_AUTH_PROVIDER=gcp")
		env = append(env, "ISTIO_META_CLOUDRUN_ADDR=" + kr.MCPAddr)
		// This is required for MCP - does not work for OSS primary cluster.
		// Will be used to set a clusterid metadata, which will locate the remote cluster id
		env = append(env, fmt.Sprintf("ISTIO_META_CLUSTER_ID=cn-%s-%s-%s",
			kr.ProjectId, kr.ClusterLocation, kr.ClusterName))
	}

	if kr.WhiteboxMode {
		env = append(env, "ISTIO_META_INTERCEPTION_MODE=NONE")
		env = append(env, "HTTP_PROXY_PORT=15007")
	}

	// WIP: automate getting the CR addr (or have Thetis handle it)
	// For example by reading a configmap in cluster
	//--set-env-vars="ISTIO_META_CLOUDRUN_ADDR=asm-stg-asm-cr-asm-managed-rapid-c-2o26nc3aha-uc.a.run.app:443" \

	// If set, let istiod generate bootstrap
	bootstrapIstiod := os.Getenv("BOOTSTRAP_XDS_AGENT")
	if bootstrapIstiod == "" {
		if _, err := os.Stat(prefix + "/var/lib/istio/envoy/hbone_tmpl.json"); os.IsNotExist(err) {
			os.MkdirAll(prefix + "/var/lib/istio/envoy/", 0755)
			err = ioutil.WriteFile(prefix + "/var/lib/istio/envoy/envoy_bootstrap_tmpl.json",
				[]byte(EnvoyBootstrapTmpl), 0755)
			if err != nil {
				panic(err)
			}
		} else {
			custom, err := ioutil.ReadFile(prefix + "/var/lib/istio/envoy/hbone_tmpl.json")
			if err != nil {
				panic(err) // no point continuing
			}
			err = ioutil.WriteFile(prefix + "/var/lib/istio/envoy/envoy_bootstrap_tmpl.json",
				[]byte(custom), 0755)
			if err != nil {
				panic(err)
			}
		}
	}

	// Environment detection: if the docker image or VM does not include an Envoy use the 'grpc agent' mode,
	// i.e. only get certificate.
	if _, err := os.Stat("/usr/local/bin/envoy"); os.IsNotExist(err) {
		env = append(env, "DISABLE_ENVOY=true")
	}

	// Generate grpc bootstrap - no harm, low cost
	if os.Getenv("GRPC_XDS_BOOTSTRAP") == "" {
		env = append(env, "GRPC_XDS_BOOTSTRAP=./var/run/grpc_bootstrap.json")
	}
	cmd := kr.agentCommand()
	var stdout io.ReadCloser
	if os.Getuid() == 0 {
		os.MkdirAll("/etc/istio/proxy", 777)
		os.Chown("/etc/istio/proxy", 1337, 1337)

		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Credential = &syscall.Credential{
			Uid: 0,
			Gid: 1337,
		}
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
		env = append(env, "ISTIO_META_UNPRIVILEGED_POD=true")
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
		kr.agentCmd = cmd
		if stdout != nil {
			go func() {
				io.Copy(os.Stdout, stdout)
			}()
		}
		err = cmd.Wait()
		if err != nil {
			log.Println("Wait err ", err)
		}
		kr.Exit(0)
	}()

	// TODO: wait for agent to be ready
	return nil
}

func (kr *KRun) Exit(code int) {
	if kr.agentCmd != nil && kr.agentCmd.Process != nil {
		kr.agentCmd.Process.Kill()
	}
	if kr.appCmd != nil && kr.appCmd.Process != nil {
		kr.appCmd.Process.Kill()
	}
	os.Exit(code)
}

func (kr *KRun) initLabelsFile() {
	if kr.Gateway != "" {
		ioutil.WriteFile("/etc/istio/pod/labels", []byte(
				`version="v1"
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

func (kr *KRun) runIptablesSetup(env []string) error {
	// TODO: make the args the default !
	// pilot-agent istio-iptables -p 15001 -u 1337 -m REDIRECT -i '*' -b "" -x "" -- crash

	//pilot-agent istio-iptables -p 15001 -u 1337 -m REDIRECT -i '10.8.4.0/24' -b "" -x ""
	cmd := exec.Command("/usr/local/bin/pilot-agent",
		"istio-iptables",
		"-p", "15001", // outbound capture port
		//"-z", "15006", - no inbound interception
		"-u", "1337",
		"-m", "REDIRECT",
		"-i",  "10.0.0.0/8", // all outbound captured
		"-b", "", // disable all inbound redirection
		// "-d", "15090,15021,15020", // exclude specific ports
		"-x", "")
	cmd.Env = env
	cmd.Dir = "/"
	so := &bytes.Buffer{}
	se := &bytes.Buffer{}
	cmd.Stdout = so
	cmd.Stderr = se
	err := cmd.Start()
	if err != nil {
		log.Println("Error starting iptables", err, so.String(), "stderr:", se.String())
		return err
	} else {
		err = cmd.Wait()
		if err != nil {
			log.Println("Error starting iptables", err, so.String(), "stderr:", se.String())
			return err
		}
	}
	// TODO: make the stdout/stderr available in a debug endpoint
	return nil
}

// TODO: lookup istiod service and endpoints ( instead of using an ILB or external name)
//
