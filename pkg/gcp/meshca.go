package gcp

import _ "embed"

// This contains the long-lived MeshCA key. Used for migrations, if the
// key is not configured in cluster.

// Since the caller may connect to clusters with either Citadel or MeshCA, and
// communicate with workloads in different clusters, we need to configure both.

//go:embed meshca.pem
var MeshCA string

