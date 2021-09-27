# GCP metadata

If running on GCP VM or CloudRun we can autoconfigure and get ID tokens using the metadata server.

curl -H "Metadata-Flavor: Google" \
'http://metadata/computeMetadata/v1/instance/service-accounts/default/identity?audience=https://www.example.com'

http://metadata.google.internal/computeMetadata/v1/project/

- project-id
- numeric-project-id

http://metadata.google.internal/computeMetadata/v1/instance/

- id - long UUID
- region - projects/NUM/regions/us-central1
- zone - projects/NUM/zones/us-central1-1
- service-accounts
    - email - 601426346923-compute@developer.gserviceaccount.com
    - ?audience=... - ID token - includes the same email, "iss": "https://accounts.google.com", sub and azp
    - token - access token for GCP

# CloudRun env

In cloudrun additional env variables are made available and can be used for autoconfiguration.

- K_REVISION=fortio-00049-liw
- PORT
- K_SERVICE=fortio
- S2A_ACCESS_TOKEN ?
- K_CONFIGURATION=fortio

PORT defaults to 8080 - the applications are expected to use the PORT as listen address for their HTTP1/2. When krun
forks the app, it will override the PORT and set it to 8080. ( TODO: allow customization ). When starting the app, PORT
must be set to 15009, which is the 'hbone' h2c tunnel port.

# Istio VM setup

Environment variables used to configure istio VM startup, used by the shell script. More recent docs use
WORKLOAD_NAMESPACE, etc - since this project may also use TrafficDirector it is better to avoid the ISTIO_ prefix.

Useful:

- ISTIO_SERVICE=myservice - used as canonical service name
- ISTIO_NAMESPACE=default -> POD_NAMESPACE
- ISTIO_AGENT_FLAGS="--proxyLogLevel debug"
- ISTIO_CUSTOM_IP_TABLES=false - do not set iptables, even if running as root
- CA_ADDR=istiod.istio-system.svc:32018

Not used in KNative/CloudRun mode, with HBONE (single port):

- ISTIO_INBOUND_PORTS=
- ISTIO_INBOUND_EXCLUDE_PORTS=

Not used:

- ISTIO_SERVICE_CIDR - original dst works now
- ISTIO_INBOUND_INTERCEPTION_MODE=REDIRECT - only one supported
- All TPROXY related
- ISTIO_SVC_IP=
- ISTIO_PILOT_PORT=15005, ISTIO_CP_AUTH=MUTUAL_TLS - replaced by XDS_ADDR
- ENVOY_PORT, ENVOY_USER - no tests or support for other values
- ISTIO_LOG_DIR - stdout works now, using pty
- ISTIO_BIN_BASE=/usr/local/bin - no support for other values (except test mode)
- ISTIO_CFG=/var/lib/istio - no support for other values
- PROV_CERT=/var/run/secrets/istio
- OUTPUT_CERTS=/var/run/secrets/istio - using metadata server, on by default

