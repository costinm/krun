package mesh

import _ "embed"

// TODO: this is only for testing hbone in envoy vs go. Will use the bootstrap file from docker image after testing is done.
// Also EnvoyFilters or server-side generated bootstrap could be used as well.

//go:embed envoy_bootstrap_tmpl.json
var EnvoyBootstrapTmpl string
