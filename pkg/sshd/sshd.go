// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sshd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

// Helpers around sshd, using exec.
// Will be used if /usr/bin/sshd is added to the docker image.
// WIP: the code is using a built-in sshd, but it may be easier to use the official sshd if present and reduce code size.
// The 'special' thing about the built-in is that it's using SSH certificates - but they can also be created as
// secrets or provisioned the same way as Istio certs, in files by the agent.

var sshdConfig = `
Port 15022
AddressFamily any
ListenAddress 0.0.0.0
ListenAddress ::
Protocol 2
LogLevel INFO

HostKey %s/id_ecdsa

PermitRootLogin yes

AuthorizedKeysFile	%s/authorized_keys

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
func StartSSHD(cfg *SSHDConfig) error {

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

	pwd, _ := os.Getwd()
	sshd := pwd + "/var/run/secrets"
	os.Mkdir("./var/run/secrets", 0755)
	os.Mkdir("./var/run/secrets/sshd", 0700)

	if _, err := os.Stat(sshd + "/id_ecdsa"); os.IsNotExist(err) {
		// -q  - quiet
		// -f - output file
		// -N "" - no passphrase
		// -t ecdsa - keytype
		os.StartProcess("/usr/bin/ssh-keygen",
			[]string{
				"-q",
				"-f",
				sshd + "/id_ecdsa",
				"-N",
				"",
				"-t",
				"ecdsa",
			},
			&os.ProcAttr{})
	}
	if _, err := os.Stat(sshd + "/sshd_config"); os.IsNotExist(err) {
		ioutil.WriteFile(sshd+"/sshd_confing", []byte(fmt.Sprintf(sshdConfig, sshd, sshd)), 0700)
	}

	_, err := os.StartProcess("/usr/sbin/sshd",
		[]string{"-f", sshd + "/sshd_config",
			"-e",
			"-D",
			"-p", strconv.Itoa(cfg.Port),
		}, nil)

	return err
}
