package sshd

import (
	"io/ioutil"
	"os"
	"strconv"
)

// Helpers around sshd, using exec.
// Will be used if /usr/bin/sshd is added to the docker image.
// WIP: the code is using a built-in sshd, but it may be easier to use the official sshd if present and reduce code size.
// The 'special' thing about the built-in is that it's using SSH certificates - but they can also be created as
// secrets or provisioned the same way as Istio certs, in files by the agent.

var SshdConfig = `
Port 15022
AddressFamily any
ListenAddress 0.0.0.0
ListenAddress ::
Protocol 2
LogLevel INFO

HostKey /tmp/sshd/ssh_host_ecdsa_key

PermitRootLogin yes

AuthorizedKeysFile	/tmp/sshd/authorized_keys

PasswordAuthentication no
PermitUserEnvironment yes

AcceptEnv LANG LC_*
PrintMotd no

Subsystem	sftp	/usr/lib/openssh/sftp-server
`

type SSHDConfig struct {
	Port int
}

// StartSSHD will start /usr/bin/sshd, with the current UID.
// This works for non-root users as well.
//
//
func StartSSHD(cfg *SSHDConfig) {

	// /usr/sbin/sshd -p 15022 -e -D -h ~/.ssh/ec-key.pem
	// -f config
	// -c host_cert_file
	// -d debug - only one connection processed
	// -e debug to stderr
	// -h or -o HostKey
	// -p or -o Port
	//
	if cfg == nil {
		cfg = &SSHDConfig{}
	}
	if cfg.Port == 0 {
		cfg.Port = 15022
	}

	os.Mkdir("/tmp/sshd", 0700)

	// -q  - quiet
	// -f - output file
	// -N "" - no passphrase
	// -t ecdsa - keytype
	os.StartProcess("/usr/bin/ssh-keygen",
		[]string{
			"-q",
			"-f",
			"/tmp/sshd/ssh_host_ecdsa_key",
			"-N",
			"",
			"-t",
			"ecdsa",
		},
		&os.ProcAttr{
		})

	ioutil.WriteFile("/tmp/sshd/sshd_confing", []byte(SshdConfig), 0700)

	os.StartProcess("/usr/sbin/sshd",
		[]string{"-f", "/tmp/sshd/sshd_config",
			"-e",
			"-D",
			"-p", strconv.Itoa(cfg.Port),
		}, nil)

}
