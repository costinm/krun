package k8s

import _ "embed"

//go:embed envoy_bootstrap_tmpl.json
var EnvoyBootstrapTmpl string
